package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"futsal-bot/internal/bot"
	"futsal-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleStart(b *bot.Bot, message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID

	// Get or create user
	user, err := b.DB.GetOrCreateUser(
		userID,
		message.From.UserName,
		message.From.FirstName,
		message.From.LastName,
		message.From.IsBot,
	)

	if err != nil {
		log.Printf("Error getting/creating user: %v", err)
		b.SendMessage(chatID, "خطا در ثبت اطلاعات کاربر. لطفا دوباره تلاش کنید.", nil)
		return
	}

	// Check if user is default admin
	isDefaultAdmin := b.IsDefaultAdmin(userID)

	// Get all groups where bot is member
	allGroups, err := b.DB.GetAllGroups()
	if err != nil {
		log.Printf("Error getting groups: %v", err)
	}

	if len(allGroups) == 0 && !isDefaultAdmin {
		b.SendMessage(chatID, "ربات در هیچ گروهی عضو نیست. لطفا ابتدا ربات را به یک گروه اضافه کنید.", nil)
		return
	}

	// For simplicity, show first group
	var groupID int64
	if len(allGroups) > 0 {
		groupID = allGroups[0].ID
	}

	// Show message about available groups if multiple exist
	if len(allGroups) > 1 {
		groupNames := make([]string, 0)
		for _, g := range allGroups {
			groupNames = append(groupNames, g.Title)
		}
		b.SendMessage(chatID,
			fmt.Sprintf("ربات در %d گروه عضو است: %s\n\nدر حال حاضر گروه اول نمایش داده می‌شود.",
				len(allGroups), strings.Join(groupNames, "، ")), nil)
	}

	isAdmin := isDefaultAdmin
	if !isDefaultAdmin && groupID > 0 {
		isAdmin, _ = b.DB.IsUserAdminInGroup(user.ID, groupID)
	}

	welcomeText := fmt.Sprintf("سلام %s! به ربات مدیریت فوتسال خوش آمدید.", message.From.FirstName)
	keyboard := b.MainMenuKeyboard(user.ID, groupID, isAdmin)

	b.SendMessage(chatID, welcomeText, keyboard)
}

func HandleMessage(b *bot.Bot, message *tgbotapi.Message) {
	// Check if user has a state
	state := b.GetState(message.From.ID)
	if state == nil {
		return
	}

	switch state.State {
	case "awaiting_name":
		handleNameInput(b, message, state)
	case "awaiting_rate":
		handleRateInput(b, message, state)
	case "awaiting_settle_sessions":
		handleSettleSessionsInput(b, message, state)
	default:
		b.ClearState(message.From.ID)
	}
}

func handleNameInput(b *bot.Bot, message *tgbotapi.Message, state *models.UserState) {
	name := strings.TrimSpace(message.Text)
	if name == "" {
		b.SendMessage(message.Chat.ID, "لطفا یک نام معتبر وارد کنید:", nil)
		return
	}

	// Save name in temp data
	state.TempData["name"] = name
	groupID := state.TempData["group_id"].(int64)

	// Get or create user
	user, err := b.DB.GetOrCreateUser(
		message.From.ID,
		message.From.UserName,
		message.From.FirstName,
		message.From.LastName,
		message.From.IsBot,
	)

	if err != nil {
		log.Printf("Error getting/creating user: %v", err)
		b.SendMessage(message.Chat.ID, "خطا در ثبت اطلاعات. لطفا دوباره تلاش کنید.", nil)
		b.ClearState(message.From.ID)
		return
	}

	isAdmin := b.IsDefaultAdmin(message.From.ID)
	if !isAdmin {
		isAdmin, _ = b.DB.IsUserAdminInGroup(user.ID, groupID)
	}

	// Update state to role selection
	state.State = "awaiting_role"
	state.TempData["user_id"] = user.ID
	b.SetState(message.From.ID, state.State, state.TempData)

	// Show role selection
	keyboard := b.RoleSelectionKeyboard(groupID, isAdmin)
	b.SendMessage(message.Chat.ID, "لطفا نقش خود را انتخاب کنید:", keyboard)
}

func handleRateInput(b *bot.Bot, message *tgbotapi.Message, state *models.UserState) {
	rateStr := strings.TrimSpace(message.Text)
	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil || rate < 0 {
		b.SendMessage(message.Chat.ID, "لطفا یک عدد معتبر وارد کنید:", nil)
		return
	}

	groupID := state.TempData["group_id"].(int64)
	role := state.TempData["role"].(models.UserRole)

	err = b.DB.SetRate(groupID, role, rate)
	if err != nil {
		log.Printf("Error setting rate: %v", err)
		b.SendMessage(message.Chat.ID, "خطا در ثبت نرخ. لطفا دوباره تلاش کنید.", nil)
		b.ClearState(message.From.ID)
		return
	}

	b.ClearState(message.From.ID)

	roleNames := map[models.UserRole]string{
		models.RoleStudent:   "دانشجو",
		models.RoleAdult:     "بزرگسال",
		models.RoleHalfAdult: "نیمه بزرگسال",
	}

	text := fmt.Sprintf("✅ نرخ برای %s به %.0f تومان تنظیم شد.", roleNames[role], rate)
	keyboard := b.RateSettingKeyboard(groupID)
	b.SendMessage(message.Chat.ID, text, keyboard)
}

func handleSettleSessionsInput(b *bot.Bot, message *tgbotapi.Message, state *models.UserState) {
	sessionsStr := strings.TrimSpace(message.Text)
	sessions, err := strconv.Atoi(sessionsStr)
	if err != nil || sessions <= 0 {
		b.SendMessage(message.Chat.ID, "لطفا یک عدد معتبر وارد کنید:", nil)
		return
	}

	userID := state.TempData["target_user_id"].(int64)
	groupID := state.TempData["group_id"].(int64)

	// Get user group info
	ug, err := b.DB.GetUserGroup(userID, groupID)
	if err != nil {
		log.Printf("Error getting user group: %v", err)
		b.SendMessage(message.Chat.ID, "خطا در دریافت اطلاعات کاربر.", nil)
		b.ClearState(message.From.ID)
		return
	}

	err = b.DB.SettleSessions(userID, groupID, sessions)
	if err != nil {
		log.Printf("Error settling sessions: %v", err)
		b.SendMessage(message.Chat.ID, "خطا در تسویه حساب.", nil)
		b.ClearState(message.From.ID)
		return
	}

	b.ClearState(message.From.ID)

	// Get updated info
	ug, _ = b.DB.GetUserGroup(userID, groupID)
	rate, _ := b.DB.GetRate(groupID, ug.Role)
	remainingDebt := float64(ug.SessionsOwed) * rate

	var text string

	if ug.SessionsOwed > 0 {
		text = fmt.Sprintf(
			"✅ تسویه حساب انجام شد.\n\n"+
				"کاربر: %s\n"+
				"جلسات تسویه شده: %d\n"+
				"جلسات باقیمانده: %d\n"+
				"بدهی باقیمانده: %.0f تومان",
			ug.Name, sessions, ug.SessionsOwed, remainingDebt,
		)
	} else if ug.SessionsOwed < 0 {
		text = fmt.Sprintf(
			"✅ تسویه حساب انجام شد.\n\n"+
				"کاربر: %s\n"+
				"جلسات تسویه شده: %d\n"+
				"جلسات طلب کار: %d\n",
			ug.Name, sessions, ug.SessionsOwed*(-1),
		)
	} else {
		text = fmt.Sprintf(
			"✅ تسویه حساب انجام شد.\n\n"+
				"کاربر: %s\n"+
				"جلسات تسویه شده: %d"+
				ug.Name, sessions,
		)
	}

	b.SendMessage(message.Chat.ID, text, nil)
}

func HandleCallbackQuery(b *bot.Bot, callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	// userID := callback.From.ID
	// chatID := callback.Message.Chat.ID

	parts := strings.Split(data, ":")
	if len(parts) < 1 {
		return
	}

	action := parts[0]

	switch action {
	case "register":
		handleRegisterCallback(b, callback, parts)
	case "edit":
		handleEditCallback(b, callback, parts)
	case "role":
		handleRoleCallback(b, callback, parts)
	case "invoice":
		handleInvoiceCallback(b, callback, parts)
	case "set_rates":
		handleSetRatesCallback(b, callback, parts)
	case "setrate":
		handleSetRateCallback(b, callback, parts)
	case "settle":
		handleSettleCallback(b, callback, parts)
	case "settle_user":
		handleSettleUserCallback(b, callback, parts)
	case "back":
		handleBackCallback(b, callback, parts)
	}

	b.AnswerCallbackQuery(callback.ID, "")
}

func handleRegisterCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	// Start registration process
	tempData := map[string]interface{}{
		"group_id": groupID,
		"action":   "register",
	}
	b.SetState(callback.From.ID, "awaiting_name", tempData)

	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID,
		"لطفا نام خود را وارد کنید:", nil)
}

func handleEditCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	// Start edit process
	tempData := map[string]interface{}{
		"group_id": groupID,
		"action":   "edit",
	}
	b.SetState(callback.From.ID, "awaiting_name", tempData)

	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID,
		"لطفا نام جدید خود را وارد کنید:", nil)
}

func handleRoleCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		return
	}

	roleStr := parts[1]
	groupID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}

	role := models.UserRole(roleStr)
	state := b.GetState(callback.From.ID)
	if state == nil {
		return
	}

	name := state.TempData["name"].(string)
	userID := state.TempData["user_id"].(int64)

	// Save user group
	err = b.DB.CreateOrUpdateUserGroup(userID, groupID, role, name)
	if err != nil {
		log.Printf("Error creating/updating user group: %v", err)
		b.SendMessage(callback.Message.Chat.ID, "خطا در ثبت اطلاعات.", nil)
		b.ClearState(callback.From.ID)
		return
	}

	b.ClearState(callback.From.ID)

	roleNames := map[models.UserRole]string{
		models.RoleAdmin:     "ادمین",
		models.RoleStudent:   "دانشجو",
		models.RoleAdult:     "بزرگسال",
		models.RoleHalfAdult: "نیمه بزرگسال",
	}

	text := fmt.Sprintf("✅ ثبت نام با موفقیت انجام شد!\n\nنام: %s\nنقش: %s", name, roleNames[role])
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text, nil)
}

func handleInvoiceCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	user, err := b.DB.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		b.SendMessage(callback.Message.Chat.ID, "خطا در دریافت اطلاعات کاربر.", nil)
		return
	}

	ug, err := b.DB.GetUserGroup(user.ID, groupID)
	if err != nil {
		b.SendMessage(callback.Message.Chat.ID, "شما در این گروه ثبت نام نکرده‌اید.", nil)
		return
	}

	rate, _ := b.DB.GetRate(groupID, ug.Role)
	totalDebt := float64(ug.SessionsOwed) * rate

	roleNames := map[models.UserRole]string{
		models.RoleAdmin:     "ادمین",
		models.RoleStudent:   "دانشجو",
		models.RoleAdult:     "بزرگسال",
		models.RoleHalfAdult: "نیمه بزرگسال",
	}

	text := fmt.Sprintf(
		"💰 *صورتحساب*\n\n"+
			"نام: %s\n"+
			"نقش: %s\n"+
			"تعداد جلسات: %d\n"+
			"نرخ هر جلسه: %.0f تومان\n"+
			"مجموع بدهی: %.0f تومان",
		ug.Name, roleNames[ug.Role], ug.SessionsOwed, rate, totalDebt,
	)

	b.SendMessageWithMarkdown(callback.Message.Chat.ID, text, nil)
	b.AnswerCallbackQuery(callback.ID, "")
}
