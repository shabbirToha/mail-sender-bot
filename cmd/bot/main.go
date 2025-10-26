package main

import (
	"log"

	"github.com/shabbirtoha/telegram-mail-bot/internal/bot"
)

func main() {
	b, err := bot.NewBotFromEnv()
	if err != nil {
		log.Fatalf("failed to initialize bot: %v", err)
	}

	// start scheduler worker (reads scheduled emails from DB and sends them)
	go b.StartScheduledWorker()

	log.Println("🤖 Bot is running...")
	b.Start()
}
