package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joshhartwig/gator/internal/database"
)

type message int

const prefix = "Gator - RSS Aggregator Command Line Interface"

func Successf(format string, args ...any) {
	fmt.Fprintf(os.Stdout, "%s info: %s\n", prefix, fmt.Sprintf(format, args...))
}

func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stdout, "%s\n", fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s warn: %s\n", prefix, fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) error {
	fmt.Fprintf(os.Stdout, "%s error: %s\n", prefix, fmt.Sprintf(format, args...))
	return fmt.Errorf(format, args...)
}

func scrapeFeeds(s *state) error {
	// fetch the latest feed
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return errors.ErrUnsupported
	}

	// mark the feed as fetched
	_, err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return Errorf("error marking fetch feed with id:%s error: %s", feed.ID.String(), err)
	}

	rss, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return Errorf("unable to fetch feed with the following url:%s error:%s", feed.Url, err)
	}

	Infof("%s\n", rss.Channel.Title)
	for _, r := range rss.Channel.Item {

		// attempt to parse the time, if not set it to now
		pubDate, err := time.Parse("time.RFC3339", r.PubDate)
		if err != nil {
			pubDate = time.Now()
		}

		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       r.Title,
			Url:         r.Link,
			Description: sql.NullString{String: r.Description},
			PublishedAt: pubDate,
			FeedID:      feed.ID,
		})

		if err != nil {
			return Errorf("error creating post %v", err)
		}
		Successf("added entry to db: %s @ %v\n", r.Title, r.PubDate)
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
