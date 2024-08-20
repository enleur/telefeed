package bot

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"telefeed/internal/db"

	"github.com/mmcdole/gofeed"

	"gopkg.in/telebot.v3"
)

type TelebotContext interface {
	Send(message interface{}, options ...interface{}) error
	Args() []string
	Callback() *telebot.Callback
	Respond(resp ...*telebot.CallbackResponse) error
}

type DbQuery interface {
	ListFeeds(ctx context.Context) ([]db.Feed, error)
	CreateFeed(ctx context.Context, arg db.CreateFeedParams) (db.Feed, error)
	DeleteFeed(ctx context.Context, id int64) error
}

type FeedParser interface {
	ParseURL(feedURL string) (*gofeed.Feed, error)
}

var welcomeMessage = `Welcome to the RSS Feed Bot!

Here are the available commands:
/subscribe <url> - Subscribe to a new RSS feed
/list - List all your subscribed feeds

To get started, try subscribing to a feed using the /subscribe command followed by the RSS feed URL.`

type Handler struct {
	queries DbQuery
	parser  FeedParser
}

func NewHandler(queries DbQuery, parser FeedParser) *Handler {
	return &Handler{
		queries: queries,
		parser:  parser,
	}
}

func (b *Handler) OnStart(c TelebotContext) error {
	return c.Send(welcomeMessage, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
}

func (b *Handler) OnList(c TelebotContext) error {
	ctx := context.Background()
	feeds, err := b.queries.ListFeeds(ctx)
	if err != nil {
		return c.Send("Error listing feeds" + err.Error())
	}

	for _, feed := range feeds {
		err = c.Send(feed.Title, &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{{Text: "Unsubscribe", Data: fmt.Sprintf("unsubscribe:%d", feed.ID)}},
			},
		})
		if err != nil {
			return c.Send("Error sending the feed" + err.Error())
		}
	}

	return nil
}

func (b *Handler) OnSubscribe(c TelebotContext) error {
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Please provide an rss feed url")
	}
	feedURL := args[0]

	feed, err := b.parser.ParseURL(feedURL)
	if err != nil {
		return c.Send("Error parsing the feed" + err.Error())
	}

	ctx := context.Background()
	_, err = b.queries.CreateFeed(ctx, db.CreateFeedParams{
		Url:   feed.FeedLink,
		Title: sql.NullString{String: feed.Title, Valid: true},
	})
	if err != nil {
		return c.Send("Error subscribing to the feed" + err.Error())
	}

	err = c.Send(&telebot.Photo{
		File:    telebot.FromURL(feed.Image.URL),
		Caption: feed.Title,
	})
	if err != nil {
		return c.Send("Error sending the preview" + err.Error())
	}

	return c.Send("Subscribed!")
}

func (b *Handler) OnCallback(c TelebotContext) error {
	data := strings.Split(c.Callback().Data, ":")
	if len(data) != 2 || data[0] != "unsubscribe" {
		return c.Respond()
	}

	id, err := strconv.ParseInt(data[1], 0, 10)
	if err != nil {
		return c.Send("Invalid feed id")
	}

	err = b.queries.DeleteFeed(context.Background(), id)
	if err != nil {
		return c.Send("Error unsubscribing from the feed" + err.Error())
	}

	return c.Send("Unsubscribed!")
}
