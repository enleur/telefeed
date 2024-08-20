package bot

import (
	"context"
	"testing"

	"telefeed/internal/db"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
	"gopkg.in/telebot.v3"
)

type MockTelebotContext struct {
	SendCalled   int
	ArgsValue    []string
	CallbackData string
}

func (m *MockTelebotContext) Send(interface{}, ...interface{}) error {
	m.SendCalled++
	return nil
}

func (m *MockTelebotContext) Args() []string {
	return m.ArgsValue
}

func (m *MockTelebotContext) Callback() *telebot.Callback {
	return &telebot.Callback{Data: m.CallbackData}
}

func (m *MockTelebotContext) Respond(...*telebot.CallbackResponse) error {
	return nil
}

type MockDbQuery struct{}

func (m *MockDbQuery) ListFeeds(context.Context) ([]db.Feed, error) {
	return []db.Feed{{ID: 1}, {ID: 2}}, nil
}

func (m *MockDbQuery) CreateFeed(context.Context, db.CreateFeedParams) (db.Feed, error) {
	return db.Feed{}, nil
}

func (m *MockDbQuery) DeleteFeed(context.Context, int64) error {
	return nil
}

type MockFeedParser struct{}

func (m *MockFeedParser) ParseURL(feedURL string) (*gofeed.Feed, error) {
	return &gofeed.Feed{
		Title:    "Test Feed",
		FeedLink: feedURL,
		Link:     feedURL,
		Image: &gofeed.Image{
			URL: "http://example.com/image.jpg",
		},
	}, nil
}

func TestOnStart(t *testing.T) {
	mockContext := &MockTelebotContext{}
	handler := NewHandler(nil, nil)

	err := handler.OnStart(mockContext)

	assert.NoError(t, err)
	assert.Equal(t, 1, mockContext.SendCalled)
}

func TestOnList(t *testing.T) {
	mockContext := &MockTelebotContext{}
	mockQueries := &MockDbQuery{}
	handler := NewHandler(mockQueries, nil)

	err := handler.OnList(mockContext)

	assert.NoError(t, err)
	assert.Equal(t, 2, mockContext.SendCalled)
}

func TestOnSubscribe(t *testing.T) {
	mockContext := &MockTelebotContext{ArgsValue: []string{"http://example.com/rss"}}
	mockQueries := &MockDbQuery{}
	mockParser := &MockFeedParser{}
	handler := NewHandler(mockQueries, mockParser)

	err := handler.OnSubscribe(mockContext)

	assert.NoError(t, err)
	assert.Equal(t, 2, mockContext.SendCalled)
}

func TestOnCallback(t *testing.T) {
	mockContext := &MockTelebotContext{CallbackData: "unsubscribe:1"}
	mockQueries := &MockDbQuery{}
	handler := NewHandler(mockQueries, nil)

	err := handler.OnCallback(mockContext)

	assert.NoError(t, err)
	assert.Equal(t, 1, mockContext.SendCalled)
}

func TestOnCallback_InvalidData(t *testing.T) {
	mockContext := &MockTelebotContext{CallbackData: "invalid"}
	handler := NewHandler(nil, nil)

	err := handler.OnCallback(mockContext)

	assert.NoError(t, err)
	assert.Equal(t, 0, mockContext.SendCalled)
}
