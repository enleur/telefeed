# Telefeed Bot

Telefeed Bot is a Telegram bot designed to follow RSS feeds and repost updates into a specified Telegram channel. It allows users to manage their RSS feed subscriptions directly through chat commands.

**Disclaimer:** This project is designed for use on my homelab server.

## Installation

1. Clone the repository:
    ```sh
    gh repo clone github.com/enleur/telefeed
    ```

2. Install dependencies:
    ```sh
    go mod download
    ```

3. Create a `.env` file or set environment variables

4. Run the bot:
    ```sh
    go run cmd/bot/main.go
    ```

## Usage

### Commands

- `/start` - Start the bot and get a welcome message
- `/subscribe <url>` - Subscribe to a new RSS feed
- `/list` - List all your subscribed feeds
- Click "Unsubscribe" button to remove a feed

## Configuration

The bot can be configured using environment variables defined in the `.env` file. The key variables are:

- `MODE`: Set to `test` or `release`.
- `BOT_TOKEN`: Your Telegram bot token.
- `DATABASE_URL`: URL to your SQLite database.
- `POLL_INTERVAL`: Interval for polling RSS feeds (e.g., `1h`).
- `TARGET_CHAT_ID`: The Telegram chat ID where updates will be posted.

## Database

The bot uses SQLite for storing feed subscriptions. The database schema is managed using [dbmate](https://github.com/amacneil/dbmate). Code for database operations is generated using [sqlc](https://github.com/kyleconroy/sqlc).

## Logging

The bot uses [zap](https://github.com/uber-go/zap) for logging. Logs are output to the console. The logging level and style is determined by the `MODE` environment variable.