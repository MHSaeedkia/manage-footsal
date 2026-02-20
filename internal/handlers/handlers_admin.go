package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"futsal-bot/internal/bot"
	"futsal-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

const (
	userColWidth    = 15
	separatorWidth  = 7
	sessionColWidth = 15
)

func handleSetRatesCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	// Check if user is admin
	user, err := b.DB.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		b.SendMessage(callback.Message.Chat.ID, "خطا در دریافت اطلاعات کاربر.", nil)
		return
	}

	isAdmin := b.IsDefaultAdmin(callback.From.ID)
	if !isAdmin {
		isAdmin, _ = b.DB.IsUserAdminInGroup(user.ID, groupID)
	}

	if !isAdmin {
		b.AnswerCallbackQuery(callback.ID, "شما دسترسی ادمین ندارید.")
		return
	}

	keyboard := b.RateSettingKeyboard(groupID)
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "برای تنظیم نرخ، یک نقش را انتخاب کنید:", &keyboard)
}

func handleSetRateCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		return
	}

	roleStr := parts[1]
	groupID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}

	role := models.UserRole(roleStr)

	roleNames := map[models.UserRole]string{
		models.RoleStudent:   "دانشجو",
		models.RoleAdult:     "بزرگسال",
		models.RoleHalfAdult: "نیمه بزرگسال",
	}

	tempData := map[string]interface{}{
		"group_id": groupID,
		"role":     role,
	}
	b.SetState(callback.From.ID, "awaiting_rate", tempData)

	text := fmt.Sprintf("لطفا نرخ هر جلسه برای %s را به تومان وارد کنید:", roleNames[role])
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text, nil)
}

func handleSettleCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	// Check if user is admin
	user, err := b.DB.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		b.SendMessage(callback.Message.Chat.ID, "خطا در دریافت اطلاعات کاربر.", nil)
		return
	}

	isAdmin := b.IsDefaultAdmin(callback.From.ID)
	if !isAdmin {
		isAdmin, _ = b.DB.IsUserAdminInGroup(user.ID, groupID)
	}

	if !isAdmin {
		b.AnswerCallbackQuery(callback.ID, "شما دسترسی ادمین ندارید.")
		return
	}

	// Get all users in group
	userGroups, err := b.DB.GetUserGroupsByGroupID(groupID)
	if err != nil || len(userGroups) == 0 {
		backKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("back:%d", groupID)),
			),
		)
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID,
			"هیچ کاربری در این گروه ثبت نشده است.", &backKeyboard)
		return
	}

	// Create keyboard with user list
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, ug := range userGroups {
		if ug.SessionsOwed > 0 {
			buttonText := fmt.Sprintf("%s - %d جلسه", ug.Name, ug.SessionsOwed)
			buttonData := fmt.Sprintf("settle_user:%d:%d", ug.UserID, groupID)
			rows = append(rows, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(buttonText, buttonData),
			})
		} else if ug.SessionsOwed < 0 {
			buttonText := fmt.Sprintf("%s - %d جلسه طلب کار", ug.Name, ug.SessionsOwed*(-1))
			rows = append(rows, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(buttonText, "noop"),
			})
		} else {
			buttonText := fmt.Sprintf("%s - تسویه", ug.Name)
			rows = append(rows, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData(buttonText, "noop"),
			})
		}
	}

	if len(rows) == 0 {
		// No users with debt - edit the message to show this
		backKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("back:%d", groupID)),
			),
		)
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID,
			"هیچ کاربری با بدهی در این گروه وجود ندارد.", &backKeyboard)
		return
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("back:%d", groupID)),
	})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "کاربری که می‌خواهید تسویه کنید را انتخاب کنید:", &keyboard)
}

func handleSettleUserCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		return
	}

	targetUserID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	groupID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}

	tempData := map[string]interface{}{
		"target_user_id": targetUserID,
		"group_id":       groupID,
	}
	b.SetState(callback.From.ID, "awaiting_settle_sessions", tempData)

	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "تعداد جلساتی که تسویه شده را وارد کنید:", nil)
}

func handleBackCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	user, err := b.DB.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		return
	}

	isAdmin := b.IsDefaultAdmin(callback.From.ID)
	if !isAdmin {
		isAdmin, _ = b.DB.IsUserAdminInGroup(user.ID, groupID)
	}

	keyboard := b.MainMenuKeyboard(user.ID, groupID, isAdmin)
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID,
		"منوی اصلی:", &keyboard)
}

// Group message handlers
func HandleGroupMessage(b *bot.Bot, message *tgbotapi.Message) {
	// Handle when bot is added to a group
	if message.NewChatMembers != nil {
		for _, member := range message.NewChatMembers {
			if member.ID == b.API.Self.ID {
				// Bot was added to group
				_, err := b.DB.GetOrCreateGroup(
					message.Chat.ID,
					message.Chat.Title,
					message.Chat.Type,
				)
				if err != nil {
					zap.L().Error("Error creating group", zap.Error(err), zap.Int64("chat_id", message.Chat.ID))
				} else {
					b.SendMessage(message.Chat.ID,
						"سلام! من ربات مدیریت فوتسال هستم. "+
							"برای استفاده از امکانات من، لطفا به پیوی من مراجعه کنید.",
						nil)
				}
			}
		}
	}

	// Handle commands in group
	if message.IsCommand() {
		switch message.Command() {
		case "attendance":
			handleAttendanceCommand(b, message)
		case "report":
			handleReportCommand(b, message)
		}
	}
}

func handleAttendanceCommand(b *bot.Bot, message *tgbotapi.Message) {
	// Check if sender is admin
	user, err := b.DB.GetUserByTelegramID(message.From.ID)
	if err != nil {
		b.SendMessage(message.Chat.ID, "خطا در دریافت اطلاعات کاربر.", nil)
		return
	}

	group, err := b.DB.GetGroupByTelegramChatID(message.Chat.ID)
	if err != nil {
		b.SendMessage(message.Chat.ID, "این گروه در سیستم ثبت نشده است.", nil)
		return
	}

	isAdmin := b.IsDefaultAdmin(message.From.ID)
	if !isAdmin {
		isAdmin, _ = b.DB.IsUserAdminInGroup(user.ID, group.ID)
	}

	if !isAdmin {
		b.SendMessage(message.Chat.ID, "فقط ادمین‌ها می‌توانند حضور و غیاب ثبت کنند.", nil)
		return
	}

	// Parse user IDs from command arguments
	args := strings.Fields(message.CommandArguments())
	if len(args) == 0 {
		b.SendMessage(message.Chat.ID, "لطفا آیدی کاربران را وارد کنید.\n"+"مثال: /attendance @user1 @user2 @user3 @user4", nil)
		return
	}

	var userIDs []int64
	var userNames []string
	for _, arg := range args {
		userName := strings.TrimPrefix(arg, "@")
		userNames = append(userNames, userName)
	}

	if len(userNames) == 0 {
		b.SendMessage(message.Chat.ID, "هیچ آیدی معتبری یافت نشد.", nil)
		return
	}

	// Add sessions for each user
	successCount := 0
	for _, userName := range userNames {
		u, err := b.DB.GetUserByUserName(userName)
		if err != nil {
			zap.L().Error("Error getting user by username", zap.String("username", userName), zap.Error(err))
			continue
		}

		// Check if user is member of this group
		isMember, err := b.DB.IsUserMemberOfGroup(u.ID, group.ID)
		if err != nil || !isMember {
			continue
		}

		userIDs = append(userIDs, u.TelegramID)

		err = b.DB.AddSessionsToUser(u.ID, group.ID, 1)
		if err != nil {
			zap.L().Error("Error adding session for user", zap.Int64("telegram_id", u.TelegramID), zap.Error(err))
			continue
		}

		successCount++
	}

	text := fmt.Sprintf(
		"✅ حضور و غیاب ثبت شد.\n\n"+
			"تعداد کاربران: %d\n",
		successCount,
	)

	b.SendMessage(message.Chat.ID, text, nil)
}

func handleReportCommand(b *bot.Bot, message *tgbotapi.Message) {
	zap.L().Info("Handling report command", zap.Int64("chat_id", message.Chat.ID))
	// Check if sender is admin
	user, err := b.DB.GetUserByTelegramID(message.From.ID)
	if err != nil {
		b.SendMessage(message.Chat.ID, "خطا در دریافت اطلاعات کاربر.", nil)
		return
	}

	group, err := b.DB.GetGroupByTelegramChatID(message.Chat.ID)
	if err != nil {
		b.SendMessage(message.Chat.ID, "این گروه در سیستم ثبت نشده است.", nil)
		return
	}

	isAdmin := b.IsDefaultAdmin(message.From.ID)
	if !isAdmin {
		isAdmin, _ = b.DB.IsUserAdminInGroup(user.ID, group.ID)
	}

	if !isAdmin {
		b.SendMessage(message.Chat.ID, "فقط ادمین‌ها می‌توانند گزارش مشاهده کنند.", nil)
		return
	}

	// Get all users with debts in this group
	userGroups, err := b.DB.GetUserGroupsByGroupID(group.ID)
	if err != nil {
		b.SendMessage(message.Chat.ID, "خطا در دریافت اطلاعات.", nil)
		return
	}

	var reportLines []string
	reportLines = append(reportLines, "📊 گزارش بدهی‌ جلسات\n")
	hasDebts := false
	for _, ug := range userGroups {
		hasDebts = true
		// Get user telegram ID
		var telegramUsername, line string
		err := b.DB.QueryRow(`
				SELECT username FROM users WHERE id = $1
			`, ug.UserID).Scan(&telegramUsername)

		if err != nil {
			continue
		}

		if ug.SessionsOwed > 0 {
			line = fmt.Sprintf("• %s = %d", telegramUsername, ug.SessionsOwed)
		} else if ug.SessionsOwed < 0 {
			line = fmt.Sprintf("• %s = %d ❤️", telegramUsername, ug.SessionsOwed)
		} else {
			line = fmt.Sprintf("• %s = %d ✅", telegramUsername, ug.SessionsOwed)
		}

		reportLines = append(reportLines, line)
	}

	if !hasDebts {
		b.SendMessage(message.Chat.ID, "هیچ بدهی در این گروه وجود ندارد.", nil)
		return
	}

	report := strings.Join(reportLines, "\n")
	b.SendMessageWithMarkdown(message.Chat.ID, report, nil)
}

func escapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

func center(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}

	padding := width - len(text)
	left := padding / 2
	right := padding - left

	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}
