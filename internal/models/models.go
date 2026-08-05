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

// GroupMember is a registered member of a group, joined with the ids needed to
// message them directly.
type GroupMember struct {
	UserID       int64
	TelegramID   int64
	Name         string
	Role         UserRole
	SessionsOwed int
}

type EventResponseType string

const (
	ResponsePresent EventResponseType = "present"
	ResponseAbsent  EventResponseType = "absent"
)

type Event struct {
	ID             int64     `db:"id"`
	GroupID        int64     `db:"group_id"`
	CreatedBy      int64     `db:"created_by"`
	Month          string    `db:"month"`
	SessionDate    string    `db:"session_date"`
	Capacity       int       `db:"capacity"`
	GroupMessageID int       `db:"group_message_id"`
	IsClosed       bool      `db:"is_closed"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type EventResponse struct {
	ID        int64             `db:"id"`
	EventID   int64             `db:"event_id"`
	UserID    int64             `db:"user_id"`
	Response  EventResponseType `db:"response"`
	CreatedAt time.Time         `db:"created_at"`
	UpdatedAt time.Time         `db:"updated_at"`
}

// EventGuest is someone with no account who an admin added to a single session.
// Guests are name-only: they carry no role, no rate and no debt.
type EventGuest struct {
	ID      int64  `db:"id"`
	EventID int64  `db:"event_id"`
	Name    string `db:"name"`
	AddedBy int64  `db:"added_by"`
	// IsFree guests cost nobody anything. Only admins can create them.
	IsFree    bool      `db:"is_free"`
	CreatedAt time.Time `db:"created_at"`
}

// EventAnswer is one person's answer joined with the name they registered with.
type EventAnswer struct {
	UserID   int64
	Name     string
	Response EventResponseType
}

type UserState struct {
	UserID      int64
	State       string
	TempData    map[string]interface{}
	LastUpdated time.Time
}
