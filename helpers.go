package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/joshhartwig/gator/internal/database"
)

const (
	prefix = "\n🐊 Gator - RSS Aggregator Command Line Interface"
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m]"
	Blue   = "\033[34m"
)

func scrapeFeeds(s *state) error {
	// fetch the latest feed
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return errors.ErrUnsupported
	}

	rss, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("unable to fetch feed with the following url:%s error:%s", feed.Url, err)
	}

	// mark the feed as fetched
	_, err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return fmt.Errorf("error marking fetch feed with id:%s error: %s", feed.ID.String(), err)
	}

	if len(rss.Channel.Item) == 0 {
		s.ui.Item("No new posts for: %s", rss.Channel.Title)
		return nil
	}

	// loop through each item in the channel
	s.ui.Item("%s", rss.Channel.Title)
	for _, r := range rss.Channel.Item {

		// attempt to parse the time, if not set it to now
		pubDate, err := time.Parse(time.RFC3339, r.PubDate)
		if err != nil {
			pubDate = time.Now()
		}

		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       r.Title,
			Url:         r.Link,
			Description: sql.NullString{String: r.Description, Valid: true},
			PublishedAt: pubDate,
			FeedID:      feed.ID,
		})

		if err != nil {
			return fmt.Errorf("error creating post %v", err)
		}

		t, err := time.Parse(time.DateTime, r.PubDate)
		if err != nil {
			s.ui.Column("  + %s\t%s\t\n", r.Title, time.DateTime)
		} else {
			s.ui.Column("  + %s\t%s\t\n", r.Title, t)
		}
	}
	return nil
}

// fetchFeed retrieves and parses an RSS feed from the specified URL.
// It sends an HTTP GET request with a custom User-Agent header, reads the response body,
// and unmarshals the XML data into an RSSFeed struct.
// Returns a pointer to the RSSFeed and any error encountered during the process.
func fetchFeed(ctx context.Context, feedUrl string) (*RSSFeed, error) {
	feed := RSSFeed{}
	req, err := http.NewRequestWithContext(ctx, "GET", feedUrl, nil)
	if err != nil {
		return &feed, err
	}

	req.Header.Set("User-Agent", "gator")
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return &feed, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return &feed, err
	}

	err = xml.Unmarshal(data, &feed)
	if err != nil {
		return &feed, err
	}
	return &feed, nil
}

// isValidURL checks to see if we have http or https
func isValidURL(url string) bool {
	return len(url) > 0 && (len(url) > 7 && (url[:7] == "http://" || (len(url) > 8 && url[:8] == "https://")))
}
