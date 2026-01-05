package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joshhartwig/gator/internal/database"
	"github.com/lib/pq"
)

// handlerHelp prints a list of all available command names to the standard output.
// It iterates over the command names stored in the commands struct and outputs each one.
// Returns an error if any occurs during execution, otherwise returns nil.
func (c *commands) handlerHelp(s *state, cmd command) error {
	fmt.Println(prefix)
	fmt.Printf("\033[31mUsage:\033[0m\n\tgator <command> [args]\n\n")
	Infof("\033[31mCommands:\033[0m\n")
	for n := range c.names {
		Infof(" %s", n)
	}

	fmt.Printf("\033[31mExamples:\033[0m\n")
	fmt.Printf("gator register ted\ngator addfeed \"hn\" \"https://hackernews.com/rss\"\ngator agg 1m")
	return nil
}

// handlerListFollows retrieves the list of feed follows from the database and outputs them using spew.Dump.
// It returns an error if the database operation fails.
func handlerListFollows(s *state, cmd command) error {
	follows, err := s.db.GetFeedFollows(context.Background())
	if err != nil {
		return Errorf("unable to get feed follows %v", err)
	}
	fmt.Println("Listing your follows")
	for _, f := range follows {
		Infof("- %s", f.FeedID)
	}

	return nil
}

// handlerAddFeed handles the addition of a new feed to the user's account.
// It expects the command arguments to contain at least a feed name and URL.
// The function fetches the feed to validate the URL, retrieves the current user from the database,
// and creates a new feed entry associated with the user. If successful, it prints the feed details.
// Returns an error if argument validation fails, feed fetching fails, user retrieval fails, or feed creation fails.
func handlerAddFeed(s *state, cmd command, user database.User) error {

	if len(cmd.args) < 2 || len(cmd.args) > 3 {
		return fmt.Errorf(`addfeed should have two or more commands (ex addfeed "hackernews" "http://url")`)
	}

	name := cmd.args[0]
	url := cmd.args[1]

	// fetch feed
	_, err := fetchFeed(context.Background(), url)
	if err != nil {
		return Errorf("error fetching feed %v", err)
	}

	// create a new feed in the db and return it
	retFeed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})

	if err != nil {
		return Errorf("error creating a new feed %v", err)
	}

	// create a new follow
	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    retFeed.UserID,
		FeedID:    retFeed.ID,
	})

	if err != nil {
		return Errorf("error creating a new feed following: %v", err)
	}

	Infof("Feed Name: %s feed url: %s feed following:%s created", retFeed.Name, retFeed.Url, follow.FeedID)
	return nil
}

// handlerGetFeeds retrieves all feeds from the database and prints their details (Name, Url, UserName) to the standard output.
// It returns an error if fetching feeds from the database fails.
func handlerGetFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return Errorf("error running getting feeds %v", err)
	}

	for _, f := range feeds {
		Infof("%s - %s - %s", f.Name, f.Url, f.UserName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 || len(cmd.args) > 2 {
		Errorf("unsupported arg count: %d", len(cmd.args))
	}

	url := cmd.args[0]
	if !isValidURL(url) {
		return Errorf("invalid url: %s", cmd.args[0])
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return Errorf("unable to get feed follows %v", err)
	}

	// loop through the feeds and find the feed id that matches the url
	// when we find it assign to feedIdToDelete
	feedIdToDelete := uuid.UUID{}
	for _, f := range follows {
		if f.FeedUrl == url {
			feedIdToDelete = f.FeedID
		}
	}

	// delete the feed now that we have the feedid & user
	if err = s.db.DeleteFeedFollowForUser(context.Background(), database.DeleteFeedFollowForUserParams{
		UserID: user.ID,
		FeedID: feedIdToDelete,
	}); err != nil {
		return Errorf("error deleting feed follow for user %v", err)
	}

	Infof("successfully unfollowed feed: %s", url)
	return nil
}

// handlerGetFollows retrieves the list of feed follows for the current user from the database
// and prints each followed feed's username and feed name to the standard output.
// It returns an error if the user or their feed follows cannot be retrieved.
func handlerGetFollows(s *state, cmd command, user database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return Errorf("unable to get feed follows for user %v", err)
	}

	for _, f := range follows {
		Infof("%s - %s", f.UserName, f.FeedName)
	}
	return nil
}

// handleReset will delete everything in the users table
func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return Errorf("error reseting database %v", err)
	}
	Successf("success reset database")
	return nil
}

// handlerFollow takes a single url record and creates a new
// follow record for the current user
func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 || len(cmd.args) > 1 {
		return Errorf("should only have one argument in follow command but got %d", len(cmd.args))
	}

	url := cmd.args[0]
	if url == "" {
		return Errorf("The url is blank, pass in a properly formatted url")
	}

	if !isValidURL(url) {
		return Errorf("The url is not a valid format, pass in a properly formatted url")
	}

	user, err := s.db.GetUser(context.Background(), s.config.Current_User_Name)
	if err != nil {
		return Errorf("unable to get user from db %v", err)
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return Errorf("unable to get feed by url %v", err)
	}

	Infof("found current user:%s and feed by url:%s", user.Name, feed.Name)

	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	Successf("new feed following created: id:%s name:%s username:%s", follow.ID, follow.FeedName, follow.UserName)
	return nil
}

// handleListUsers will list all users in db and the one currently logged in
func handlerListUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return Errorf("error listing users %s", err.Error())
	}

	Infof("users:")
	for _, u := range users {
		if u.Name == s.config.Current_User_Name {
			Infof("%s (current logged in user)", u.Name)
		} else {
			Infof("%s", u.Name)
		}
	}

	return nil
}

// handlerBrowsePosts shows all RSS Items that have been gathered in the database
func handlerBrowsePosts(s *state, cmd command, user database.User) error {
	limit := 3
	if len(cmd.args) > 0 {
		if l, err := strconv.Atoi(cmd.args[0]); err == nil {
			limit = l
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return Errorf("unable to get posts for user %s", user.Name)
	}

	Infof("browsing the %d most recent posts", limit)

	for _, p := range posts {
		Infof("- %s %s %s", p.Title, p.Description.String, p.PublishedAt)
	}
	return nil
}

// handleAgg will aggregate all posts from the feeds and write them to the database
// expects duration argument in time ex 1m
func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) < 1 || len(cmd.args) > 1 {
		return Errorf("The agg command should only have one duration argument, recieved %d arguments.", len(cmd.args))
	}
	timeArg := cmd.args[0]
	duration, err := time.ParseDuration(timeArg)
	if err != nil {
		return err
	}

	Infof("We will begin downloading posts from your feeds starting in %v", duration)
	var wg sync.WaitGroup

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for range ticker.C {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scrapeFeeds(s)
		}()
	}
	wg.Wait()
	return nil
}

// register attempts to create a new user with the provided username from the command arguments.
// If no username is provided, it returns an error. If the user already exists, it returns a specific error message.
// On successful creation, it sets the user in the configuration and prints the user details.
// Returns an error if any step fails.
func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return Errorf("a username is required")
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
			s.renderer.Error(fmt.Sprintf("User %q already exists in database", username))
			return Errorf("error user %q already exists", username)
		}
		return err
	}

	err = s.config.SetUser(user.Name)
	if err != nil {
		return Errorf("error setting username %s", err.Error())
	}

	Infof("the following user was created successfully id:%s name:%s", user.ID.String(), user.Name)
	return nil

}

// will login the passed in user
func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return Errorf("a username is required")
	}
	username := cmd.args[0]

	// find user in db
	userInDb, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return Errorf("unable to find user in database %s", username)
	}

	err = s.config.SetUser(userInDb.Name)
	if err != nil {
		return err
	}
	print(message(1), fmt.Sprintf("successfully unfollowed feed: %s", username))
	fmt.Printf("gator: the username:%s has been set", username)
	return nil
}
