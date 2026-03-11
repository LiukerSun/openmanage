package preferences

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type UserPreferences struct {
	Username     string            `json:"username"`
	Style        string            `json:"style"`
	Tools        string            `json:"tools"`
	ExtraContext string            `json:"extraContext"`
	Variables    map[string]string `json:"variables,omitempty"`
}

// Resolved returns a copy with {{VAR}} placeholders replaced by actual values.
func (p *UserPreferences) Resolved() *UserPreferences {
	if len(p.Variables) == 0 {
		return p
	}
	out := *p
	for k, v := range p.Variables {
		placeholder := "{{" + k + "}}"
		out.Tools = strings.ReplaceAll(out.Tools, placeholder, v)
		out.ExtraContext = strings.ReplaceAll(out.ExtraContext, placeholder, v)
	}
	return &out
}

// Masked returns a copy with variable values masked for frontend display.
func (p *UserPreferences) Masked() *UserPreferences {
	out := *p
	if len(p.Variables) > 0 {
		out.Variables = make(map[string]string, len(p.Variables))
		for k, v := range p.Variables {
			out.Variables[k] = maskValue(v)
		}
	}
	return &out
}

func maskValue(v string) string {
	if len(v) <= 4 {
		return "****"
	}
	return v[:4] + "****"
}

type Store struct {
	filePath string
	mu       sync.RWMutex
}

func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(homeDir, ".openmanage")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Store{
		filePath: filepath.Join(dir, "preferences.json"),
	}, nil
}

func (s *Store) Get() (*UserPreferences, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &UserPreferences{}, nil
		}
		return nil, err
	}

	var prefs UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return &UserPreferences{}, nil
	}
	return &prefs, nil
}

// Save stores preferences. For variables, if a value ends with "****"
// it means the frontend didn't change it — preserve the original.
func (s *Store) Save(prefs *UserPreferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load existing to preserve unchanged masked variables
	var existing UserPreferences
	if data, err := os.ReadFile(s.filePath); err == nil {
		json.Unmarshal(data, &existing)
	}

	if prefs.Variables != nil && existing.Variables != nil {
		for k, v := range prefs.Variables {
			if strings.HasSuffix(v, "****") {
				if orig, ok := existing.Variables[k]; ok {
					prefs.Variables[k] = orig
				}
			}
		}
	}

	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0600)
}
