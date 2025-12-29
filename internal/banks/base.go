package banks

import (
	"PaymentsBot/internal/tg"
)

var TgBot *tg.TelegramService

func SetTelegram(telegram *tg.TelegramService) {
	TgBot = telegram
}
