package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"telefeed/internal/bot"
	"telefeed/internal/db"
	"time"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

type Config struct {
	Mode         string        `env:"MODE"`
	BotToken     string        `env:"BOT_TOKEN"`
	DatabaseUrl  string        `env:"DATABASE_URL"`
	PollInterval time.Duration `env:"POLL_INTERVAL" envDefault:"1h"`
	TargetChatId int64         `env:"TARGET_CHAT_ID"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, err := loadConfig()
	if err != nil {
		panic(err)
	}

	logger := initLogger(config.Mode)
	defer func() { _ = logger.Sync() }()

	sqlDB := initDatabase(config.DatabaseUrl, logger)
	defer func() { _ = sqlDB.Close() }()

	parser := gofeed.NewParser()

	queries := db.New(sqlDB)

	pref := tele.Settings{
		Token:  config.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	telebot, err := tele.NewBot(pref)
	if err != nil {
		logger.Fatal("Error creating bot", zap.Error(err))
	}

	h := bot.NewHandler(queries, parser)

	telebot.Use(middleware.AutoRespond())
	telebot.Handle("/start", func(c tele.Context) error {
		return h.OnStart(c)
	})
	telebot.Handle("/list", func(c tele.Context) error {
		return h.OnList(c)
	})
	telebot.Handle("/subscribe", func(c tele.Context) error {
		return h.OnSubscribe(c)
	})
	telebot.Handle(tele.OnCallback, func(c tele.Context) error {
		return h.OnCallback(c)
	})

	options := bot.PollerOptions{
		Interval: config.PollInterval,
		ChatID:   config.TargetChatId,
	}
	poller := bot.NewPoller(options, queries, logger, parser, telebot)
	go poller.Poll(ctx)

	go telebot.Start()
	defer telebot.Stop()

	<-ctx.Done()
	logger.Info("Received shutdown signal")
}

func loadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

func initLogger(mode string) *zap.Logger {
	if mode == "release" {
		return zap.Must(zap.NewProduction())
	}
	return zap.Must(zap.NewDevelopment())
}

func initDatabase(dbUrl string, logger *zap.Logger) *sql.DB {
	u, _ := url.Parse(dbUrl)
	migrate := dbmate.New(u)

	err := migrate.CreateAndMigrate()
	if err != nil {
		logger.Fatal("Error creating and migrating the database", zap.Error(err))
	}

	sqlDB, err := sql.Open("sqlite3", u.Opaque)
	if err != nil {
		logger.Fatal("Error opening the database", zap.Error(err))
	}

	return sqlDB
}
