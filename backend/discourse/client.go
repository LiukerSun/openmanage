package discourse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"openmanage/backend/model"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) doGet(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Api-Username", "system")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("discourse API %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) GetUserActivity(username string) (*model.ForumActivity, error) {
	// Fetch recent actions: 1=like given, 2=like received, 4=new topic, 5=reply
	actionsData, err := c.doGet(fmt.Sprintf("/user_actions.json?username=%s&filter=1,2,4,5&limit=50", username))
	if err != nil {
		return nil, fmt.Errorf("fetch actions: %w", err)
	}

	var actionsResp struct {
		Actions []struct {
			ActionType int    `json:"action_type"`
			Title      string `json:"title"`
			TopicID    int    `json:"topic_id"`
			PostNumber int    `json:"post_number"`
			CreatedAt  string `json:"created_at"`
			Excerpt    string `json:"excerpt"`
			Slug       string `json:"slug"`
		} `json:"user_actions"`
	}
	if err := json.Unmarshal(actionsData, &actionsResp); err != nil {
		return nil, fmt.Errorf("parse actions: %w", err)
	}

	// Compute stats from actions instead of relying on summary API cache
	var topicCount, postCount, likesGiven, likesRecv int
	days := make(map[string]bool)
	actions := make([]model.ForumAction, 0, len(actionsResp.Actions))

	for _, a := range actionsResp.Actions {
		// Count by day for daysVisited
		if len(a.CreatedAt) >= 10 {
			days[a.CreatedAt[:10]] = true
		}

		switch a.ActionType {
		case 1: // like given
			likesGiven++
		case 2: // like received
			likesRecv++
		case 4: // new topic
			topicCount++
		case 5: // reply
			postCount++
		}

		// Only show topics and replies in the action list
		if a.ActionType == 4 || a.ActionType == 5 {
			typ := "topic"
			if a.ActionType == 5 {
				typ = "reply"
			}
			actions = append(actions, model.ForumAction{
				Type:      typ,
				Title:     a.Title,
				TopicID:   a.TopicID,
				PostNum:   a.PostNumber,
				CreatedAt: a.CreatedAt,
				Excerpt:   a.Excerpt,
				Slug:      a.Slug,
			})
		}
	}

	return &model.ForumActivity{
		Username:    username,
		TopicCount:  topicCount,
		PostCount:   postCount,
		LikesGiven:  likesGiven,
		LikesRecv:   likesRecv,
		DaysVisited: len(days),
		Actions:     actions,
	}, nil
}

// CreateUser creates a Discourse user via Admin API.
// If the user already exists, it returns nil (idempotent).
func (c *Client) CreateUser(username, name, email, password string) error {
	form := url.Values{
		"name":     {name},
		"email":    {email},
		"password": {password},
		"username": {username},
		"active":   {"true"},
		"approved": {"true"},
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/users.json", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Api-Username", "system")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("discourse create user request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	// 200 = success, check response for "success" field
	if resp.StatusCode == 200 {
		var result struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &result); err == nil {
			if result.Success {
				return nil
			}
			// "Username must be unique" etc. — treat as already exists
			if strings.Contains(result.Message, "already") ||
				strings.Contains(strings.ToLower(result.Message), "username") {
				return nil
			}
		}
		return nil
	}

	// 422 usually means validation error (user exists, etc.)
	if resp.StatusCode == 422 {
		return nil
	}

	return fmt.Errorf("discourse create user returned %d: %s", resp.StatusCode, string(body))
}
