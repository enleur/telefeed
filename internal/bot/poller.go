package bot

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"telefeed/internal/db"
	"time"

	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type Storage interface {
	ListFeeds(ctx context.Context) ([]db.Feed, error)
	UpdateFeedLastFetched(ctx context.Context, params db.UpdateFeedLastFetchedParams) error
}

type Parser interface {
	ParseURL(url string) (*gofeed.Feed, error)
}

type Bot interface {
	Send(to tele.Recipient, what interface{}, opts ...interface{}) (*tele.Message, error)
}

type PollerOptions struct {
	Interval time.Duration
	ChatID   int64
}

type Poller struct {
	options PollerOptions
	storage Storage
	logger  *zap.Logger
	parser  Parser
	bot     Bot
}

func NewPoller(options PollerOptions, storage Storage, logger *zap.Logger, parser Parser, bot Bot) *Poller {
	return &Poller{
		options: options,
		storage: storage,
		logger:  logger,
		parser:  parser,
		bot:     bot,
	}
}

func (p *Poller) Poll(ctx context.Context) {
	ticker := time.NewTicker(p.options.Interval)
	defer ticker.Stop()

	err := p.PollOnce(ctx)
	if err != nil {
		p.logger.Error("Error polling feeds", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Poller stopped")
			return
		case <-ticker.C:
			err := p.PollOnce(ctx)
			if err != nil {
				p.logger.Error("Error polling feeds", zap.Error(err))
				continue
			}
		}
	}
}

func (p *Poller) PollOnce(ctx context.Context) error {
	p.logger.Info("Polling feeds")

	feeds, err := p.storage.ListFeeds(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	feedChan := make(chan db.Feed, len(feeds))
	for _, f := range feeds {
		feedChan <- f
	}
	close(feedChan)

	workerCount := runtime.NumCPU()
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range feedChan {
				p.processFeed(ctx, f)
			}
		}()
	}

	wg.Wait()
	return nil
}

func (p *Poller) processFeed(ctx context.Context, f db.Feed) {
	feed, err := p.parser.ParseURL(f.Url)
	if err != nil {
		p.logger.Error("Error parsing feed URL", zap.Error(err))
		return
	}

	sort.Slice(feed.Items, func(i, j int) bool {
		return feed.Items[i].PublishedParsed.Before(*feed.Items[j].PublishedParsed)
	})

	for _, item := range feed.Items {
		if item.PublishedParsed.Before(f.LastFetchedAt.Time) {
			continue
		}

		_, err = p.bot.Send(tele.ChatID(p.options.ChatID), fmt.Sprintf("%s\n%s", item.Title, item.Link))
		if err != nil {
			p.logger.Error("Error sending message", zap.Error(err))
			continue
		}

		err = p.storage.UpdateFeedLastFetched(ctx, db.UpdateFeedLastFetchedParams{
			LastFetchedAt: sql.NullTime{Time: *item.PublishedParsed, Valid: true},
			ID:            f.ID,
		})
		if err != nil {
			p.logger.Error("Error updating feed last fetched", zap.Error(err))
			continue
		}
	}
}
