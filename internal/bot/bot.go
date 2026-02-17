package bot

import (
	"fmt"
	"futsal-bot/internal/database"
	"futsal-bot/internal/models"
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	API            *tgbotapi.BotAPI
	DB             *database.DB
	DefaultAdminID int64
	States         map[int64]*models.UserState
	StatesMutex    sync.RWMutex
}

func New(token string, db *database.DB, defaultAdminID int64) (*Bot, error) {
	// api, err := tgbotapi.NewBotAPI(token)
	api, err := tgbotapi.NewBotAPIWithAPIEndpoint(token, "https://tapi.bale.ai/bot%s/%s")
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	return &Bot{
		API:            api,
		DB:             db,
		DefaultAdminID: defaultAdminID,
		States:         make(map[int64]*models.UserState),
	}, nil
}

func (b *Bot) SetState(userID int64, state string, data map[string]interface{}) {
	b.StatesMutex.Lock()
	defer b.StatesMutex.Unlock()

	b.States[userID] = &models.UserState{
		UserID:   userID,
		State:    state,
		TempData: data,
	}
}

func (b *Bot) GetState(userID int64) *models.UserState {
	b.StatesMutex.RLock()
	defer b.StatesMutex.RUnlock()

	return b.States[userID]
}

func (b *Bot) ClearState(userID int64) {
	b.StatesMutex.Lock()
	defer b.StatesMutex.Unlock()

	delete(b.States, userID)
}

func (b *Bot) IsDefaultAdmin(userID int64) bool {
	return userID == b.DefaultAdminID
}

func (b *Bot) SendMessage(chatID int64, text string, replyMarkup interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if replyMarkup != nil {
		msg.ReplyMarkup = replyMarkup
	}

	_, err := b.API.Send(msg)
	return err
}

func (b *Bot) SendMessageWithMarkdown(chatID int64, text string, replyMarkup interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if replyMarkup != nil {
		msg.ReplyMarkup = replyMarkup
	}

	_, err := b.API.Send(msg)
	return err
}

func (b *Bot) EditMessage(chatID int64, messageID int, text string, replyMarkup interface{}) error {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if replyMarkup != nil {
		if markup, ok := replyMarkup.(*tgbotapi.InlineKeyboardMarkup); ok {
			msg.ReplyMarkup = markup
		}
	}

	_, err := b.API.Send(msg)
	return err
}

func (b *Bot) AnswerCallbackQuery(callbackID string, text string) error {
	callback := tgbotapi.NewCallback(callbackID, text)
	_, err := b.API.Request(callback)
	return err
}

// Keyboard builders
func (b *Bot) MainMenuKeyboard(userID, groupID int64, isAdmin bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Check if user is registered in this group
	ug, err := b.DB.GetUserGroup(userID, groupID)

	if err != nil || ug == nil {
		// Not registered - show register button
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📝 ثبت نام", fmt.Sprintf("register:%d", groupID)),
		})
	} else {
		// Registered - show edit and invoice buttons
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✏️ ویرایش مشخصات", fmt.Sprintf("edit:%d", groupID)),
		})
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("💰 صورتحساب", fmt.Sprintf("invoice:%d", groupID)),
		})
	}

	if isAdmin {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("💵 تعیین نرخ", fmt.Sprintf("set_rates:%d", groupID)),
		})
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✅ تسویه حساب کاربر", fmt.Sprintf("settle:%d", groupID)),
		})
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) RoleSelectionKeyboard(groupID int64, isAdmin bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🎓 دانشجو", fmt.Sprintf("role:student:%d", groupID)),
	})
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("👤 بزرگسال", fmt.Sprintf("role:adult:%d", groupID)),
	})
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("👦 نیمه بزرگسال", fmt.Sprintf("role:half_adult:%d", groupID)),
	})

	if isAdmin {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👑 ادمین", fmt.Sprintf("role:admin:%d", groupID)),
		})
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) RateSettingKeyboard(groupID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎓 دانشجو", fmt.Sprintf("setrate:student:%d", groupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 بزرگسال", fmt.Sprintf("setrate:adult:%d", groupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👦 نیمه بزرگسال", fmt.Sprintf("setrate:half_adult:%d", groupID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("back:%d", groupID)),
		),
	)
}
