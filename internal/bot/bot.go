package bot

import (
	"log"
	"time"

	"tg_bot/internal/repository"
	"tg_bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserState struct {
	ShiftID     int
	Step        string
	StartTime   time.Time
	EndTime     time.Time
	ReportStart time.Time
}

type Bot struct {
	api                  *tgbotapi.BotAPI
	userRepo             *repository.UserRepository
	userStates           map[int64]*UserState
	shiftService         *service.ShiftService
	salaryService        *service.SalaryService
	extraRepo            *repository.ExtraRepository
	activeInlineMessages map[int64]int
}

func New(
	api *tgbotapi.BotAPI,
	userRepo *repository.UserRepository,
	shiftService *service.ShiftService,
	salaryService *service.SalaryService,
	extraRepo *repository.ExtraRepository,
) *Bot {
	return &Bot{
		api:                  api,
		userRepo:             userRepo,
		shiftService:         shiftService,
		salaryService:        salaryService,
		extraRepo:            extraRepo,
		userStates:           make(map[int64]*UserState),
		activeInlineMessages: make(map[int64]int),
	}
}

// Убирает inline-клавиатуру и забывает активное сообщение чата.
func (b *Bot) clearActiveInlineKeyboard(chatID int64) {
	messageID, exists := b.activeInlineMessages[chatID]
	if !exists {
		return
	}

	markup := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
	}

	edit := tgbotapi.NewEditMessageReplyMarkup(
		chatID,
		messageID,
		markup,
	)

	if _, err := b.api.Request(edit); err != nil {
		log.Println("не удалось убрать inline-клавиатуру:", err)
	}

	delete(b.activeInlineMessages, chatID)
}

func (b *Bot) isActiveInlineMessage(
	chatID int64,
	messageID int,
) bool {
	activeID, exists := b.activeInlineMessages[chatID]

	return exists && activeID == messageID
}

// Завершает текущий сценарий перед переходом в меню.
func (b *Bot) resetUserFlow(chatID int64) {
	b.clearActiveInlineKeyboard(chatID)
	delete(b.userStates, chatID)
}

// Отвечает на нажатие inline-кнопки и записывает ошибку, если ответ не отправился.
func (b *Bot) answerCallback(callbackID, text string) {
	answer := tgbotapi.NewCallback(callbackID, text)

	if _, err := b.api.Request(answer); err != nil {
		log.Println("не удалось обработать callback:", err)
	}
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	text := "Привет! Я бот для учёта смен.\n\nПеред добавлением первой смены сначала укажи свою часовую ставку через кнопку «💰 Указать ставку». Она нужна для расчёта зарплаты. После этого нажми «➕ Добавить смену».\n\nВыбери действие:"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		text,
	)

	msg.ReplyMarkup = mainKeyboard()

	if _, err := b.api.Send(msg); err != nil {
		log.Println("Ошибка отправки сообщения:", err)
	}
}
