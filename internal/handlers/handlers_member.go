package handlers

import (
	"fmt"
	"strconv"

	"futsal-bot/internal/bot"
	"futsal-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

// An admin can set a member's حاضر/غایب status for a session from private chat,
// instead of waiting for that member to tap their own buttons. The money moves
// exactly as if the member had answered themselves, and the member is told.
//
// Unlike the member's own buttons, this also works on a CLOSED session: closing
// stops members changing their mind, it does not stop the admin fixing mistakes.

// eventListLimit caps the session picker so it stays readable as events pile up.
const eventListLimit = 10

func handleMembersCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
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

	events, err := b.DB.GetEvents(groupID, eventListLimit)
	if err != nil {
		zap.L().Error("Error getting events", zap.Int64("group_id", groupID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در دریافت سانس‌ها.")
		return
	}

	backRow := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("back:%d", groupID)),
	}

	if len(events) == 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(backRow)
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "هیچ سانسی وجود ندارد.", &keyboard)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, e := range events {
		label := fmt.Sprintf("%s %s", e.SessionDate, e.Month)
		if e.IsClosed {
			label += " 🔒"
		}
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("mem_event:%d:%d", e.ID, groupID)),
		})
	}
	rows = append(rows, backRow)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "وضعیت اعضای کدام سانس؟", &keyboard)
}

// showMemberList draws every member of the group with the status they currently
// have for this session.
func showMemberList(b *bot.Bot, chatID int64, messageID int, eventID, groupID int64) {
	event, err := b.DB.GetEventByID(eventID)
	if err != nil {
		zap.L().Error("Error getting event for member list", zap.Int64("event_id", eventID), zap.Error(err))
		return
	}

	members, err := b.DB.GetGroupMembers(groupID)
	if err != nil {
		zap.L().Error("Error getting members", zap.Int64("group_id", groupID), zap.Error(err))
		return
	}

	answers, err := b.DB.GetEventAnswers(eventID)
	if err != nil {
		zap.L().Error("Error getting answers", zap.Int64("event_id", eventID), zap.Error(err))
		return
	}

	status := make(map[int64]models.EventResponseType, len(answers))
	for _, a := range answers {
		status[a.UserID] = a.Response
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, m := range members {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s %s", statusMark(status[m.UserID]), m.Name),
				fmt.Sprintf("mem_pick:%d:%d:%d", m.UserID, eventID, groupID)),
		})
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("members:%d", groupID)),
	})

	text := fmt.Sprintf("سانس %s %s\n\nیک عضو را انتخاب کنید:\n✅ حاضر    ❌ غایب    ➖ بدون پاسخ",
		event.SessionDate, event.Month)

	if event.IsClosed {
		text += "\n\n🔒 این سانس بسته است، ولی شما به عنوان ادمین می‌توانید تغییر دهید."
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.EditMessage(chatID, messageID, text, &keyboard)
}

// statusMark is the icon shown next to a member's name in the list.
func statusMark(response models.EventResponseType) string {
	switch response {
	case models.ResponsePresent:
		return "✅"
	case models.ResponseAbsent:
		return "❌"
	default:
		return "➖"
	}
}

func handleMemberEventCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
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

	showMemberList(b, callback.Message.Chat.ID, callback.Message.MessageID, eventID, groupID)
}

func handleMemberPickCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 4 {
		return
	}

	targetID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	eventID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}

	groupID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return
	}

	if !isAdminFor(b, callback.From.ID, groupID) {
		b.AnswerCallbackQuery(callback.ID, "شما دسترسی ادمین ندارید.")
		return
	}

	ug, err := b.DB.GetUserGroup(targetID, groupID)
	if err != nil {
		b.AnswerCallbackQuery(callback.ID, "این عضو پیدا نشد.")
		return
	}

	current, err := b.DB.GetEventResponse(eventID, targetID)
	if err != nil {
		zap.L().Error("Error getting response", zap.Int64("event_id", eventID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در دریافت وضعیت.")
		return
	}

	var currentResponse models.EventResponseType
	if current != nil {
		currentResponse = current.Response
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ حاضر", fmt.Sprintf("mem_set:present:%d:%d:%d", targetID, eventID, groupID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ غایب", fmt.Sprintf("mem_set:absent:%d:%d:%d", targetID, eventID, groupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("mem_event:%d:%d", eventID, groupID)),
		),
	)

	text := fmt.Sprintf(
		"عضو: %s\nوضعیت فعلی: %s\nجلسات فعلی: %d\n\nوضعیت جدید را انتخاب کنید:",
		ug.Name, statusLabel(currentResponse), ug.SessionsOwed)

	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text, &keyboard)
}

func statusLabel(response models.EventResponseType) string {
	switch response {
	case models.ResponsePresent:
		return "حاضر ✅"
	case models.ResponseAbsent:
		return "غایب ❌"
	default:
		return "بدون پاسخ ➖"
	}
}

func handleMemberSetCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 5 {
		return
	}

	response := models.EventResponseType(parts[1])
	if response != models.ResponsePresent && response != models.ResponseAbsent {
		return
	}

	targetID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}

	eventID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return
	}

	groupID, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return
	}

	if !isAdminFor(b, callback.From.ID, groupID) {
		b.AnswerCallbackQuery(callback.ID, "شما دسترسی ادمین ندارید.")
		return
	}

	event, err := b.DB.GetEventByID(eventID)
	if err != nil {
		b.AnswerCallbackQuery(callback.ID, "این سانس پیدا نشد.")
		return
	}

	previous, err := b.DB.GetEventResponse(eventID, targetID)
	if err != nil {
		zap.L().Error("Error getting previous response",
			zap.Int64("event_id", eventID), zap.Int64("user_id", targetID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در ثبت وضعیت.")
		return
	}

	// Same rule as when the member answers themselves, so an admin override can
	// never charge differently from a self-service tap.
	delta := sessionDelta(previous, response)
	changed := previous == nil || previous.Response != response

	if delta != 0 {
		if err := b.DB.AddSessionsToUser(targetID, groupID, delta); err != nil {
			zap.L().Error("Error updating sessions from admin override",
				zap.Int64("event_id", eventID), zap.Int64("user_id", targetID), zap.Error(err))
			b.AnswerCallbackQuery(callback.ID, "خطا در ثبت وضعیت.")
			return
		}
	}

	if err := b.DB.SetEventResponse(eventID, targetID, response); err != nil {
		zap.L().Error("Error saving response from admin override",
			zap.Int64("event_id", eventID), zap.Int64("user_id", targetID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در ثبت وضعیت.")
		return
	}

	refreshEventBoard(b, event)

	if changed {
		notifyMemberOfOverride(b, targetID, event, response)
	}

	b.AnswerCallbackQuery(callback.ID, "وضعیت ثبت شد.")
	showMemberList(b, callback.Message.Chat.ID, callback.Message.MessageID, eventID, groupID)
}

// notifyMemberOfOverride tells the member an admin changed their status, since it
// moves their debt without them touching anything.
func notifyMemberOfOverride(b *bot.Bot, userID int64, event *models.Event, response models.EventResponseType) {
	user, err := b.DB.GetUserByID(userID)
	if err != nil {
		zap.L().Error("Error getting user to notify", zap.Int64("user_id", userID), zap.Error(err))
		return
	}

	sendSessionBill(b, user.TelegramID, userID, event, response,
		"ℹ️ ادمین وضعیت شما را برای این سانس ثبت کرد.\n\n")
}
