package bot

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

const (
	attachmentsDir = "attachments"
	sqliteFile     = "botdata.db"
	pollInterval   = time.Minute
)

// EmailSession stores temporary email composition data per chat
type EmailSession struct {
	Step      int
	To        string
	Subject   string
	Body      string
	FilePath  string
	FileName  string
	Schedule  string
	ChatID    int64
	CreatedAt time.Time
}

// Bot is the main bot struct
type Bot struct {
	API      *tgbotapi.BotAPI
	SMTPHost string
	SMTPPort int
	Username string
	Password string

	sessions   map[int64]*EmailSession
	sessionsMu sync.RWMutex

	db   *sql.DB
	dbMu sync.Mutex
}

// NewBotFromEnv loads env and initializes the bot
func NewBotFromEnv() (*Bot, error) {
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}
	smtpPortStr := os.Getenv("SMTP_PORT")
	if smtpPortStr == "" {
		smtpPortStr = "587"
	}
	smtpPort, _ := strconv.Atoi(smtpPortStr)

	username := os.Getenv("GMAIL_USERNAME")
	password := os.Getenv("GMAIL_PASSWORD")
	if username == "" || password == "" {
		return nil, fmt.Errorf("GMAIL_USERNAME and GMAIL_PASSWORD are required")
	}

	if err := os.MkdirAll(attachmentsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create attachments dir: %v", err)
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot api: %w", err)
	}

	db, err := sql.Open("sqlite3", sqliteFile)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := initDB(db); err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	return &Bot{
		API:      api,
		SMTPHost: smtpHost,
		SMTPPort: smtpPort,
		Username: username,
		Password: password,
		sessions: make(map[int64]*EmailSession),
		db:       db,
	}, nil
}

// initDB creates scheduled emails table
func initDB(db *sql.DB) error {
	create := `
	CREATE TABLE IF NOT EXISTS scheduled_emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER,
		recipients TEXT,
		subject TEXT,
		body TEXT,
		attachments_json TEXT,
		send_at TEXT,
		status TEXT,
		created_at TEXT
	);
	`
	_, err := db.Exec(create)
	return err
}

// Start listens to Telegram updates
func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.API.GetUpdatesChan(u)

	log.Printf("authorized on account %s", b.API.Self.UserName)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		msg := update.Message

		if b.hasSession(msg.Chat.ID) && !msg.IsCommand() {
			if msg.Document != nil || len(msg.Photo) > 0 {
				b.handleAttachment(msg)
			} else {
				b.handleConversation(msg)
			}
			continue
		}

		if msg.IsCommand() {
			switch msg.Command() {
			case "start":
				b.cmdStart(msg)
			case "help":
				b.cmdHelp(msg)
			case "sendmail":
				b.cmdSendMail(msg)
			case "scheduled":
				b.cmdListScheduled(msg)
			case "cancel":
				b.cmdCancelSession(msg)
			default:
				b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "Unknown command. Use /help"))
			}
			continue
		}

		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "Hello! Use /sendmail to start composing an email."))
	}
}

// ---------- Commands ----------

func (b *Bot) cmdStart(msg *tgbotapi.Message) {
	text := "👋 *Telegram Mail Wizard*\n\n" +
		"Type `/sendmail` to start sending an email step-by-step.\n" +
		"You can attach one file and optionally schedule delivery.\n\n" +
		"Use `/scheduled` to list pending scheduled emails."
	m := tgbotapi.NewMessage(msg.Chat.ID, text)
	m.ParseMode = "Markdown"
	b.API.Send(m)
}

func (b *Bot) cmdHelp(msg *tgbotapi.Message) {
	text := "ℹ️ *Commands*\n\n" +
		"/sendmail - start interactive email composer\n" +
		"/scheduled - list pending scheduled emails\n" +
		"/cancel - cancel current compose session\n\n" +
		"Interactive flow will ask: recipient(s), subject, body, attachment (optional), schedule (now or `YYYY-MM-DD HH:MM`)."
	m := tgbotapi.NewMessage(msg.Chat.ID, text)
	m.ParseMode = "Markdown"
	b.API.Send(m)
}

func (b *Bot) cmdSendMail(msg *tgbotapi.Message) {
	s := &EmailSession{
		Step:      1,
		ChatID:    msg.Chat.ID,
		CreatedAt: time.Now().UTC(),
	}
	b.setSession(msg.Chat.ID, s)
	b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "📬 Who do you want to send the email to? (comma separated addresses are allowed)"))
}

func (b *Bot) cmdCancelSession(msg *tgbotapi.Message) {
	if b.hasSession(msg.Chat.ID) {
		b.deleteSession(msg.Chat.ID)
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Session cancelled."))
	} else {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "No active session to cancel."))
	}
}

func (b *Bot) cmdListScheduled(msg *tgbotapi.Message) {
	rows, err := b.db.Query("SELECT id, recipients, subject, send_at, status FROM scheduled_emails WHERE chat_id = ? ORDER BY created_at DESC LIMIT 20", msg.Chat.ID)
	if err != nil {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "Failed to query scheduled emails: "+err.Error()))
		return
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var id int
		var recipients, subject, sendAt, status string
		_ = rows.Scan(&id, &recipients, &subject, &sendAt, &status)
		lines = append(lines, fmt.Sprintf("ID:%d — to:%s — at:%s — %s", id, recipients, sendAt, status))
	}
	if len(lines) == 0 {
		b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, "No scheduled emails found."))
		return
	}
	b.API.Send(tgbotapi.NewMessage(msg.Chat.ID, strings.Join(lines, "\n")))
}

// ---------- Session helpers ----------

func (b *Bot) setSession(chatID int64, s *EmailSession) {
	b.sessionsMu.Lock()
	defer b.sessionsMu.Unlock()
	b.sessions[chatID] = s
}

func (b *Bot) getSession(chatID int64) (*EmailSession, bool) {
	b.sessionsMu.RLock()
	defer b.sessionsMu.RUnlock()
	s, ok := b.sessions[chatID]
	return s, ok
}

func (b *Bot) deleteSession(chatID int64) {
	b.sessionsMu.Lock()
	defer b.sessionsMu.Unlock()
	delete(b.sessions, chatID)
}

func (b *Bot) hasSession(chatID int64) bool {
	b.sessionsMu.RLock()
	defer b.sessionsMu.RUnlock()
	_, ok := b.sessions[chatID]
	return ok
}

// ---------- Conversation flow ----------

func (b *Bot) handleConversation(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	session, ok := b.getSession(chatID)
	if !ok {
		b.API.Send(tgbotapi.NewMessage(chatID, "No active session. Use /sendmail to start."))
		return
	}
	text := strings.TrimSpace(msg.Text)
	lower := strings.ToLower(text)

	switch session.Step {
	case 1: // recipients
		session.To = text
		session.Step = 2
		b.API.Send(tgbotapi.NewMessage(chatID, "✏️ Subject?"))
	case 2: // subject
		session.Subject = text
		session.Step = 3
		b.API.Send(tgbotapi.NewMessage(chatID, "📝 Body text (send a single message or multiple; type /done to finish):"))
	case 3: // body
		session.Body = text
		session.Step = 4
		b.API.Send(tgbotapi.NewMessage(chatID, "📎 Do you want to attach a file? Reply `yes` to attach or `no` to skip."))
	case 4:
		if lower == "yes" {
			session.Step = 5
			b.API.Send(tgbotapi.NewMessage(chatID, "📂 Please upload the file now (send as document)."))
		} else if lower == "no" {
			session.Step = 6
			b.sendPreview(chatID, session)
		} else {
			b.API.Send(tgbotapi.NewMessage(chatID, "Please reply with `yes` or `no`."))
		}
	case 5:
		b.API.Send(tgbotapi.NewMessage(chatID, "Waiting for file upload. Send document or type `skip`."))
		if lower == "skip" {
			session.Step = 6
			b.sendPreview(chatID, session)
		}
	case 6:
		if lower == "now" || lower == "send now" || lower == "send" {
			b.API.Send(tgbotapi.NewMessage(chatID, "📤 Sending now..."))
			if err := b.sendMailMulti(session); err != nil {
				b.API.Send(tgbotapi.NewMessage(chatID, "Failed to send: "+err.Error()))
			} else {
				b.API.Send(tgbotapi.NewMessage(chatID, "✅ Email sent!"))
			}
			b.deleteSession(chatID)
		} else {
			session.Schedule = text
			if err := b.schedulePersist(session); err != nil {
				b.API.Send(tgbotapi.NewMessage(chatID, "Failed to schedule: "+err.Error()))
			} else {
				b.API.Send(tgbotapi.NewMessage(chatID, "⏰ Email scheduled successfully!"))
			}
			b.deleteSession(chatID)
		}
	default:
		b.API.Send(tgbotapi.NewMessage(chatID, "Unknown session state. Use /cancel and try again."))
	}
}

// sendPreview shows summary before asking schedule/send
func (b *Bot) sendPreview(chatID int64, session *EmailSession) {
	attach := "No"
	if session.FileName != "" {
		attach = session.FileName
	}
	preview := fmt.Sprintf(
		"📬 *Preview*\nTo: %s\nSubject: %s\nBody: %s\nAttachment: %s\n\nType `now` to send immediately or provide time `YYYY-MM-DD HH:MM` to schedule.",
		session.To, session.Subject, session.Body, attach,
	)
	msg := tgbotapi.NewMessage(chatID, preview)
	msg.ParseMode = "Markdown"
	b.API.Send(msg)
	session.Step = 6
}

// ---------- Mail sending ----------

func (b *Bot) sendMailMulti(session *EmailSession) error {
	toList := strings.Split(session.To, ",")
	for _, t := range toList {
		if err := b.sendMail(strings.TrimSpace(t), session.Subject, session.Body, session.FilePath, session.FileName); err != nil {
			return err
		}
	}
	return nil
}

// sendMail sends email with optional attachment
func (b *Bot) sendMail(to, subject, body, filePath, filename string) error {
	from := b.Username
	msg := bytes.Buffer{}
	boundary := "BOUNDARY-12345"

	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if filePath != "" {
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary))
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		msg.WriteString(body + "\r\n\r\n")

		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: application/octet-stream\r\n")
		msg.WriteString("Content-Transfer-Encoding: base64\r\n")
		msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", filename))
		msg.WriteString(base64.StdEncoding.EncodeToString(content))
		msg.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	} else {
		msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		msg.WriteString(body)
	}

	auth := smtp.PlainAuth("", b.Username, b.Password, b.SMTPHost)
	addr := fmt.Sprintf("%s:%d", b.SMTPHost, b.SMTPPort)
	return smtp.SendMail(addr, auth, from, []string{to}, msg.Bytes())
}

// ---------- Scheduling ----------

func (b *Bot) schedulePersist(session *EmailSession) error {
	b.dbMu.Lock()
	defer b.dbMu.Unlock()

	attJSON := "[]"
	if session.FileName != "" {
		arr := []map[string]string{{"name": session.FileName, "path": session.FilePath}}
		j, _ := json.Marshal(arr)
		attJSON = string(j)
	}

	_, err := b.db.Exec(`INSERT INTO scheduled_emails 
	(chat_id, recipients, subject, body, attachments_json, send_at, status, created_at) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ChatID, session.To, session.Subject, session.Body, attJSON, session.Schedule, "pending", time.Now().UTC().Format(time.RFC3339))
	return err
}

// ---------- Attachment ----------

func (b *Bot) handleAttachment(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	session, ok := b.getSession(chatID)
	if !ok {
		b.API.Send(tgbotapi.NewMessage(chatID, "No active session."))
		return
	}
	doc := msg.Document
	if doc == nil {
		b.API.Send(tgbotapi.NewMessage(chatID, "Please send a document."))
		return
	}
	fileID := doc.FileID
	file, err := b.API.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		b.API.Send(tgbotapi.NewMessage(chatID, "Failed to get file info: "+err.Error()))
		return
	}
	localPath := filepath.Join(attachmentsDir, doc.FileName)
	url := file.Link(b.API.Token)
	resp, err := http.Get(url)
	if err != nil {
		b.API.Send(tgbotapi.NewMessage(chatID, "Failed to download file: "+err.Error()))
		return
	}
	defer resp.Body.Close()
	out, err := os.Create(localPath)
	if err != nil {
		b.API.Send(tgbotapi.NewMessage(chatID, "Failed to create local file: "+err.Error()))
		return
	}
	defer out.Close()
	io.Copy(out, resp.Body)

	session.FilePath = localPath
	session.FileName = doc.FileName
	session.Step = 6
	b.sendPreview(chatID, session)
}

// StartScheduledWorker periodically sends scheduled emails
func (b *Bot) StartScheduledWorker() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		b.dbMu.Lock()
		rows, err := b.db.Query("SELECT id, chat_id, recipients, subject, body, attachments_json, send_at FROM scheduled_emails WHERE status = 'pending'")
		if err != nil {
			b.dbMu.Unlock()
			log.Println("ScheduledWorker query error:", err)
			continue
		}

		var idsToMark []int
		for rows.Next() {
			var id int
			var chatID int64
			var recipients, subject, body, attachmentsJSON, sendAt string
			_ = rows.Scan(&id, &chatID, &recipients, &subject, &body, &attachmentsJSON, &sendAt)

			sendTime, err := time.Parse("2006-01-02 15:04", sendAt)
			if err != nil {
				sendTime, _ = time.Parse(time.RFC3339, sendAt)
			}
			if time.Now().After(sendTime) {
				// parse attachments
				var attachments []map[string]string
				_ = json.Unmarshal([]byte(attachmentsJSON), &attachments)

				for _, to := range strings.Split(recipients, ",") {
					to = strings.TrimSpace(to)
					if err := b.sendMailWithAttachments(to, subject, body, attachments); err != nil {
						log.Println("Scheduled sendMail error:", err)
					}
				}
				idsToMark = append(idsToMark, id)
			}
		}
		rows.Close()

		// mark sent
		for _, id := range idsToMark {
			_, _ = b.db.Exec("UPDATE scheduled_emails SET status='sent' WHERE id=?", id)
		}
		b.dbMu.Unlock()
	}
}

// helper for sending attachments from DB
func (b *Bot) sendMailWithAttachments(to, subject, body string, attachments []map[string]string) error {
	if len(attachments) > 0 {
		for _, att := range attachments {
			err := b.sendMail(to, subject, body, att["path"], att["name"])
			if err != nil {
				return err
			}
		}
		return nil
	}
	return b.sendMail(to, subject, body, "", "")
}
