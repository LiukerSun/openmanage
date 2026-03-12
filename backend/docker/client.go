package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"openmanage/backend/model"
)

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

// IsOpenClaw checks if a container is an OpenClaw container.
// Priority: label openmanage.openclaw=true > image contains "openclaw" > mount path contains .openclaw
func IsOpenClaw(ct types.Container) bool {
	if v, ok := ct.Labels["openmanage.openclaw"]; ok && v == "true" {
		return true
	}
	if strings.Contains(ct.Image, "openclaw") {
		return true
	}
	for _, m := range ct.Mounts {
		if strings.Contains(m.Destination, ".openclaw") {
			return true
		}
	}
	return false
}

func (c *Client) ListOpenClawContainers(ctx context.Context) ([]types.Container, error) {
	all, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	var result []types.Container
	for _, ct := range all {
		if IsOpenClaw(ct) {
			result = append(result, ct)
		}
	}
	return result, nil
}

func (c *Client) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
	return c.cli.ContainerInspect(ctx, id)
}

// GetContainerEnv returns a map of environment variables set on the container.
func (c *Client) GetContainerEnv(ctx context.Context, id string) (map[string]string, error) {
	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}
	env := make(map[string]string)
	for _, e := range info.Config.Env {
		if idx := strings.Index(e, "="); idx >= 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	return env, nil
}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (c *Client) StopContainer(ctx context.Context, id string) error {
	return c.cli.ContainerStop(ctx, id, container.StopOptions{})
}

func (c *Client) RestartContainer(ctx context.Context, id string) error {
	return c.cli.ContainerRestart(ctx, id, container.StopOptions{})
}

func (c *Client) ContainerLogs(ctx context.Context, id string, tail string, follow bool) (interface{ Read([]byte) (int, error); Close() error }, error) {
	return c.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
		Timestamps: true,
	})
}

// GetMountSource returns the host path for a given container mount destination.
func (c *Client) GetMountSource(ctx context.Context, id string, dest string) (string, error) {
	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}
	for _, m := range info.Mounts {
		if strings.HasPrefix(dest, string(m.Destination)) {
			return string(m.Source), nil
		}
	}
	return "", nil
}

// ListContainersByLabel lists containers with a specific label.
func (c *Client) ListContainersByLabel(ctx context.Context, label string) ([]types.Container, error) {
	f := filters.NewArgs()
	f.Add("label", label)
	return c.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
}

// CreateContainer creates and starts a new OpenClaw container.
func (c *Client) CreateContainer(ctx context.Context, name, image, dataPath string, port int, env map[string]string) (string, error) {
	envList := make([]string, 0, len(env)+1)
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	// Ensure container processes use the correct HOME directory
	envList = append(envList, "HOME=/home/node")

	portStr := fmt.Sprintf("%d/tcp", 3000)
	exposedPorts := nat.PortSet{nat.Port(portStr): struct{}{}}
	portBindings := nat.PortMap{
		nat.Port(portStr): []nat.PortBinding{{HostPort: fmt.Sprintf("%d", port)}},
	}

	config := &container.Config{
		Image:        image,
		Labels:       map[string]string{"openmanage.openclaw": "true"},
		Env:          envList,
		ExposedPorts: exposedPorts,
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: dataPath,
				Target: "/home/node/.openclaw",
			},
		},
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyUnlessStopped,
		},
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, nil, name)
	if err != nil {
		return "", err
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Start failed (e.g. port conflict) — remove the created container to free the name
		c.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", err
	}

	return resp.ID, nil
}

// ExecCommand runs a command inside a container and returns the combined output.
func (c *Client) ExecCommand(ctx context.Context, containerID string, cmd []string) (string, error) {
	execCfg := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}
	execID, err := c.cli.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		return "", fmt.Errorf("exec create: %w", err)
	}

	resp, err := c.cli.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("exec attach: %w", err)
	}
	defer resp.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Reader)
	if err != nil {
		return "", fmt.Errorf("exec read: %w", err)
	}

	// Strip Docker multiplexed stream headers (8-byte prefix per frame)
	raw := buf.Bytes()
	var clean strings.Builder
	for len(raw) >= 8 {
		size := int(raw[4])<<24 | int(raw[5])<<16 | int(raw[6])<<8 | int(raw[7])
		raw = raw[8:]
		if size > len(raw) {
			size = len(raw)
		}
		clean.Write(raw[:size])
		raw = raw[size:]
	}
	if clean.Len() > 0 {
		return clean.String(), nil
	}
	return buf.String(), nil
}

// RemoveContainer stops (if running) and removes a container.
func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	return c.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
}

// RenameContainer renames a container.
func (c *Client) RenameContainer(ctx context.Context, id string, newName string) error {
	return c.cli.ContainerRename(ctx, id, newName)
}

// ContainerStats returns a one-shot stats snapshot for a container.
func (c *Client) ContainerStats(ctx context.Context, id string) (*model.ContainerStats, error) {
	resp, err := c.cli.ContainerStatsOneShot(ctx, id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs     uint64 `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
		PidsStats struct {
			Current uint64 `json:"current"`
		} `json:"pids_stats"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	// Calculate CPU percentage
	cpuPercent := 0.0
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage)
	if sysDelta > 0 && cpuDelta >= 0 {
		cpus := float64(raw.CPUStats.OnlineCPUs)
		if cpus == 0 {
			cpus = 1
		}
		cpuPercent = (cpuDelta / sysDelta) * cpus * 100.0
	}

	// Calculate memory percentage
	memPercent := 0.0
	if raw.MemoryStats.Limit > 0 {
		memPercent = float64(raw.MemoryStats.Usage) / float64(raw.MemoryStats.Limit) * 100.0
	}

	// Sum network stats
	var netRx, netTx uint64
	for _, n := range raw.Networks {
		netRx += n.RxBytes
		netTx += n.TxBytes
	}

	return &model.ContainerStats{
		CPUPercent: cpuPercent,
		MemUsage:   raw.MemoryStats.Usage,
		MemLimit:   raw.MemoryStats.Limit,
		MemPercent: memPercent,
		NetRx:      netRx,
		NetTx:      netTx,
		PIDs:       raw.PidsStats.Current,
	}, nil
}
