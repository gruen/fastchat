package db

import "time"

type Session struct {
	ID        string
	Title     string
	Provider  string
	Model     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Archived  bool
}

type Message struct {
	ID        int64
	SessionID string
	Role      string // "user", "assistant", "system"
	Content   string
	CreatedAt time.Time
	Tokens    int
}
