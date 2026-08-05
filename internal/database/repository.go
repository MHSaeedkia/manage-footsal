package database

import (
	"database/sql"
	"fmt"

	"futsal-bot/internal/models"
)

// User operations
func (db *DB) GetOrCreateUser(telegramID int64, username, firstName, lastName string, isBot bool) (*models.User, error) {
	var user models.User

	err := db.QueryRow(`
		INSERT INTO users (telegram_id, username, first_name, last_name, is_bot)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (telegram_id) DO UPDATE
		SET username = EXCLUDED.username,
		    first_name = EXCLUDED.first_name,
		    last_name = EXCLUDED.last_name,
		    updated_at = CURRENT_TIMESTAMP
		RETURNING id, telegram_id, username, first_name, last_name, is_bot, created_at, updated_at
	`, telegramID, username, firstName, lastName, isBot).Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.IsBot, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	return &user, nil
}

func (db *DB) GetUserByTelegramID(telegramID int64) (*models.User, error) {
	var user models.User

	err := db.QueryRow(`
		SELECT id, telegram_id, username, first_name, last_name, is_bot, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`, telegramID).Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.IsBot, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Group operations
func (db *DB) GetOrCreateGroup(telegramChatID int64, title, chatType string) (*models.Group, error) {
	var group models.Group

	err := db.QueryRow(`
		INSERT INTO groups (telegram_chat_id, title, type)
		VALUES ($1, $2, $3)
		ON CONFLICT (telegram_chat_id) DO UPDATE
		SET title = EXCLUDED.title,
		    type = EXCLUDED.type,
		    updated_at = CURRENT_TIMESTAMP
		RETURNING id, telegram_chat_id, title, type, created_at, updated_at
	`, telegramChatID, title, chatType).Scan(
		&group.ID, &group.TelegramChatID, &group.Title, &group.Type,
		&group.CreatedAt, &group.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get or create group: %w", err)
	}

	return &group, nil
}

func (db *DB) GetGroupByTelegramChatID(telegramChatID int64) (*models.Group, error) {
	var group models.Group

	err := db.QueryRow(`
		SELECT id, telegram_chat_id, title, type, created_at, updated_at
		FROM groups
		WHERE telegram_chat_id = $1
	`, telegramChatID).Scan(
		&group.ID, &group.TelegramChatID, &group.Title, &group.Type,
		&group.CreatedAt, &group.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &group, nil
}

// UserGroup operations
func (db *DB) CreateOrUpdateUserGroup(userID, groupID int64, role models.UserRole, name string) error {
	_, err := db.Exec(`
		INSERT INTO user_groups (user_id, group_id, role, name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, group_id) DO UPDATE
		SET role = EXCLUDED.role,
		    name = EXCLUDED.name,
		    updated_at = CURRENT_TIMESTAMP
	`, userID, groupID, role, name)

	return err
}

func (db *DB) GetUserGroup(userID, groupID int64) (*models.UserGroup, error) {
	var ug models.UserGroup

	err := db.QueryRow(`
		SELECT id, user_id, group_id, role, name, sessions_owed, created_at, updated_at
		FROM user_groups
		WHERE user_id = $1 AND group_id = $2
	`, userID, groupID).Scan(
		&ug.ID, &ug.UserID, &ug.GroupID, &ug.Role, &ug.Name,
		&ug.SessionsOwed, &ug.CreatedAt, &ug.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &ug, nil
}

func (db *DB) GetUserGroupsByGroupID(groupID int64) ([]models.UserGroup, error) {
	rows, err := db.Query(`
		SELECT id, user_id, group_id, role, name, sessions_owed, created_at, updated_at
		FROM user_groups
		WHERE group_id = $1
		ORDER BY name
	`, groupID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userGroups []models.UserGroup
	for rows.Next() {
		var ug models.UserGroup
		err := rows.Scan(
			&ug.ID, &ug.UserID, &ug.GroupID, &ug.Role, &ug.Name,
			&ug.SessionsOwed, &ug.CreatedAt, &ug.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		userGroups = append(userGroups, ug)
	}

	return userGroups, nil
}

func (db *DB) IsUserMemberOfGroup(userID, groupID int64) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM user_groups WHERE user_id = $1 AND group_id = $2)
	`, userID, groupID).Scan(&exists)

	return exists, err
}

func (db *DB) GetUserGroups(userID int64) ([]int64, error) {
	rows, err := db.Query(`
		SELECT group_id FROM user_groups WHERE user_id = $1
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groupIDs []int64
	for rows.Next() {
		var groupID int64
		if err := rows.Scan(&groupID); err != nil {
			return nil, err
		}
		groupIDs = append(groupIDs, groupID)
	}

	return groupIDs, nil
}

func (db *DB) IsUserAdminInGroup(userID, groupID int64) (bool, error) {
	var role string
	err := db.QueryRow(`
		SELECT role FROM user_groups WHERE user_id = $1 AND group_id = $2
	`, userID, groupID).Scan(&role)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return role == string(models.RoleAdmin), nil
}

// Rate operations
func (db *DB) SetRate(groupID int64, role models.UserRole, rate float64) error {
	_, err := db.Exec(`
		INSERT INTO rates (group_id, role, rate_per_session)
		VALUES ($1, $2, $3)
		ON CONFLICT (group_id, role) DO UPDATE
		SET rate_per_session = EXCLUDED.rate_per_session,
		    updated_at = CURRENT_TIMESTAMP
	`, groupID, role, rate)

	return err
}

func (db *DB) GetRate(groupID int64, role models.UserRole) (float64, error) {
	var rate float64
	err := db.QueryRow(`
		SELECT rate_per_session FROM rates WHERE group_id = $1 AND role = $2
	`, groupID, role).Scan(&rate)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return rate, err
}

func (db *DB) GetAllRates(groupID int64) (map[models.UserRole]float64, error) {
	rows, err := db.Query(`
		SELECT role, rate_per_session FROM rates WHERE group_id = $1
	`, groupID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rates := make(map[models.UserRole]float64)
	for rows.Next() {
		var role models.UserRole
		var rate float64
		if err := rows.Scan(&role, &rate); err != nil {
			return nil, err
		}
		rates[role] = rate
	}

	return rates, nil
}

// Session operations
func (db *DB) AddSessionsToUser(userID, groupID int64, sessions int) error {
	_, err := db.Exec(`
		UPDATE user_groups
		SET sessions_owed = sessions_owed + $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2 AND group_id = $3
	`, sessions, userID, groupID)

	return err
}

func (db *DB) SettleSessions(userID, groupID int64, sessions int) error {
	_, err := db.Exec(`
		UPDATE user_groups
		SET sessions_owed = (sessions_owed - $1),
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2 AND group_id = $3
	`, sessions, userID, groupID)

	return err
}

// Event operations
func (db *DB) CreateEvent(groupID, createdBy int64, month, sessionDate string, capacity int) (*models.Event, error) {
	var e models.Event

	err := db.QueryRow(`
		INSERT INTO events (group_id, created_by, month, session_date, capacity)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, group_id, created_by, month, session_date, capacity, is_closed, created_at, updated_at
	`, groupID, createdBy, month, sessionDate, capacity).Scan(
		&e.ID, &e.GroupID, &e.CreatedBy, &e.Month, &e.SessionDate,
		&e.Capacity, &e.IsClosed, &e.CreatedAt, &e.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return &e, nil
}

func (db *DB) SetEventGroupMessageID(eventID int64, messageID int) error {
	_, err := db.Exec(`
		UPDATE events
		SET group_message_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, messageID, eventID)

	return err
}

func (db *DB) GetEventByID(eventID int64) (*models.Event, error) {
	var e models.Event
	var groupMessageID sql.NullInt64

	err := db.QueryRow(`
		SELECT id, group_id, created_by, month, session_date, capacity,
		       group_message_id, is_closed, created_at, updated_at
		FROM events
		WHERE id = $1
	`, eventID).Scan(
		&e.ID, &e.GroupID, &e.CreatedBy, &e.Month, &e.SessionDate, &e.Capacity,
		&groupMessageID, &e.IsClosed, &e.CreatedAt, &e.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	e.GroupMessageID = int(groupMessageID.Int64)

	return &e, nil
}

func (db *DB) GetOpenEvents(groupID int64) ([]models.Event, error) {
	rows, err := db.Query(`
		SELECT id, group_id, created_by, month, session_date, capacity,
		       group_message_id, is_closed, created_at, updated_at
		FROM events
		WHERE group_id = $1 AND is_closed = FALSE
		ORDER BY created_at DESC
	`, groupID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		var groupMessageID sql.NullInt64

		err := rows.Scan(
			&e.ID, &e.GroupID, &e.CreatedBy, &e.Month, &e.SessionDate, &e.Capacity,
			&groupMessageID, &e.IsClosed, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		e.GroupMessageID = int(groupMessageID.Int64)
		events = append(events, e)
	}

	return events, nil
}

// GetEvents returns the most recent events of a group, closed ones included.
// Admins need closed events too, because they may still override a member there.
func (db *DB) GetEvents(groupID int64, limit int) ([]models.Event, error) {
	rows, err := db.Query(`
		SELECT id, group_id, created_by, month, session_date, capacity,
		       group_message_id, is_closed, created_at, updated_at
		FROM events
		WHERE group_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, groupID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		var groupMessageID sql.NullInt64

		err := rows.Scan(
			&e.ID, &e.GroupID, &e.CreatedBy, &e.Month, &e.SessionDate, &e.Capacity,
			&groupMessageID, &e.IsClosed, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		e.GroupMessageID = int(groupMessageID.Int64)
		events = append(events, e)
	}

	return events, nil
}

func (db *DB) GetUserByID(userID int64) (*models.User, error) {
	var user models.User

	err := db.QueryRow(`
		SELECT id, telegram_id, username, first_name, last_name, is_bot, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.LastName, &user.IsBot, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *DB) CloseEvent(eventID int64) error {
	_, err := db.Exec(`
		UPDATE events
		SET is_closed = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, eventID)

	return err
}

// GetEventResponse returns nil (with a nil error) when the user has not answered yet.
func (db *DB) GetEventResponse(eventID, userID int64) (*models.EventResponse, error) {
	var r models.EventResponse

	err := db.QueryRow(`
		SELECT id, event_id, user_id, response, created_at, updated_at
		FROM event_responses
		WHERE event_id = $1 AND user_id = $2
	`, eventID, userID).Scan(
		&r.ID, &r.EventID, &r.UserID, &r.Response, &r.CreatedAt, &r.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (db *DB) SetEventResponse(eventID, userID int64, response models.EventResponseType) error {
	_, err := db.Exec(`
		INSERT INTO event_responses (event_id, user_id, response)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id, user_id) DO UPDATE
		SET response = EXCLUDED.response,
		    updated_at = CURRENT_TIMESTAMP
	`, eventID, userID, response)

	return err
}

// GetEventAnswers returns every answer for an event, oldest first, joined with the
// name each person registered with in that event's group.
func (db *DB) GetEventAnswers(eventID int64) ([]models.EventAnswer, error) {
	rows, err := db.Query(`
		SELECT er.user_id, ug.name, er.response
		FROM event_responses er
		JOIN events e ON e.id = er.event_id
		JOIN user_groups ug ON ug.user_id = er.user_id AND ug.group_id = e.group_id
		WHERE er.event_id = $1
		ORDER BY er.created_at
	`, eventID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var answers []models.EventAnswer
	for rows.Next() {
		var a models.EventAnswer
		if err := rows.Scan(&a.UserID, &a.Name, &a.Response); err != nil {
			return nil, err
		}
		answers = append(answers, a)
	}

	return answers, nil
}

// Guest operations. A guest belongs to a single event and costs the member who
// added them one session at that member's own rate. The guest row and the charge
// are written in one transaction so the board can never disagree with the money.
func (db *DB) AddEventGuest(eventID, addedBy, groupID int64, name string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO event_guests (event_id, name, added_by)
		VALUES ($1, $2, $3)
	`, eventID, name, addedBy); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE user_groups
		SET sessions_owed = sessions_owed + 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND group_id = $2
	`, addedBy, groupID); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) GetEventGuestByID(guestID int64) (*models.EventGuest, error) {
	var g models.EventGuest

	err := db.QueryRow(`
		SELECT id, event_id, name, added_by, created_at
		FROM event_guests
		WHERE id = $1
	`, guestID).Scan(&g.ID, &g.EventID, &g.Name, &g.AddedBy, &g.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &g, nil
}

func (db *DB) GetEventGuests(eventID int64) ([]models.EventGuest, error) {
	rows, err := db.Query(`
		SELECT id, event_id, name, added_by, created_at
		FROM event_guests
		WHERE event_id = $1
		ORDER BY created_at
	`, eventID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []models.EventGuest
	for rows.Next() {
		var g models.EventGuest
		if err := rows.Scan(&g.ID, &g.EventID, &g.Name, &g.AddedBy, &g.CreatedAt); err != nil {
			return nil, err
		}
		guests = append(guests, g)
	}

	return guests, nil
}

// DeleteEventGuest removes a guest and gives the session back to whoever added
// them. The DELETE ... RETURNING is what makes a double tap safe: the second call
// deletes no row, so no second refund happens.
func (db *DB) DeleteEventGuest(guestID, groupID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var addedBy int64
	err = tx.QueryRow(`
		DELETE FROM event_guests WHERE id = $1 RETURNING added_by
	`, guestID).Scan(&addedBy)

	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE user_groups
		SET sessions_owed = sessions_owed - 1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND group_id = $2
	`, addedBy, groupID); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) GetGroupByID(groupID int64) (*models.Group, error) {
	var group models.Group

	err := db.QueryRow(`
		SELECT id, telegram_chat_id, title, type, created_at, updated_at
		FROM groups
		WHERE id = $1
	`, groupID).Scan(
		&group.ID, &group.TelegramChatID, &group.Title, &group.Type,
		&group.CreatedAt, &group.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &group, nil
}

// GetGroupMembers returns everyone registered in a group together with the
// numeric id needed to message them directly.
func (db *DB) GetGroupMembers(groupID int64) ([]models.GroupMember, error) {
	rows, err := db.Query(`
		SELECT ug.user_id, u.telegram_id, ug.name, ug.role, ug.sessions_owed
		FROM user_groups ug
		JOIN users u ON u.id = ug.user_id
		WHERE ug.group_id = $1
		ORDER BY ug.name
	`, groupID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.GroupMember
	for rows.Next() {
		var m models.GroupMember
		err := rows.Scan(&m.UserID, &m.TelegramID, &m.Name, &m.Role, &m.SessionsOwed)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

func (db *DB) GetAllGroups() ([]models.Group, error) {
	rows, err := db.Query(`
		SELECT id, telegram_chat_id, title, type, created_at, updated_at
		FROM groups
		ORDER BY created_at DESC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		err := rows.Scan(
			&g.ID, &g.TelegramChatID, &g.Title, &g.Type,
			&g.CreatedAt, &g.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	return groups, nil
}
