🤖 Telegram Auto Mail Sender Bot (Go)
=====================================

Send emails directly from **Telegram** using this lightweight **Golang** bot.  
No complex setup — just message your bot, and it sends your mail instantly! 🚀

* * *

🌟 Features
-----------

*   ✅ Send emails via Telegram instantly
*   ✅ Secure `.env` configuration for mail credentials
*   ✅ Supports **Gmail**, **Outlook**, and **Yahoo**
*   ✅ Simple setup, beginner-friendly Go project
*   ✅ Fully open-source and extendable

🗂️ Project Structure
---------------------

    
    telegram-mail-bot/
    ├── cmd/
    │   └── bot/
    │       └── main.go          # Entry point
    ├── internal/
    │   └── bot/
    │       └── bot.go           # Telegram & Mail logic
    ├── .env.example             # Example configuration
    ├── .env                     # Your secrets (ignored by git)
    ├── go.mod
    ├── go.sum
    └── README.md
        

⚙️ Setup Guide
--------------

### 🧩 Step 1: Clone the repository

    bash
    git clone https://github.com/YOUR_GITHUB_USERNAME/telegram-mail-bot.git
    cd telegram-mail-bot
        

### 🧩 Step 2: Install dependencies

    bash
    go mod tidy
        

### 🔐 Step 3: Configure your `.env` file

Create a `.env` file in your project root with the following content 👇

#### ✉️ For Gmail

    bash
    TELEGRAM_BOT_TOKEN=your_telegram_bot_token
    SMTP_HOST=smtp.gmail.com
    SMTP_PORT=587
    GMAIL_USERNAME=youremail@gmail.com
    GMAIL_PASSWORD=your_16_char_app_password
        

⚠️ You must use a Gmail App Password.  
Normal Gmail passwords will not work — Google blocks “less secure apps.”

#### ✉️ For Outlook

    bash
    TELEGRAM_BOT_TOKEN=your_telegram_bot_token
    SMTP_HOST=smtp.office365.com
    SMTP_PORT=587
    GMAIL_USERNAME=youremail@outlook.com
    GMAIL_PASSWORD=your_outlook_password_or_app_password
        

#### ✉️ For Yahoo Mail

    bash
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

    bash
    go run ./cmd/bot
        

If everything is set up correctly, you’ll see:

    
    🤖 Bot is running...
        

Now, in Telegram, send a message in this format:

    
    to: example@gmail.com
    subject: Test Email
    body: Hello from my Go Telegram bot!
        

✅ Within seconds, you’ll receive the email in your inbox.

## 🧠 Troubleshooting

| Problem | Solution |
|---------|----------|
| `TELEGRAM_BOT_TOKEN not found` | Check `.env` file or ensure `godotenv.Load()` is called |
| Bot not responding | Start a chat with the bot and press **Start** |
| `GMAIL_USERNAME` or `GMAIL_PASSWORD` not set | Verify your `.env` variables |
| Mail not sending | Use an App Password and check your SMTP host/port |
| Timeout or auth errors | Make sure 2FA is enabled and you used the correct app password |

🧩 Planned Features
-------------------

*   📦 File attachments (images, PDFs, etc.)
*   🕓 Scheduled emails
*   🧾 Email logs (SQLite/PostgreSQL)
*   🔒 OAuth2 authentication for Gmail
*   🧰 Admin command panel

⚙️ Build a Compiled Version
---------------------------

To create a standalone executable (so others can run it without Go):

### 🧩 Build for your system

    bash
    go build -o bot ./cmd/bot
        

Run it:

    bash
    ./bot
        

or on Windows:
    
    cmd
    bot.exe 

### 🧩 Cross-compile for other platforms

| Platform | Command |
|----------|---------|
| Linux | `GOOS=linux GOARCH=amd64 go build -o bot-linux ./cmd/bot` |
| Windows | `GOOS=windows GOARCH=amd64 go build -o bot.exe ./cmd/bot` |
| macOS | `GOOS=darwin GOARCH=amd64 go build -o bot-mac ./cmd/bot` |

Each build will produce a platform-specific binary you can share.

🎉 **Ready to send emails from Telegram like a pro?** Fire up this bot, blast some messages, and let the good vibes (and emails) flow! 🚀😎