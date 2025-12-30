package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joshhartwig/gator/internal/config"
	"github.com/joshhartwig/gator/internal/database"

	"github.com/lib/pq"
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
	cmds.register("addfeed", handlerAddFeed)
	cmds.register("feeds", handlerGetFeeds)

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

// handlerAddFeed will regsiter 'addfeed' and get a url from the command
// it will then add that feed to the database and associate it to the current user
func handlerAddFeed(s *state, cmd command) error {

	if len(cmd.args) < 2 || len(cmd.args) > 3 {
		return fmt.Errorf("addfeed should have two or more commands")
	}

	name := cmd.args[0]
	url := cmd.args[1]

	// fetch feed
	_, err := fetchFeed(context.Background(), url)
	if err != nil {
		return err
	}

	// fetch user
	user, err := s.db.GetUser(context.Background(), s.config.Current_User_Name)
	if err != nil {
		return err
	}

	retFeed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})

	fmt.Printf("Feed Name: %s Feed Url: %s\n", retFeed.Name, retFeed.Url)

	return nil

}

func handlerGetFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, f := range feeds {
		fmt.Printf("%s - %s - %s\n", f.Name, f.Url, f.ID)
	}
	return nil
}

func register(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("gator: a username is required\n")
	}

	username := cmd.args[0]

	// search for user first prior to creating new user

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	})
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && string(pgErr.Code) == "23505" {
			return fmt.Errorf("gator: user %q already exists", username)
		}
		return err
	}

	err = s.config.SetUser(user.Name)
	if err != nil {
		return err
	}

	fmt.Printf("gator: the following user was created:\nid: %s\n name: %s\n",
		user.ID.String(), user.Name)

	return nil

}

// will login the passed in user
func handlerLogin(s *state, cmd command) error {
	// check to see if we have any args
	if len(cmd.args) == 0 {
		return errors.New("gator: a username is required\n")
	}
	username := cmd.args[0]

	// find user in db
	userInDb, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("error! unable to find user:%s in database, please try again", username)
	}

	err = s.config.SetUser(userInDb.Name)
	if err != nil {
		return err
	}
	fmt.Printf("gator: the username:%s has been set \n", username)
	return nil
}

// handleReset will delete everything in the users table
func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("gator: reset ran, delete users complete")
	return nil
}

// handleListUsers will list all users in db and the one currently logged in
func handleListUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, u := range users {
		if u.Name == s.config.Current_User_Name {
			fmt.Printf("%s (current)\n", u.Name)
		} else {
			fmt.Printf("%s\n", u.Name)
		}
	}

	return nil
}

func handleAgg(s *state, cmd command) error {
	feed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	fmt.Printf("%v", feed)
	return nil
}

// prints each arg with a newline for debugging purposes
func printArgs(args []string) {
	for _, i := range args {
		fmt.Printf("%s\n", i)
	}
}
