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

// Guests are people with no account. Any registered member adds them to one
// session by name; they show up in the present list marked as guests. A guest
// costs the member who added them one session, charged at that member's own role
// rate — there is no separate guest role or guest price.

// guestMember checks that the caller is a registered member of the group, which
// is required to add or remove guests: the charge lands on their user_groups row,
// so someone without one could add guests for free.
func guestMember(b *bot.Bot, callback *tgbotapi.CallbackQuery, groupID int64) *models.User {
	user, err := b.DB.GetUserByTelegramID(callback.From.ID)
	if err != nil {
		b.AnswerCallbackQuery(callback.ID, "خطا در دریافت اطلاعات کاربر.")
		return nil
	}

	isMember, err := b.DB.IsUserMemberOfGroup(user.ID, groupID)
	if err != nil || !isMember {
		b.AnswerCallbackQuery(callback.ID, "برای افزودن مهمان باید در این گروه ثبت نام کنید.")
		return nil
	}

	return user
}

// handleGuestsCallback lists the open sessions to manage guests for.
func handleGuestsCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 2 {
		return
	}

	groupID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	if guestMember(b, callback, groupID) == nil {
		return
	}

	events, err := b.DB.GetOpenEvents(groupID)
	if err != nil {
		zap.L().Error("Error getting open events for guests", zap.Int64("group_id", groupID), zap.Error(err))
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
				fmt.Sprintf("guest_menu:%d:%d", e.ID, groupID)),
		})
	}
	rows = append(rows, backRow)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "مهمان‌های کدام سانس؟", &keyboard)
}

// showGuestMenu draws the add/remove menu for one session. A member only gets a
// remove button for the guests they added themselves, because removing gives the
// session back to whoever paid for it. Admins may remove anyone's guest.
func showGuestMenu(b *bot.Bot, chatID int64, messageID int, eventID, groupID, viewerID int64, isAdmin bool) {
	event, err := b.DB.GetEventByID(eventID)
	if err != nil {
		zap.L().Error("Error getting event for guest menu", zap.Int64("event_id", eventID), zap.Error(err))
		return
	}

	guests, err := b.DB.GetEventGuests(eventID)
	if err != nil {
		zap.L().Error("Error getting guests", zap.Int64("event_id", eventID), zap.Error(err))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("➕ افزودن مهمان", fmt.Sprintf("add_guest:%d:%d", eventID, groupID)),
	})

	mine := 0
	for _, g := range guests {
		if g.AddedBy == viewerID {
			mine++
		}
		if g.AddedBy != viewerID && !isAdmin {
			continue
		}
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("❌ حذف %s", g.Name),
				fmt.Sprintf("del_guest:%d:%d:%d", g.ID, eventID, groupID)),
		})
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("guests:%d", groupID)),
	})

	text := fmt.Sprintf(
		"سانس %s %s\n\nکل مهمان‌ها: %d\nمهمان‌های شما: %d\n\nهر مهمان یک جلسه به حساب شما اضافه می‌کند.",
		event.SessionDate, event.Month, len(guests), mine)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	b.EditMessage(chatID, messageID, text, &keyboard)
}

func handleGuestMenuCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
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

	user := guestMember(b, callback, groupID)
	if user == nil {
		return
	}

	showGuestMenu(b, callback.Message.Chat.ID, callback.Message.MessageID,
		eventID, groupID, user.ID, isAdminFor(b, callback.From.ID, groupID))
}

func handleAddGuestCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
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

	if guestMember(b, callback, groupID) == nil {
		return
	}

	b.SetState(callback.From.ID, "awaiting_guest_name", map[string]interface{}{
		"event_id": eventID,
		"group_id": groupID,
	})

	b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "نام مهمان را وارد کنید:", nil)
}

func handleGuestNameInput(b *bot.Bot, message *tgbotapi.Message, state *models.UserState) {
	name := strings.TrimSpace(message.Text)
	if name == "" {
		b.SendMessage(message.Chat.ID, "لطفا یک نام معتبر وارد کنید:", nil)
		return
	}

	eventID := state.TempData["event_id"].(int64)
	groupID := state.TempData["group_id"].(int64)

	b.ClearState(message.From.ID)

	user, err := b.DB.GetUserByTelegramID(message.From.ID)
	if err != nil {
		zap.L().Error("Error getting user for guest", zap.Int64("telegram_id", message.From.ID), zap.Error(err))
		b.SendMessage(message.Chat.ID, "خطا در دریافت اطلاعات کاربر.", nil)
		return
	}

	if err := b.DB.AddEventGuest(eventID, user.ID, groupID, name); err != nil {
		zap.L().Error("Error adding guest", zap.Int64("event_id", eventID), zap.Error(err))
		b.SendMessage(message.Chat.ID, "خطا در افزودن مهمان.", nil)
		return
	}

	event, err := b.DB.GetEventByID(eventID)
	if err == nil {
		refreshEventBoard(b, event)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧑‍🤝‍🧑 مهمان‌های این سانس", fmt.Sprintf("guest_menu:%d:%d", eventID, groupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("back:%d", groupID)),
		),
	)

	text := fmt.Sprintf("✅ مهمان «%s» اضافه شد.\nیک جلسه به حساب شما اضافه شد.", name)

	if ug, err := b.DB.GetUserGroup(user.ID, groupID); err == nil {
		rate, _ := b.DB.GetRate(groupID, ug.Role)
		text += fmt.Sprintf("\n\nمجموع جلسات شما: %d\nمجموع بدهی: %.0f تومان",
			ug.SessionsOwed, float64(ug.SessionsOwed)*rate)
	}

	b.SendMessage(message.Chat.ID, text, keyboard)
}

func handleDeleteGuestCallback(b *bot.Bot, callback *tgbotapi.CallbackQuery, parts []string) {
	if len(parts) < 4 {
		return
	}

	guestID, err := strconv.ParseInt(parts[1], 10, 64)
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

	user := guestMember(b, callback, groupID)
	if user == nil {
		return
	}

	guest, err := b.DB.GetEventGuestByID(guestID)
	if err != nil {
		b.AnswerCallbackQuery(callback.ID, "این مهمان پیدا نشد.")
		return
	}

	isAdmin := isAdminFor(b, callback.From.ID, groupID)
	if guest.AddedBy != user.ID && !isAdmin {
		b.AnswerCallbackQuery(callback.ID, "فقط کسی که مهمان را اضافه کرده می‌تواند حذفش کند.")
		return
	}

	if err := b.DB.DeleteEventGuest(guestID, groupID); err != nil {
		zap.L().Error("Error deleting guest", zap.Int64("guest_id", guestID), zap.Error(err))
		b.AnswerCallbackQuery(callback.ID, "خطا در حذف مهمان.")
		return
	}

	event, err := b.DB.GetEventByID(eventID)
	if err == nil {
		refreshEventBoard(b, event)
	}

	b.AnswerCallbackQuery(callback.ID, "مهمان حذف شد.")
	showGuestMenu(b, callback.Message.Chat.ID, callback.Message.MessageID, eventID, groupID, user.ID, isAdmin)
}
