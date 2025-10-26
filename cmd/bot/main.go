package main

import (
	"log"

	"github.com/shabbirtoha/telegram-mail-bot/internal/bot" // change this to your module path

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	// Start the bot
	b, err := bot.NewBotFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("🤖 Bot is running...")
	b.Tele.Start()
}
