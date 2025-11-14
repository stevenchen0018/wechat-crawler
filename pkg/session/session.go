package session

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// Session 会话数据
type Session struct {
	ID        string
	Data      map[string]interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Store 会话存储
type Store struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	timeout  time.Duration
}

// NewStore 创建会话存储
func NewStore(timeout time.Duration) *Store {
	store := &Store{
		sessions: make(map[string]*Session),
		timeout:  timeout,
	}
	
	// 启动清理过期会话的goroutine
	go store.cleanExpiredSessions()
	
	return store
}

// Create 创建新会话
func (s *Store) Create() (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}
	
	now := time.Now()
	session := &Session{
		ID:        id,
		Data:      make(map[string]interface{}),
		CreatedAt: now,
		ExpiresAt: now.Add(s.timeout),
	}
	
	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()
	
	return session, nil
}

// Get 获取会话
func (s *Store) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	session, exists := s.sessions[id]
	if !exists {
		return nil, false
	}
	
	// 检查是否过期
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}
	
	return session, true
}

// Update 更新会话
func (s *Store) Update(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if session, exists := s.sessions[id]; exists {
		session.ExpiresAt = time.Now().Add(s.timeout)
	}
}

// Delete 删除会话
func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.sessions, id)
}

// Set 设置会话数据
func (s *Session) Set(key string, value interface{}) {
	s.Data[key] = value
}

// Get 获取会话数据
func (s *Session) Get(key string) (interface{}, bool) {
	value, exists := s.Data[key]
	return value, exists
}

// GetString 获取字符串类型的会话数据
func (s *Session) GetString(key string) string {
	if value, exists := s.Data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// cleanExpiredSessions 清理过期会话
func (s *Store) cleanExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// generateSessionID 生成会话ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

