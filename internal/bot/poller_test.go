package bot

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"telefeed/internal/db"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type MockStorage struct {
	feeds         []db.Feed
	err           error
	updateErr     error
	listFeedCalls int
}

func (m *MockStorage) ListFeeds(context.Context) ([]db.Feed, error) {
	m.listFeedCalls++
	return m.feeds, m.err
}

func (m *MockStorage) UpdateFeedLastFetched(context.Context, db.UpdateFeedLastFetchedParams) error {
	return m.updateErr
}

type MockParser struct {
	feed *gofeed.Feed
	err  error
}

func (m *MockParser) ParseURL(string) (*gofeed.Feed, error) {
	return m.feed, m.err
}

type MockBot struct {
	err error
}

func (m *MockBot) Send(tele.Recipient, interface{}, ...interface{}) (*tele.Message, error) {
	return &tele.Message{}, m.err
}

func TestNewPoller(t *testing.T) {
	options := PollerOptions{Interval: time.Minute, ChatID: 123}
	storage := &MockStorage{}
	logger := zap.NewNop()
	parser := &MockParser{}
	bot := &MockBot{}

	poller := NewPoller(options, storage, logger, parser, bot)

	assert.NotNil(t, poller)
	assert.Equal(t, options, poller.options)
	assert.Equal(t, storage, poller.storage)
	assert.Equal(t, logger, poller.logger)
	assert.Equal(t, parser, poller.parser)
	assert.Equal(t, bot, poller.bot)
}

func TestPollOnce(t *testing.T) {
	options := PollerOptions{Interval: time.Minute, ChatID: 123}
	storage := &MockStorage{
		feeds: []db.Feed{
			{ID: 1, Url: "http://example.com/feed", LastFetchedAt: sql.NullTime{Time: time.Now().Add(-2 * time.Hour), Valid: true}},
		},
	}
	logger := zap.NewNop()
	parser := &MockParser{
		feed: &gofeed.Feed{
			Items: []*gofeed.Item{
				{
					Title:           "New Post",
					Link:            "http://example.com/new-post",
					PublishedParsed: &time.Time{},
				},
			},
		},
	}
	bot := &MockBot{}

	poller := NewPoller(options, storage, logger, parser, bot)

	err := poller.PollOnce(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 1, storage.listFeedCalls)
}

func TestPollOnce_ErrorListingFeeds(t *testing.T) {
	options := PollerOptions{Interval: time.Minute, ChatID: 123}
	storage := &MockStorage{
		err: errors.New("database error"),
	}
	logger := zap.NewNop()
	parser := &MockParser{}
	bot := &MockBot{}

	poller := NewPoller(options, storage, logger, parser, bot)

	err := poller.PollOnce(context.Background())

	assert.Error(t, err)
	assert.Equal(t, storage.err, err)
	assert.Equal(t, 1, storage.listFeedCalls)
}

func TestPoll(t *testing.T) {
	options := PollerOptions{Interval: 10 * time.Millisecond, ChatID: 123}
	storage := &MockStorage{
		feeds: []db.Feed{
			{ID: 1, Url: "http://example.com/feed", LastFetchedAt: sql.NullTime{Time: time.Now().Add(-2 * time.Hour), Valid: true}},
		},
	}
	logger := zap.NewNop()
	parser := &MockParser{
		feed: &gofeed.Feed{
			Items: []*gofeed.Item{
				{
					Title:           "New Post",
					Link:            "http://example.com/new-post",
					PublishedParsed: &time.Time{},
				},
			},
		},
	}
	bot := &MockBot{}

	poller := NewPoller(options, storage, logger, parser, bot)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	poller.Poll(ctx)

	assert.Greater(t, storage.listFeedCalls, 1)
}

func TestPoll_Cancellation(t *testing.T) {
	options := PollerOptions{Interval: 10 * time.Millisecond, ChatID: 123}
	storage := &MockStorage{}
	logger := zap.NewNop()
	parser := &MockParser{}
	bot := &MockBot{}

	poller := NewPoller(options, storage, logger, parser, bot)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		poller.Poll(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Poll did not stop after context cancellation")
	}
}
