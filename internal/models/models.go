package models

import "time"

type UserRole string

const (
	RoleAdmin     UserRole = "admin"
	RoleStudent   UserRole = "student"
	RoleAdult     UserRole = "adult"
	RoleHalfAdult UserRole = "half_adult"
)

type User struct {
	ID         int64     `db:"id"`
	TelegramID int64     `db:"telegram_id"`
	Username   string    `db:"username"`
	FirstName  string    `db:"first_name"`
	LastName   string    `db:"last_name"`
	IsBot      bool      `db:"is_bot"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type Group struct {
	ID             int64     `db:"id"`
	TelegramChatID int64     `db:"telegram_chat_id"`
	Title          string    `db:"title"`
	Type           string    `db:"type"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type UserGroup struct {
	ID           int64     `db:"id"`
	UserID       int64     `db:"user_id"`
	GroupID      int64     `db:"group_id"`
	Role         UserRole  `db:"role"`
	Name         string    `db:"name"`
	SessionsOwed int       `db:"sessions_owed"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type Rate struct {
	ID             int64     `db:"id"`
	GroupID        int64     `db:"group_id"`
	Role           UserRole  `db:"role"`
	RatePerSession float64   `db:"rate_per_session"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type AttendanceRecord struct {
	ID         int64      `db:"id"`
	GroupID    int64      `db:"group_id"`
	AdminID    int64      `db:"admin_id"`
	UserIDs    []int64    `db:"user_ids"`
	CreatedAt  time.Time  `db:"created_at"`
	RevertedAt *time.Time `db:"reverted_at"`
	IsReverted bool       `db:"is_reverted"`
}

type UserState struct {
	UserID      int64
	State       string
	TempData    map[string]interface{}
	LastUpdated time.Time
}
