package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joshhartwig/gator/internal/config"
	"github.com/joshhartwig/gator/internal/database"

	_ "github.com/lib/pq"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type state struct {
	db     *database.Queries
	config *config.Config
}

// command struct contains name and slice of string args
// ex 'login' - args in slice
type command struct {
	name string
	args []string
}

type commands struct {
	names map[string]func(*state, command) error
}

// run a given command with the provided state
func (c *commands) run(s *state, cmd command) error {
	f, ok := c.names[cmd.name]
	if !ok {
		return fmt.Errorf("gator: error unable to find the requested command %s\n", cmd.name)
	}

	err := f(s, cmd)
	if err != nil {
		return err
	}
	return nil
}

// register a handler for a command name
func (c *commands) register(name string, f func(*state, command) error) {
	_, ok := c.names[name]
	if !ok {
		c.names[name] = f
	}
}

func main() {

	cfg, err := config.Read()
	if err != nil {
		log.Panic("gator: error reading config file")
	}

	db, err := sql.Open("postgres", cfg.DB_URL)
	if err != nil {
		log.Panic("gator: error opening database")
	}

	queries := database.New(db)

	st := state{
		db:     queries,
		config: &cfg,
	}

	// create a new commands struct and register login with handler
	cmds := commands{names: make(map[string]func(*state, command) error)}
	cmds.register("login", handlerLogin)
	cmds.register("register", register)
	cmds.register("reset", handlerReset)
	cmds.register("users", handleListUsers)
	cmds.register("agg", handleAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerGetFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("listfollows", handlerListFollows)
	cmds.register("following", middlewareLoggedIn(handlerGetFollows))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("help", cmds.handlerHelp)

	// get os orgs
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("gator: error not enough arguments passed in, need at least two, got %d\n", len(args))
		printArgs(args)
		os.Exit(1)
	}

	action := args[1]
	actionsArgs := args[2:]

	cmd := command{
		name: action,
		args: actionsArgs,
	}

	err = cmds.run(&st, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

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

// prints each arg with a newline for debugging purposes
func printArgs(args []string) {
	for _, i := range args {
		fmt.Printf("%s\n", i)
	}
	fmt.Printf("args length: %d\n", len(args))
}

// isValidURL checks to see if we have http or https
func isValidURL(url string) bool {
	return len(url) > 0 && (len(url) > 7 && (url[:7] == "http://" || (len(url) > 8 && url[:8] == "https://")))
}
