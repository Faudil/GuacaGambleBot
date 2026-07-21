# GuacaGambleBot — Setup Guide

This guide walks you through hosting your own **GuacaGambleBot** — a Discord
economy, pet-battle, and casino bot — from scratch.

> **Quick start (Docker, recommended):** 4 steps, ~5 minutes.
>
> 1. [Create a Discord application](#step-1-create-a-discord-application)
> 2. Copy the `.env.example` to `.env` and paste your token
> 3. Run `docker compose up -d`
> 4. Invite the bot to your server

---

## Prerequisites

| Requirement | Notes |
|---|---|
| A Discord account | You need to own a Discord server or have *Manage Server* permission on one. |
| A Discord Bot Token | Created on the [Discord Developer Portal](https://discord.com/developers/applications). Free. |
| Docker & Docker Compose | [Install Docker](https://docs.docker.com/get-docker/). If you prefer not to use Docker, you can also [run the Go binary directly](#option-2-running-the-binary-directly). |

---

## Step-by-step

### Step 1: Create a Discord application

1. Go to the **[Discord Developer Portal](https://discord.com/developers/applications)** and click **New Application**.
2. Give it a name (e.g. "GuacaGambleBot") and click **Create**.
3. Go to the **Bot** page in the left sidebar.
4. Click **Reset Token** (or **Copy** if one already exists). **Save this token — you will never be able to see it again.**
5. Under *Privileged Gateway Intents*, enable:
   - **MESSAGE CONTENT INTENT** (required for prefix commands like `!daily`)
   - **SERVER MEMBERS INTENT** (recommended)
6. Go to the **OAuth2 > URL Generator** page.
   - Select **bot** and **applications.commands** scopes.
   - Under *Bot Permissions*, select:
     - `Send Messages`
     - `Embed Links`
     - `Use Slash Commands`
     - `Read Message History`
     - `Attach Files` (optional, for rich pet images)
   - Copy the generated URL and open it in your browser.
   - Select the server you want to add the bot to and click **Authorize**.

### Step 2: Get the bot files

**Option A — Download the source (easiest)**

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/GuacaGambleBot.git
cd GuacaGambleBot
```

> If you don't have Git installed, you can also download the repository as a ZIP
> from GitHub and extract it.

**Option B — Copy the files manually**

Copy the entire project folder onto the machine that will run the bot (a
dedicated server, a Raspberry Pi, your desktop computer, etc.).

### Step 3: Configure the bot

```bash
# Copy the example configuration file
cp .env.example .env
```

Open the `.env` file in any text editor. It looks like this:

```
DISCORD_TOKEN=
GUILD_ID=0
PREFIX=!
STARTING_BALANCE=100
DAILY_AMOUNT=50
BASE_JACKPOT=500
DB_PATH=./data/guacabot_go.db
TZ=
ENVIRONMENT=
```

**You only need to change `DISCORD_TOKEN`** — paste the token you copied in
Step 1. Everything else already has a sensible default.

| Variable | What it does | Default |
|---|---|---|
| `DISCORD_TOKEN` | Your bot's secret token. **Required.** | — |
| `GUILD_ID` | Server ID for instant slash-command registration. Leave `0` for global registration (may take an hour to propagate). | `0` |
| `PREFIX` | Character used before text commands (e.g. `!daily`). Users can change this per server with `/setup`. | `!` |
| `STARTING_BALANCE` | Amount of in-game money new players start with. | `100` |
| `DAILY_AMOUNT` | Amount given by the `!daily` command. | `50` |
| `BASE_JACKPOT` | Starting jackpot for the lotto system. | `500` |
| `DB_PATH` | Where the SQLite database file is stored. | `./data/guacabot_go.db` |
| `TZ` | Server timezone (e.g. `America/New_York`, `Europe/Paris`). Leave empty for system timezone. | *(empty)* |
| `ENVIRONMENT` | Optional label for your tooling. Not used by the bot itself. | *(empty)* |

### Step 4: Start the bot

#### Option 1: Using Docker (recommended)

**This is the easiest method. You don't need to install Go.**

```bash
docker compose up -d
```

To check that it's running:

```bash
docker compose logs -f
```

You should see `GuacaGambleBot (Go) is online. Press CTRL-C to exit.`

To stop the bot:

```bash
docker compose down
```

#### Option 2: Running the binary directly

**You need Go 1.23+ installed.**

```bash
# Build the bot
go build -o bot ./cmd/bot

# Run it
./bot
```

The bot will log to the terminal. Press `CTRL-C` to stop.

---

## What happens after the bot starts

When the bot comes online and joins a server, it automatically posts an
interactive configuration menu. This lets you:

1. **Choose a channel** — where the bot will send announcements and game results.
2. **Choose a language** — currently English or French.
3. **Change the prefix** — if you don't like `!`, you can use a different symbol
   (click **Advanced**).
4. **Enable or disable** the bot on your server.
5. **Finish** — saves everything.

> If the bot cannot find a suitable channel, it will send the menu to the server
> owner's DMs instead. If that also fails, it will try again the next time it
> reconnects.

You can reopen this menu at any time with:

```
/setup
```

Or by typing:

```
!setup
```

---

## Updating the bot

```bash
# Pull the latest source code
git pull

# If using Docker
docker compose up -d --build

# If running the binary directly
go build -o bot ./cmd/bot
./bot
```

---

## Troubleshooting

### The bot won't start — "DISCORD_TOKEN is not set"

Open your `.env` file and make sure the line `DISCORD_TOKEN=your_token_here`
has your actual token after the `=` sign. No spaces.

### The bot is online but doesn't respond to slash commands

- Slash commands can take up to 1 hour to appear on all servers if `GUILD_ID` is
  `0`. Set `GUILD_ID` to your server's ID for instant registration.
- Make sure the bot has the `applications.commands` scope when you invite it
  (see [Step 1](#step-1-create-a-discord-application)).
- Re-invite the bot with the correct permissions if needed.

### The bot doesn't respond to `!` commands

Check that *MESSAGE CONTENT INTENT* is enabled on the Discord Developer Portal
(Bot page). Without it, Discord does not send message content to bots.

### "failed to open database"

Make sure the `data/` directory exists (or change `DB_PATH` in `.env` to a path
that does exist). Docker creates it automatically.

---

## Security notes

- **Never share your `.env` file.** It contains your bot token. Anyone with this
  token can control your bot.
- The `.env` file is listed in `.gitignore` and will not be committed to Git.
- If you suspect your token has been compromised, regenerate it immediately on
  the Discord Developer Portal.

---

## Need help?

Open an issue on the project's GitHub repository.
