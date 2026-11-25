package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func mainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить смену"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🗓 Календарь смен"),
			tgbotapi.NewKeyboardButton("📊 Статистика"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📦 Дополнительные работы"),
			tgbotapi.NewKeyboardButton("💰 Указать ставку"),
		),
	)

	keyboard.ResizeKeyboard = true
	return keyboard
}

func incomePeriodKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📅 Текущий месяц")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("↩ Прошлый месяц")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🗓 Свой период")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🏠 Главное меню")),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}
