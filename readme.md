🤖 Telegram Auto Mail Sender Bot (Go)
=====================================

Send emails directly from **Telegram** using this lightweight **Golang** bot.  
No complex setup — just message your bot, and it sends your mail instantly! 🚀

* * *

🌟 Features
-----------

* ✅ Send emails via Telegram instantly
* ✅ Multi-recipient support (send to multiple email addresses at once)
* ✅ Attach files to emails (single file per email)
* ✅ Schedule emails for later delivery (YYYY-MM-DD HH:MM or send immediately)
* ✅ Interactive step-by-step email composer in Telegram
* ✅ Preview email before sending (recipients, subject, body, attachment)
* ✅ Cancel email composition anytime with /cancel
* ✅ View pending scheduled emails with /scheduled
* ✅ Secure .env configuration for mail credentials
* ✅ Supports Gmail, Outlook, Yahoo (SMTP configurable)
* ✅ Beginner-friendly Go project, fully open-source and extendable
* ✅ Works with both text body and attachments
* ✅ Background worker automatically sends scheduled emails
* ✅ Logs success and errors for email sending

⚙️ Setup Guide
--------------

### 🧩 Step 1: Clone the repository

    git clone https://github.com/YOUR_GITHUB_USERNAME/telegram-mail-bot.git
    
    cd telegram-mail-bot
        

### 🧩 Step 2: Install dependencies

    go mod tidy
        

### 🔐 Step 3: Configure your `.env` file

Create a `.env` file in your project root with the following content 👇

#### ✉️ For Gmail

    TELEGRAM_BOT_TOKEN=your_telegram_bot_token
    SMTP_HOST=smtp.gmail.com
    SMTP_PORT=587
    GMAIL_USERNAME=youremail@gmail.com
    GMAIL_PASSWORD=your_16_char_app_password
        

⚠️ You must use a Gmail App Password.  
Normal Gmail passwords will not work — Google blocks “less secure apps.”

#### ✉️ For Outlook

    TELEGRAM_BOT_TOKEN=your_telegram_bot_token
    SMTP_HOST=smtp.office365.com
    SMTP_PORT=587
    GMAIL_USERNAME=youremail@outlook.com
    GMAIL_PASSWORD=your_outlook_password_or_app_password
        

#### ✉️ For Yahoo Mail

    TELEGRAM_BOT_TOKEN=your_telegram_bot_token
    SMTP_HOST=smtp.mail.yahoo.com
    SMTP_PORT=587
    GMAIL_USERNAME=youremail@yahoo.com
    GMAIL_PASSWORD=your_yahoo_app_password
        

💡 You can rename `GMAIL_` variables to `EMAIL_` in your code for a more generic setup.

### 🤖 Step 4: Set Up Your Telegram Bot

1.  Open Telegram and search for **@BotFather**.
2.  Run `/newbot`.
3.  Give it a name and username.
4.  Copy the bot token it gives you.
5.  Paste it in `.env` as `TELEGRAM_BOT_TOKEN=...`.
6.  Start a chat with your bot and press **Start**.

### 🚀 Step 5: Run the Bot

    go run ./cmd/bot

✅ Within seconds, you’ll receive the email in your inbox.

## 🧠 Troubleshooting

| Problem | Solution |
|---------|----------|
| `TELEGRAM_BOT_TOKEN not found` | Check `.env` file or ensure `godotenv.Load()` is called |
| Bot not responding | Start a chat with the bot and press **Start** |
| `GMAIL_USERNAME` or `GMAIL_PASSWORD` not set | Verify your `.env` variables |
| Mail not sending | Use an App Password and check your SMTP host/port |
| Timeout or auth errors | Make sure 2FA is enabled and you used the correct app password |


⚙️ Build a Compiled Version
---------------------------

To create a standalone executable (so others can run it without Go):

### 🧩 Build for your system

    go build -o bot ./cmd/bot
        

Run it:

    ./bot
        

or on Windows:
    
    bot.exe 

### 🧩 Cross-compile for other platforms

| Platform | Command |
|----------|---------|
| Linux | `GOOS=linux GOARCH=amd64 go build -o bot-linux ./cmd/bot` |
| Windows | `GOOS=windows GOARCH=amd64 go build -o bot.exe ./cmd/bot` |
| macOS | `GOOS=darwin GOARCH=amd64 go build -o bot-mac ./cmd/bot` |

Each build will produce a platform-specific binary you can share.

🎉 **Ready to send emails from Telegram like a pro?** Fire up this bot, blast some messages, and let the good vibes (and emails) flow! 🚀😎
