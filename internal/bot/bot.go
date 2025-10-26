package bot

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/shabbirtoha/telegram-mail-bot/internal/mail"

	"gopkg.in/telebot.v3"
)

// Bot wraps the Telebot instance.
type Bot struct {
	Tele *telebot.Bot
}

// EmailSession tracks a user's email composition progress.
type EmailSession struct {
	Step      int
	Recipient string
	Subject   string
	Body      string
}

// Global in-memory session storage
var sessions = struct {
	sync.RWMutex
	data map[int64]*EmailSession
}{data: make(map[int64]*EmailSession)}

// NewBotFromEnv initializes the Telegram bot using environment variables.
func NewBotFromEnv() (*Bot, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN not found in environment")
	}

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	bot := &Bot{Tele: b}
	bot.registerHandlers()

	return bot, nil
}

// registerHandlers sets up bot commands and text message handling.
func (b *Bot) registerHandlers() {
	// /start command
	b.Tele.Handle("/start", func(c telebot.Context) error {
		return c.Send("👋 Hello! I'm your Mail Sender Bot.\nUse /help to see what I can do.")
	})

	// /help command
	b.Tele.Handle("/help", func(c telebot.Context) error {
		return c.Send("📬 Commands:\n/start — greet the bot\n/help — show help\n/sendmail — send an email")
	})

	// /sendmail command starts the email composition flow
	b.Tele.Handle("/sendmail", func(c telebot.Context) error {
		userID := c.Sender().ID

		sessions.Lock()
		sessions.data[userID] = &EmailSession{Step: 1}
		sessions.Unlock()

		return c.Send("📨 Please enter recipient email address:")
	})

	// Handles user text input for all email steps
	b.Tele.Handle(telebot.OnText, func(c telebot.Context) error {
		userID := c.Sender().ID
		msg := c.Message().Text

		sessions.RLock()
		session, exists := sessions.data[userID]
		sessions.RUnlock()

		// No active session
		if !exists {
			return c.Send("💬 Send /sendmail to start composing a new email.")
		}

		switch session.Step {
		case 1:
			session.Recipient = msg
			session.Step = 2
			return c.Send("✏️ Now enter your email subject:")

		case 2:
			session.Subject = msg
			session.Step = 3
			return c.Send("📝 Now enter your email body:")

		case 3:
			session.Body = msg
			c.Send("📤 Sending your email...")

			err := mail.SendMail(session.Recipient, session.Subject, session.Body)
			if err != nil {
				c.Send("❌ Failed to send mail: " + err.Error())
			} else {
				c.Send("✅ Mail sent successfully to " + session.Recipient)
			}

			// Clear session after sending
			sessions.Lock()
			delete(sessions.data, userID)
			sessions.Unlock()

			return nil

		default:
			return c.Send("⚠️ Unexpected input. Send /sendmail to start over.")
		}
	})
}
