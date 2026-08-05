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

// roleNames is the Persian label for each role.
var roleNames = map[models.UserRole]string{
	models.RoleAdmin:     "ادمین",
	models.RoleStudent:   "دانشجو",
	models.RoleAdult:     "بزرگسال",
	models.RoleHalfAdult: "نیمه بزرگسال",
}

// eventTemplate is the board posted in the group. Order of the arguments is
// month (var1), session date (var2), capacity (var3), present list, absent list.
const eventTemplate = `بسم الله

 *%s ماه*
حاضرین و غایبین سالن *%s* : فوتسال

🔹 لطفا تا قبل ساعت ۱۲ جمعه اعلام حضور کنین.
🔹 ️لطفا به موقع تشریف بیاورید.

                      ⭕️ *ظرفیت %d نفر* ⭕️

🔹 حاضرین :
%s
🔻 غایبین:
%s`

// escapeMarkdown escapes the characters that are special in Telegram/Bale legacy
// Markdown. Names are free text typed by users, and an unescaped "*" or "_" makes
// the whole message fail to parse, which would stop the board from updating.
func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"`", "\\`",
		"[", "\\[",
	)
	return replacer.Replace(text)
}

// buildInvoiceText is the shared wording for "how much does this person owe".
func buildInvoiceText(name string, role models.UserRole, sessions int, rate float64) string {
	return fmt.Sprintf(
		"💰 *صورتحساب*\n\n"+
			"نام: %s\n"+
			"نقش: %s\n"+
			"تعداد جلسات: %d\n"+
			"نرخ هر جلسه: %.0f تومان\n"+
			"مجموع بدهی: %.0f تومان",
		escapeMarkdown(name), roleNames[role], sessions, rate, float64(sessions)*rate,
	)
}

// buildEventBoard fills the template with the present/absent lists. Guests are
// listed after the members, inside the present list, and marked as guests.
func buildEventBoard(event *models.Event, answers []models.EventAnswer, guests []models.EventGuest) string {
	var present, absent []string
	for _, a := range answers {
		if a.Response == models.ResponsePresent {
			present = append(present, fmt.Sprintf("%d- %s", len(present)+1, escapeMarkdown(a.Name)))
		} else {
			absent = append(absent, fmt.Sprintf("%d- %s", len(absent)+1, escapeMarkdown(a.Name)))
		}
	}

	for _, g := range guests {
		present = append(present, fmt.Sprintf("%d- %s (مهمان)", len(present)+1, escapeMarkdown(g.Name)))
	}

	board := fmt.Sprintf(eventTemplate,
		escapeMarkdown(event.Month),
		escapeMarkdown(event.SessionDate),
		event.Capacity,
		strings.Join(present, "\n"),
		strings.Join(absent, "\n"),
	)

	if event.IsClosed {
		board += "\n\n🔒 اعلام حضور بسته شد."
	}

	return board
}

// sessionDelta is how many sessions to add to the member's account when they move
// from their previous answer (nil when they had not answered) to a new one.
func sessionDelta(previous *models.EventResponse, response models.EventResponseType) int {
	if previous == nil {
		if response == models.ResponsePresent {
			return 1
		}
		return 0
	}

	if previous.Response == response {
		return 0
	}

	if response == models.ResponsePresent {
		return 1
	}
	return -1
}

// renderEventBoard fills the template with the current present/absent lists.
func renderEventBoard(b *bot.Bot, event *models.Event) (string, error) {
	answers, err := b.DB.GetEventAnswers(event.ID)
	if err != nil {
		return "", err
	}

	guests, err := b.DB.GetEventGuests(event.ID)
	if err != nil {
		return "", err
	}

	return buildEventBoard(event, answers, guests), nil
}

// refreshEventBoard rewrites the group message so it matches the database.
func refreshEventBoard(b *bot.Bot, event *models.Event) {
	if event.GroupMessageID == 0 {
		return
	}

	group, err := b.DB.GetGroupByID(event.GroupID)
	if err != nil {
		zap.L().Error("Error getting group for event board", zap.Int64("event_id", event.ID), zap.Error(err))
		return
	}

	board, err := renderEventBoard(b, event)
	if err != nil {
		zap.L().Error("Error rendering event board", zap.Int64("event_id", event.ID), zap.Error(err))
		return
	}

	if err := b.EditMessageWithMarkdown(group.TelegramChatID, event.GroupMessageID, board, nil); err != nil {
		zap.L().Error("Error updating event board", zap.Int64("event_id", event.ID), zap.Error(err))
	}
}

// Event creation (admin)

func handleNewEventCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	if !isAdminFor(b, callback.From.ID, groupID) {
		b.AnswerCallbackQuery(callback.ID, "شما دسترسی ادمین ندارید.")
		return
	}

	b.SetState(callback.From.ID, "awaiting_event_month", map[string]interface{}{"group_id": groupID})
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "نام ماه را وارد کنید (مثال: مرداد):", nil)
}

func handleEventMonthInput(b *bot.Bot, message *tgbotapi.Message, state *models.UserState) {
	month := strings.TrimSpace(message.Text)
	if month == "" {
		b.SendMessage(message.Chat.ID, "لطفا نام ماه را وارد کنید:", nil)
		return
	}

	state.TempData["month"] = month
	b.SetState(message.From.ID, "awaiting_event_date", state.TempData)
	b.SendMessage(message.Chat.ID, "تاریخ سانس را وارد کنید (مثال: جمعه ۱۵):", nil)
}

func handleEventDateInput(b *bot.Bot, message *tgbotapi.Message, state *models.UserState) {
	sessionDate := strings.TrimSpace(message.Text)
	if sessionDate == "" {
		b.SendMessage(message.Chat.ID, "لطفا تاریخ سانس را وارد کنید:", nil)
		return
	}

	state.TempData["session_date"] = sessionDate
	b.SetState(message.From.ID, "awaiting_event_capacity", state.TempData)
	b.SendMessage(message.Chat.ID, "ظرفیت سانس را وارد کنید (تعداد نفرات):", nil)
}

func handleEventCapacityInput(b *bot.Bot, message *tgbotapi.Message, state *models.UserState) {
	capacity, err := strconv.Atoi(strings.TrimSpace(message.Text))
	if err != nil || capacity <= 0 {
		b.SendMessage(message.Chat.ID, "لطفا یک عدد معتبر وارد کنید:", nil)
		return
	}

	groupID := state.TempData["group_id"].(int64)
	month := state.TempData["month"].(string)
	sessionDate := state.TempData["session_date"].(string)

	b.ClearState(message.From.ID)

	user, err := b.DB.GetUserByTelegramID(message.From.ID)
	if err != nil {
		zap.L().Error("Error getting admin user", zap.Int64("telegram_id", message.From.ID), zap.Error(err))
		b.SendMessage(message.Chat.ID, "خطا در دریافت اطلاعات کاربر.", nil)
		return
	}

	event, err := b.DB.CreateEvent(groupID, user.ID, month, sessionDate, capacity)
	if err != nil {
		zap.L().Error("Error creating event", zap.Int64("group_id", groupID), zap.Error(err))
		b.SendMessage(message.Chat.ID, "خطا در ایجاد سانس.", nil)
		return
	}

	invited := publishEvent(b, event)

	b.SendMessage(message.Chat.ID, fmt.Sprintf(
		"✅ سانس ایجاد شد.\n\n"+
			"ماه: %s\n"+
			"تاریخ: %s\n"+
			"ظرفیت: %d نفر\n"+
			"دعوت ارسال شده برای: %d نفر",
		month, sessionDate, capacity, invited), nil)
}

// publishEvent posts the board in the group and sends the buttons to every
// registered member. It returns how many members were reached.
func publishEvent(b *bot.Bot, event *models.Event) int {
	group, err := b.DB.GetGroupByID(event.GroupID)
	if err != nil {
		zap.L().Error("Error getting group for new event", zap.Int64("event_id", event.ID), zap.Error(err))
		return 0
	}

	board, err := renderEventBoard(b, event)
	if err != nil {
		zap.L().Error("Error rendering new event board", zap.Int64("event_id", event.ID), zap.Error(err))
		return 0
	}

	sent, err := b.SendMarkdownMessage(group.TelegramChatID, board, nil)
	if err != nil {
		zap.L().Error("Error posting event board to group", zap.Int64("event_id", event.ID), zap.Error(err))
	} else {
		event.GroupMessageID = sent.MessageID
		if err := b.DB.SetEventGroupMessageID(event.ID, sent.MessageID); err != nil {
			zap.L().Error("Error saving event message id", zap.Int64("event_id", event.ID), zap.Error(err))
		}
	}

	members, err := b.DB.GetGroupMembers(event.GroupID)
	if err != nil {
		zap.L().Error("Error getting group members", zap.Int64("group_id", event.GroupID), zap.Error(err))
		return 0
	}

	invite := fmt.Sprintf(
		"🏟 *سانس جدید فوتسال*\n\n"+
			"ماه: %s\n"+
			"تاریخ: %s\n"+
			"ظرفیت: %d نفر\n\n"+
			"لطفا حضور خود را اعلام کنید:",
		escapeMarkdown(event.Month), escapeMarkdown(event.SessionDate), event.Capacity)

	keyboard := b.EventResponseKeyboard(event.ID)

	invited := 0
	for _, m := range members {
		if _, err := b.SendMarkdownMessage(m.TelegramID, invite, keyboard); err != nil {
			zap.L().Error("Error sending event invite",
				zap.Int64("telegram_id", m.TelegramID), zap.Int64("event_id", event.ID), zap.Error(err))
			continue
		}
		invited++
	}

	return invited
}

// Member answering the event

func handleRSVPCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		return
	}

	response := models.EventResponseType(parts[1])
	if response != models.ResponsePresent && response != models.ResponseAbsent {
		return
	}

	eventID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}

	event, err := b.DB.GetEventByID(eventID)
	if err != nil {
		zap.L().Error("Error getting event", zap.Int64("event_id", eventID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "این سانس پیدا نشد.")
		return
	}

	if event.IsClosed {
		b.AnswerCallbackQuery(callback.ID, "این سانس بسته شده است.")
		return
	}

	user, err := b.DB.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		b.AnswerCallbackQuery(callback.ID, "خطا در دریافت اطلاعات کاربر.")
		return
	}

	isMember, err := b.DB.IsUserMemberOfGroup(user.ID, event.GroupID)
	if err != nil || !isMember {
		b.AnswerCallbackQuery(callback.ID, "شما در این گروه ثبت نام نکرده‌اید.")
		return
	}

	previous, err := b.DB.GetEventResponse(eventID, user.ID)
	if err != nil {
		zap.L().Error("Error getting previous response",
			zap.Int64("event_id", eventID), zap.Int64("user_id", user.ID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در ثبت پاسخ.")
		return
	}

	// Saying "present" costs one session, exactly like /attendance. Changing the
	// answer later moves the session back or forward again.
	delta := sessionDelta(previous, response)

	if delta != 0 {
		if err := b.DB.AddSessionsToUser(user.ID, event.GroupID, delta); err != nil {
			zap.L().Error("Error updating sessions for rsvp",
				zap.Int64("event_id", eventID), zap.Int64("user_id", user.ID), zap.Error(err))
			b.AnswerCallbackQuery(callback.ID, "خطا در ثبت پاسخ.")
			return
		}
	}

	if err := b.DB.SetEventResponse(eventID, user.ID, response); err != nil {
		zap.L().Error("Error saving response",
			zap.Int64("event_id", eventID), zap.Int64("user_id", user.ID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در ثبت پاسخ.")
		return
	}

	refreshEventBoard(b, event)

	if response == models.ResponsePresent {
		b.AnswerCallbackQuery(callback.ID, "حضور شما ثبت شد.")
	} else {
		b.AnswerCallbackQuery(callback.ID, "غیبت شما ثبت شد.")
	}

	sendSessionBill(b, callback.From.ID, user.ID, event, response, "")
}

// sendSessionBill tells the member what this session cost them and where their
// total stands now. notice is an extra line put on top, used when an admin
// changed the status instead of the member themselves.
func sendSessionBill(b *bot.Bot, chatID, userID int64, event *models.Event, response models.EventResponseType, notice string) {
	ug, err := b.DB.GetUserGroup(userID, event.GroupID)
	if err != nil {
		zap.L().Error("Error getting user group for bill", zap.Int64("user_id", userID), zap.Error(err))
		return
	}

	rate, err := b.DB.GetRate(event.GroupID, ug.Role)
	if err != nil {
		zap.L().Error("Error getting rate for bill", zap.Int64("group_id", event.GroupID), zap.Error(err))
		return
	}

	status := "غایب"
	sessionCost := 0.0
	if response == models.ResponsePresent {
		status = "حاضر"
		sessionCost = rate
	}

	text := notice + fmt.Sprintf(
		"🧾 *صورتحساب این سانس*\n\n"+
			"سانس: %s %s\n"+
			"وضعیت شما: %s\n"+
			"هزینه این سانس: %.0f تومان\n\n"+
			"مجموع جلسات: %d\n"+
			"مجموع بدهی: %.0f تومان",
		escapeMarkdown(event.SessionDate), escapeMarkdown(event.Month),
		status, sessionCost, ug.SessionsOwed, float64(ug.SessionsOwed)*rate)

	if err := b.SendMessageWithMarkdown(chatID, text, nil); err != nil {
		zap.L().Error("Error sending session bill", zap.Int64("chat_id", chatID), zap.Error(err))
	}
}

// Closing an event (admin)

func handleCloseEventCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	if !isAdminFor(b, callback.From.ID, groupID) {
		b.AnswerCallbackQuery(callback.ID, "شما دسترسی ادمین ندارید.")
		return
	}

	events, err := b.DB.GetOpenEvents(groupID)
	if err != nil {
		zap.L().Error("Error getting open events", zap.Int64("group_id", groupID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در دریافت سانس‌ها.")
		return
	}

	backRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("back:%d", groupID)),
	}

	if len(events) == 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(backRow)
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "هیچ سانس بازی وجود ندارد.", &keyboard)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, e := range events {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %s", e.SessionDate, e.Month),
				fmt.Sprintf("do_close:%d:%d", e.ID, groupID)),
		})
	}
	rows = append(rows, backRow)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "کدام سانس بسته شود؟", &keyboard)
}

func handleDoCloseEventCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 3 {
		return
	}

	eventID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	groupID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}

	if !isAdminFor(b, callback.From.ID, groupID) {
		b.AnswerCallbackQuery(callback.ID, "شما دسترسی ادمین ندارید.")
		return
	}

	if err := b.DB.CloseEvent(eventID); err != nil {
		zap.L().Error("Error closing event", zap.Int64("event_id", eventID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در بستن سانس.")
		return
	}

	event, err := b.DB.GetEventByID(eventID)
	if err == nil {
		refreshEventBoard(b, event)
	}

	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "🔒 سانس بسته شد. دیگر کسی نمی‌تواند پاسخ خود را تغییر دهد.", nil)
}

// Sending everyone their bill (admin)

func handleBillAllCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	if !isAdminFor(b, callback.From.ID, groupID) {
		b.AnswerCallbackQuery(callback.ID, "شما دسترسی ادمین ندارید.")
		return
	}

	members, err := b.DB.GetGroupMembers(groupID)
	if err != nil {
		zap.L().Error("Error getting group members for bills", zap.Int64("group_id", groupID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در دریافت اعضا.")
		return
	}

	rates, err := b.DB.GetAllRates(groupID)
	if err != nil {
		zap.L().Error("Error getting rates for bills", zap.Int64("group_id", groupID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در دریافت نرخ‌ها.")
		return
	}

	sentCount := 0
	for _, m := range members {
		text := buildInvoiceText(m.Name, m.Role, m.SessionsOwed, rates[m.Role])
		if err := b.SendMessageWithMarkdown(m.TelegramID, text, nil); err != nil {
			zap.L().Error("Error sending bill", zap.Int64("telegram_id", m.TelegramID), zap.Error(err))
			continue
		}
		sentCount++
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("back:%d", groupID)),
		),
	)

	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID,
		fmt.Sprintf("📤 صورتحساب برای %d نفر از %d عضو ارسال شد.", sentCount, len(members)), &keyboard)
}

// isAdminFor reports whether the given Bale user may administer this group.
func isAdminFor(b *bot.Bot, telegramID, groupID int64) bool {
	if b.IsDefaultAdmin(telegramID) {
		return true
	}

	user, err := b.DB.GetUserByTelegramID(telegramID)
	if err != nil {
		return false
	}

	isAdmin, _ := b.DB.IsUserAdminInGroup(user.ID, groupID)
	return isAdmin
}
