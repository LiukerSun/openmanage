package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"openmanage/backend/docker"
)

type Client struct {
	Docker      *docker.Client
	MountPrefix string
}

type ContainerConfig struct {
	Port         int
	Token        string
	NeedsRestart bool // true if openclaw.json was patched and container needs restart
}

func (c *Client) configPath(ctx context.Context, containerID string) (string, error) {
	src, err := c.Docker.GetMountSource(ctx, containerID, "/home/node/.openclaw")
	if err != nil {
		return "", fmt.Errorf("get mount source: %w", err)
	}
	if src == "" {
		return "", fmt.Errorf("no .openclaw mount found")
	}
	return filepath.Join(c.MountPrefix+src, "openclaw.json"), nil
}

// GetConfig reads the gateway port and auth token from the container's environment
// variables (OPENCLAW_GATEWAY_PORT, OPENCLAW_GATEWAY_TOKEN), and ensures the
// responses endpoint is enabled in openclaw.json.
func (c *Client) GetConfig(ctx context.Context, containerID string) (*ContainerConfig, error) {
	// Read port and token from Docker env vars (the gateway uses these, not openclaw.json)
	env, err := c.Docker.GetContainerEnv(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("get container env: %w", err)
	}

	port := 18789
	if p, err := strconv.Atoi(env["OPENCLAW_GATEWAY_PORT"]); err == nil && p > 0 {
		port = p
	}
	token := env["OPENCLAW_GATEWAY_TOKEN"]

	// Auto-patch openclaw.json: ensure http.endpoints.responses is enabled
	needsPatch := false
	configPath, cfgErr := c.configPath(ctx, containerID)
	if cfgErr == nil {
		data, readErr := os.ReadFile(configPath)
		if readErr == nil {
			var full map[string]interface{}
			if json.Unmarshal(data, &full) == nil {
				gw, _ := full["gateway"].(map[string]interface{})
				if gw != nil {
					httpCfg, _ := gw["http"].(map[string]interface{})
					if httpCfg == nil {
						httpCfg = map[string]interface{}{}
						needsPatch = true
					}
					endpoints, _ := httpCfg["endpoints"].(map[string]interface{})
					if endpoints == nil {
						endpoints = map[string]interface{}{}
						needsPatch = true
					}
					resp, _ := endpoints["responses"].(map[string]interface{})
					if resp == nil || resp["enabled"] != true {
						endpoints["responses"] = map[string]interface{}{"enabled": true}
						needsPatch = true
					}
					if needsPatch {
						httpCfg["endpoints"] = endpoints
						gw["http"] = httpCfg
						full["gateway"] = gw
						patched, err := json.MarshalIndent(full, "", "  ")
						if err == nil {
							os.WriteFile(configPath, patched, 0644)
						}
					}
				}
			}
		}
	}

	return &ContainerConfig{
		Port:         port,
		Token:        token,
		NeedsRestart: needsPatch,
	}, nil
}

// SendMessage sends a message to the agent running in the container via docker exec curl.
// It returns the agent's reply text.
func (c *Client) SendMessage(ctx context.Context, containerID string, cfg *ContainerConfig, message string) (string, error) {
	payload := fmt.Sprintf(`{"model":"openclaw","input":%s}`, jsonString(message))

	cmd := []string{
		"curl", "-sS", "--max-time", "120",
		fmt.Sprintf("http://127.0.0.1:%d/v1/responses", cfg.Port),
		"-H", fmt.Sprintf("Authorization: Bearer %s", cfg.Token),
		"-H", "Content-Type: application/json",
		"-H", "x-openclaw-agent-id: main",
		"-d", payload,
	}

	output, err := c.Docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return "", fmt.Errorf("exec curl: %w", err)
	}

	// Parse the OpenResponses API response to extract the output text
	reply := extractReply(output)
	return reply, nil
}

// extractReply parses the /v1/responses JSON and extracts the output_text or
// the first output message content.
func extractReply(raw string) string {
	raw = strings.TrimSpace(raw)

	// Try to extract output_text field first
	var resp struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		// Not valid JSON — return raw output
		return raw
	}

	if resp.Error != nil {
		return fmt.Sprintf("[Error] %s", resp.Error.Message)
	}

	if resp.OutputText != "" {
		return resp.OutputText
	}

	// Fallback: iterate output items
	for _, item := range resp.Output {
		if item.Type == "message" {
			for _, c := range item.Content {
				if c.Type == "output_text" && c.Text != "" {
					return c.Text
				}
			}
		}
	}

	return raw
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// CronJob is the flattened representation we return to the frontend.
type CronJob struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Prompt   string `json:"prompt"`
	Enabled  bool   `json:"enabled"`
	LastRun  string `json:"lastRun,omitempty"`
	NextRun  string `json:"nextRun,omitempty"`
}

// rawCronJob matches the actual JSON structure from `openclaw cron list --json`.
type rawCronJob struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Schedule struct {
		Kind string `json:"kind"`
		Expr string `json:"expr"`
	} `json:"schedule"`
	Payload struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
	} `json:"payload"`
	State struct {
		NextRunAtMs int64  `json:"nextRunAtMs"`
		LastRunAtMs int64  `json:"lastRunAtMs"`
	} `json:"state"`
}

// HeartbeatConfig represents the heartbeat section in openclaw.json.
type HeartbeatConfig struct {
	Every string `json:"every"`
	Mode  string `json:"mode,omitempty"`
}

// ListCronJobs returns all cron jobs for the container.
func (c *Client) ListCronJobs(ctx context.Context, containerID string) ([]CronJob, error) {
	cmd := []string{"openclaw", "cron", "list", "--json"}
	output, err := c.Docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return nil, fmt.Errorf("exec openclaw cron list: %w", err)
	}
	output = strings.TrimSpace(output)
	if output == "" || output == "[]" || output == "null" {
		return []CronJob{}, nil
	}

	// Parse wrapper object { "jobs": [...] }
	var wrapper struct {
		Jobs []rawCronJob `json:"jobs"`
	}
	if err := json.Unmarshal([]byte(output), &wrapper); err != nil {
		// Try bare array
		var rawJobs []rawCronJob
		if err2 := json.Unmarshal([]byte(output), &rawJobs); err2 != nil {
			return nil, fmt.Errorf("parse cron list: %w (raw: %s)", err, output)
		}
		wrapper.Jobs = rawJobs
	}

	jobs := make([]CronJob, 0, len(wrapper.Jobs))
	for _, r := range wrapper.Jobs {
		j := CronJob{
			ID:      r.ID,
			Name:    r.Name,
			Enabled: r.Enabled,
			Schedule: r.Schedule.Expr,
			Prompt:  r.Payload.Message,
		}
		if r.State.LastRunAtMs > 0 {
			j.LastRun = time.UnixMilli(r.State.LastRunAtMs).Format("2006-01-02 15:04:05")
		}
		if r.State.NextRunAtMs > 0 {
			j.NextRun = time.UnixMilli(r.State.NextRunAtMs).Format("2006-01-02 15:04:05")
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// AddCronJob creates a new cron job in the container.
func (c *Client) AddCronJob(ctx context.Context, containerID, name, schedule, prompt string) (string, error) {
	if name == "" {
		name = "job-" + schedule
	}
	cmd := []string{"openclaw", "cron", "add", "--name", name, "--cron", schedule, "--message", prompt, "--session", "isolated", "--json"}
	output, err := c.Docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return "", fmt.Errorf("exec openclaw cron add: %w", err)
	}
	// Try to extract job ID from JSON response
	var result struct {
		ID string `json:"id"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(output)), &result) == nil && result.ID != "" {
		return result.ID, nil
	}
	return strings.TrimSpace(output), nil
}

// ToggleCronJob enables or disables a cron job.
func (c *Client) ToggleCronJob(ctx context.Context, containerID, jobID string, enable bool) error {
	action := "disable"
	if enable {
		action = "enable"
	}
	cmd := []string{"openclaw", "cron", action, jobID}
	_, err := c.Docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return fmt.Errorf("exec openclaw cron %s: %w", action, err)
	}
	return nil
}

// RunCronJob triggers immediate execution of a cron job.
func (c *Client) RunCronJob(ctx context.Context, containerID, jobID string) error {
	cmd := []string{"openclaw", "cron", "run", jobID}
	_, err := c.Docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return fmt.Errorf("exec openclaw cron run: %w", err)
	}
	return nil
}

// RemoveCronJob deletes a cron job.
func (c *Client) RemoveCronJob(ctx context.Context, containerID, jobID string) error {
	cmd := []string{"openclaw", "cron", "remove", jobID}
	_, err := c.Docker.ExecCommand(ctx, containerID, cmd)
	if err != nil {
		return fmt.Errorf("exec openclaw cron remove: %w", err)
	}
	return nil
}

// GetHeartbeat reads the heartbeat config from openclaw.json.
func (c *Client) GetHeartbeat(ctx context.Context, containerID string) (*HeartbeatConfig, error) {
	cfgPath, err := c.configPath(ctx, containerID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read openclaw.json: %w", err)
	}
	var full map[string]interface{}
	if err := json.Unmarshal(data, &full); err != nil {
		return nil, fmt.Errorf("parse openclaw.json: %w", err)
	}
	agents, _ := full["agents"].(map[string]interface{})
	if agents == nil {
		return &HeartbeatConfig{}, nil
	}
	defaults, _ := agents["defaults"].(map[string]interface{})
	if defaults == nil {
		return &HeartbeatConfig{}, nil
	}
	hb, _ := defaults["heartbeat"].(map[string]interface{})
	if hb == nil {
		return &HeartbeatConfig{}, nil
	}
	cfg := &HeartbeatConfig{}
	if v, ok := hb["every"].(string); ok {
		cfg.Every = v
	}
	if v, ok := hb["mode"].(string); ok {
		cfg.Mode = v
	}
	return cfg, nil
}

// SetHeartbeat updates the heartbeat config in openclaw.json.
// Pass empty `every` to disable heartbeat.
func (c *Client) SetHeartbeat(ctx context.Context, containerID string, every string) error {
	cfgPath, err := c.configPath(ctx, containerID)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read openclaw.json: %w", err)
	}
	var full map[string]interface{}
	if err := json.Unmarshal(data, &full); err != nil {
		return fmt.Errorf("parse openclaw.json: %w", err)
	}
	agents, _ := full["agents"].(map[string]interface{})
	if agents == nil {
		agents = map[string]interface{}{}
	}
	defaults, _ := agents["defaults"].(map[string]interface{})
	if defaults == nil {
		defaults = map[string]interface{}{}
	}
	if every == "" {
		delete(defaults, "heartbeat")
	} else {
		defaults["heartbeat"] = map[string]interface{}{
			"every": every,
		}
	}
	agents["defaults"] = defaults
	full["agents"] = agents
	patched, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, patched, 0644)
}
