package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// User represents a stored user credential
type User struct {
	Username string `json:"username"`
	Password string `json:"password"` // bcrypt hash
}

// Store manages user credentials storage
type Store struct {
	filePath string
	mu       sync.RWMutex
}

// NewStore creates a new credential store
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
		filePath: filepath.Join(dir, "auth.json"),
	}, nil
}

// Get returns the stored user, creating default if not exists
func (s *Store) Get() (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default user: admin / admin123
			return s.createDefault()
		}
		return nil, err
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) createDefault() (*User, error) {
	s.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check after acquiring write lock
	data, err := os.ReadFile(s.filePath)
	if err == nil {
		var user User
		json.Unmarshal(data, &user)
		return &user, nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), 12)
	if err != nil {
		return nil, err
	}

	user := &User{
		Username: "admin",
		Password: string(hash),
	}

	if err := s.save(user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdatePassword updates the user's password
func (s *Store) UpdatePassword(oldPassword, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return ErrInvalidPassword
	}

	if len(newPassword) < 6 {
		return ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	user.Password = string(hash)
	return s.save(&user)
}

func (s *Store) save(user *User) error {
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0600)
}

// Verify checks if the username and password are valid
func (s *Store) Verify(username, password string) bool {
	user, err := s.Get()
	if err != nil {
		return false
	}
	if username != user.Username {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil
}

var (
	ErrInvalidPassword  = &AuthError{Message: "invalid password"}
	ErrPasswordTooShort = &AuthError{Message: "password must be at least 6 characters"}
)

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
