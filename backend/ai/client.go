package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL = "https://open.bigmodel.cn/api/coding/paas/v4"
	model   = "glm-4.7-flash"
)

var mdFiles = []string{
	"SOUL.md", "IDENTITY.md", "AGENTS.md", "BOOTSTRAP.md",
	"HEARTBEAT.md", "MEMORY.md", "TOOLS.md", "USER.md",
}

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type GenerateRequest struct {
	Name         string
	Description  string
	Username     string
	Style        string
	Tools        string
	ExtraContext string
}

// Generate calls the GLM API to produce all 8 config files in one shot.
// Returns a map of filename -> content.
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (map[string]string, error) {
	systemPrompt := `你是一个 OpenClaw Agent 配置文件生成器。用户会提供 Agent 的名称和描述，你需要根据这些信息生成 8 个 markdown 配置文件。

请严格以 JSON 格式输出，key 为文件名，value 为文件内容。不要输出任何其他内容。

需要生成的文件：
1. SOUL.md - Agent 的核心灵魂设定，包括性格、价值观、行为准则
2. IDENTITY.md - Agent 的身份信息，包括名称、角色定位、自我介绍
3. AGENTS.md - 可调用的子 Agent 列表及其能力描述
4. BOOTSTRAP.md - Agent 启动时的初始化指令和欢迎语
5. HEARTBEAT.md - 定期提醒事项，如检查任务进度、主动关怀用户
6. MEMORY.md - 初始记忆和需要长期记住的关键信息
7. TOOLS.md - Agent 可使用的工具说明和使用注意事项
8. USER.md - 目标用户画像和交互偏好

每个文件内容应该丰富、具体、有针对性，不要使用空泛的占位符。内容使用 Markdown 格式。

输出格式示例：
{"SOUL.md": "# Soul\n\n...", "IDENTITY.md": "# Identity\n\n...", ...}`

	userPrompt := fmt.Sprintf("Agent 名称：%s\nAgent 描述：%s", req.Name, req.Description)

	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.7,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	content := result.Choices[0].Message.Content
	jsonStr := extractJSON(content)

	var files map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &files); err != nil {
		return nil, fmt.Errorf("parse generated JSON: %w", err)
	}

	return files, nil
}

// extractJSON strips markdown code fences if present.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Try to find ```json ... ``` or ``` ... ```
	if idx := strings.Index(s, "```"); idx != -1 {
		s = s[idx+3:]
		// Skip optional language tag on the same line
		if nl := strings.Index(s, "\n"); nl != -1 {
			s = s[nl+1:]
		}
		if end := strings.LastIndex(s, "```"); end != -1 {
			s = s[:end]
		}
		return strings.TrimSpace(s)
	}
	return s
}

// GenerateStream calls the GLM API with stream=true and invokes onChunk
// with the cumulative character count as tokens arrive.
// Returns the final parsed file map.
func (c *Client) GenerateStream(ctx context.Context, req GenerateRequest, onChunk func(chars int)) (map[string]string, error) {
	systemPrompt := `你是一个 OpenClaw Agent 配置文件生成器。用户会提供 Agent 的名称和描述，你需要根据这些信息生成 8 个 markdown 配置文件。

请严格以 JSON 格式输出，key 为文件名，value 为文件内容。不要输出任何其他内容。

需要生成的文件：
1. SOUL.md - Agent 的核心灵魂设定，包括性格、价值观、行为准则
2. IDENTITY.md - Agent 的身份信息，包括名称、角色定位、自我介绍
3. AGENTS.md - 可调用的子 Agent 列表及其能力描述
4. BOOTSTRAP.md - Agent 启动时的初始化指令和欢迎语
5. HEARTBEAT.md - 定期提醒事项，如检查任务进度、主动关怀用户
6. MEMORY.md - 初始记忆和需要长期记住的关键信息
7. TOOLS.md - Agent 可使用的工具说明和使用注意事项
8. USER.md - 目标用户画像和交互偏好

每个文件内容应该丰富、具体、有针对性，不要使用空泛的占位符。内容使用 Markdown 格式。

输出格式示例：
{"SOUL.md": "# Soul\n\n...", "IDENTITY.md": "# Identity\n\n...", ...}`

	userPrompt := fmt.Sprintf("Agent 名称：%s\nAgent 描述：%s", req.Name, req.Description)
	if req.Username != "" {
		userPrompt += fmt.Sprintf("\n用户名称：%s", req.Username)
	}
	if req.Style != "" {
		userPrompt += fmt.Sprintf("\n偏好风格：%s", req.Style)
	}
	if req.Tools != "" {
		userPrompt += fmt.Sprintf("\n常用工具：%s", req.Tools)
	}
	if req.ExtraContext != "" {
		userPrompt += fmt.Sprintf("\n补充信息：%s", req.ExtraContext)
	}

	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.7,
		"stream":      true,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("api returned %d: %s", resp.StatusCode, string(respBody))
	}

	var fullContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			fullContent.WriteString(chunk.Choices[0].Delta.Content)
			if onChunk != nil {
				onChunk(fullContent.Len())
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	content := fullContent.String()
	if content == "" {
		return nil, fmt.Errorf("empty response from API")
	}

	jsonStr := extractJSON(content)
	var files map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &files); err != nil {
		return nil, fmt.Errorf("parse generated JSON: %w", err)
	}

	return files, nil
}
