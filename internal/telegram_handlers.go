package main

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	API      *tgbotapi.BotAPI
	sender   string
	password string
	smtpHost string
	smtpPort string
	admins   map[int64]bool
	state    map[int64]*ComposeState
	stateMu  sync.Mutex
}

type ComposeState struct {
	Step        int
	To          string
	Subject     string
	Body        string
	Attachments map[string][]byte
}

func NewBotFromEnv() (*Bot, error) {
	token := os.Getenv("TELEGRAM_TOKEN")
	sender := os.Getenv("EMAIL_SENDER")
	pass := os.Getenv("EMAIL_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	if token == "" || sender == "" || pass == "" || smtpHost == "" || smtpPort == "" {
		return nil, errors.New("missing required .env variables")
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		API:      api,
		sender:   sender,
		password: pass,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		admins:   map[int64]bool{},
		state:    map[int64]*ComposeState{},
	}

	if v := os.Getenv("ADMIN_IDS"); v != "" {
		for _, idStr := range strings.Split(v, ",") {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				b.admins[id] = true
			}
		}
	}

	return b, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.API.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "help":
				b.reply(chatID, "/send <to> <subject> <body>\\n/compose\\n/cancel")
			case "send":
				b.handleSend(update.Message)
			case "compose":
				b.startCompose(chatID)
			case "cancel":
				b.cancelCompose(chatID)
			}
			continue
		}

		b.stateMu.Lock()
		state, active := b.state[chatID]
		b.stateMu.Unlock()

		if active {
			b.handleCompose(chatID, update.Message, state)
		}
	}
}

func (b *Bot) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.API.Send(msg)
}

func (b *Bot) handleSend(msg *tgbotapi.Message) {
	if !b.isAllowed(msg.Chat.ID) {
		b.reply(msg.Chat.ID, "Not authorized")
		return
	}

	args := strings.SplitN(msg.CommandArguments(), " ", 3)
	if len(args) < 3 {
		b.reply(msg.Chat.ID, "Usage: /send <to> <subject> <body>")
		return
	}

	to, subject, body := args[0], args[1], args[2]
	emailMsg, err := buildMessage(b.sender, to, subject, body, nil)
	if err != nil {
		b.reply(msg.Chat.ID, "Build error: "+err.Error())
		return
	}

	err = sendMail(b.smtpHost, b.smtpPort, b.sender, b.password, []string{to}, emailMsg)
	if err != nil {
		b.reply(msg.Chat.ID, "Send error: "+err.Error())
		return
	}
	b.reply(msg.Chat.ID, "✅ Email sent!")
}

func (b *Bot) isAllowed(id int64) bool {
	if len(b.admins) == 0 {
		return true
	}
	return b.admins[id]
}

func (b *Bot) startCompose(chatID int64) {
	b.stateMu.Lock()
	b.state[chatID] = &ComposeState{Step: 1, Attachments: map[string][]byte{}}
	b.stateMu.Unlock()
	b.reply(chatID, "Recipient email address?")
}

func (b *Bot) cancelCompose(chatID int64) {
	b.stateMu.Lock()
	delete(b.state, chatID)
	b.stateMu.Unlock()
	b.reply(chatID, "Cancelled.")
}

func (b *Bot) handleCompose(chatID int64, msg *tgbotapi.Message, st *ComposeState) {
	switch st.Step {
	case 1:
		st.To = msg.Text
		st.Step = 2
		b.reply(chatID, "Subject?")
	case 2:
		st.Subject = msg.Text
		st.Step = 3
		b.reply(chatID, "Body? (Send /done when finished)")
	case 3:
		if msg.IsCommand() && msg.Command() == "done" {
			b.sendComposed(chatID, st)
			b.stateMu.Lock()
			delete(b.state, chatID)
			b.stateMu.Unlock()
			return
		}
		st.Body += msg.Text + "\n"
		b.reply(chatID, "(Body updated — /done when finished)")
	}
}

func (b *Bot) sendComposed(chatID int64, st *ComposeState) {
	emailMsg, err := buildMessage(b.sender, st.To, st.Subject, st.Body, st.Attachments)
	if err != nil {
		b.reply(chatID, "Build error: "+err.Error())
		return
	}

	err = sendMail(b.smtpHost, b.smtpPort, b.sender, b.password, []string{st.To}, emailMsg)
	if err != nil {
		b.reply(chatID, "Send error: "+err.Error())
		return
	}
	b.reply(chatID, "✅ Email sent successfully!")
}
