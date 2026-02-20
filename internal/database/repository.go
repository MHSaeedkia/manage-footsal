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

func (db *DB) GetUserByUserName(userName string) (*models.User, error) {
	var user models.User

	err := db.QueryRow(`
		SELECT id, telegram_id, username, first_name, last_name, is_bot, created_at, updated_at
		FROM users
		WHERE username = $1
	`, userName).Scan(
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
