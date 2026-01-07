package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joshhartwig/gator/internal/database"
	"github.com/joshhartwig/gator/internal/ui"
	"github.com/lib/pq"
)

// quit the app
func (c *commands) handlerQuit(s *state, cmd command) error {
	os.Exit(1)
	return nil
}

// handlerHelp prints a list of all available command names to the standard output.
// It iterates over the command names stored in the commands struct and outputs each one.
// Returns an error if any occurs during execution, otherwise returns nil.
func (c *commands) handlerHelp(s *state, cmd command) error {
	s.ui.Header("Help")
	s.ui.Item("%sUsage:%s", ui.Red, ui.Reset)
	s.ui.Item("  gator <command> [args]\n")
	s.ui.Item("%sCommands:%s", ui.Red, ui.Reset)
	for n := range c.names {
		s.ui.Item("  %s", n)
	}

	s.ui.Item("\n%sExamples:%s", ui.Red, ui.Reset)
	s.ui.Item("  gator register ted")
	s.ui.Item("  gator addfeed \"hn\" \"https://hackernews.com/rss")
	s.ui.Item("  gator agg 1m")
	return nil
}

// handlerListFollows retrieves the list of feed follows from the database and outputs
// It returns an error if the database operation fails.
func handlerListFollows(s *state, cmd command) error {
	follows, err := s.db.GetFeedFollows(context.Background())
	if err != nil {
		s.ui.Error(err.Error())
		return err
	}

	s.ui.Header("Listing Feed Follows for each user")
	for _, f := range follows {
		s.ui.Column("%s\t%s\t%s\t%s\n", f.FeedID, f.UserID, f.UserName, f.FeedName)
	}

	return nil
}

// handlerAddFeed handles the addition of a new feed to the user's account.
// It expects the command arguments to contain at least a feed name and URL.
// The function fetches the feed to validate the URL, retrieves the current user from the database,
// and creates a new feed entry associated with the user. If successful, it prints the feed details.
// Returns an error if argument validation fails, feed fetching fails, user retrieval fails, or feed creation fails.
func handlerAddFeed(s *state, cmd command, user database.User) error {
	s.ui.Header("Add Feed")

	if len(cmd.args) < 2 || len(cmd.args) > 3 {
		return fmt.Errorf(`addfeed should have two or more commands (ex addfeed "hackernews" "http://url")`)
	}

	name := cmd.args[0]
	url := cmd.args[1]

	// fetch feed
	_, err := fetchFeed(context.Background(), url)
	if err != nil {
		s.ui.Error(err.Error())
		return err
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
		s.ui.Error(err.Error())
		return err
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
		s.ui.Error(err.Error())
		return err
	}

	s.ui.Item("Feed Name: %s feed url: %s feed following: %s created", retFeed.Name, retFeed.Url, follow.FeedID)
	return nil
}

// handlerGetFeeds retrieves all feeds from the database and prints their details (Name, Url, UserName) to the standard output.
// It returns an error if fetching feeds from the database fails.
func handlerGetFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		s.ui.Error(err.Error())
		return err
	}
	s.ui.Header("Feeds")
	for _, f := range feeds {
		s.ui.Column("%s\t%s\t%s\t\n", f.Name, f.Url, f.UserName)
	}
	return nil
}

// handlerUnfollow will unfollow a feed assigned to a user if that user is currently following the feed
func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 || len(cmd.args) > 2 {
		return fmt.Errorf("unsupported error count %d", len(cmd.args))
	}

	url := cmd.args[0]

	s.ui.Header("Unfollow Feed")
	if !isValidURL(url) {
		return fmt.Errorf("invalid url: %s", cmd.args[0])
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("unable to get feed follows %v", err)
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
		return fmt.Errorf("error deleting feed follow for user %v", err)
	}

	s.ui.Item("Successfully unfollowed feed: %s", url)
	return nil
}

// handlerGetFollows retrieves the list of feed follows for the current user from the database
// and prints each followed feed's username and feed name to the standard output.
// It returns an error if the user or their feed follows cannot be retrieved.
func handlerGetFollows(s *state, cmd command, user database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("unable to get feed follows for user %v", err)
	}
	s.ui.Header("Show Follows")
	for _, f := range follows {
		s.ui.Column("%s\t%s\t\n", f.UserName, f.FeedName)
	}
	return nil
}

// handleReset will delete everything in the users table
func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error reseting database %v", err)
	}
	s.ui.Header("Reset Database")
	s.ui.Info("Reset database complete")
	return nil
}

// handlerFollow takes a single url record and creates a new
// follow record for the current user
func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 || len(cmd.args) > 1 {
		return fmt.Errorf("should only have one argument in follow command but got %d", len(cmd.args))
	}

	url := cmd.args[0]
	if url == "" {
		return fmt.Errorf("The url is blank, pass in a properly formatted url")
	}

	if !isValidURL(url) {
		return fmt.Errorf("The url is not a valid format, pass in a properly formatted url")
	}

	user, err := s.db.GetUser(context.Background(), s.config.Current_User_Name)
	if err != nil {
		return fmt.Errorf("unable to get user from db %v", err)
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return fmt.Errorf("unable to get feed by url %v", err)
	}
	s.ui.Header("Create a Following for User")
	s.ui.Item("Found current user:%s and feed by url:%s", user.Name, feed.Name)

	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	s.ui.Item("New feed following created: id:%s name:%s username:%s", follow.ID, follow.FeedName, follow.UserName)
	return nil
}

// handleListUsers will list all users in db and the one currently logged in
func handlerListUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error listing users %s", err.Error())
	}

	s.ui.Header("List Users")
	for _, u := range users {
		if u.Name == s.config.Current_User_Name {
			s.ui.Item("%s (current logged in user)", u.Name)
		} else {
			s.ui.Item("%s", u.Name)
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
		return fmt.Errorf("unable to get posts for user %s", user.Name)
	}
	s.ui.Header("Browse Posts")

	for _, p := range posts {
		s.ui.Column("%s\t%s\t%s\t\n", p.Title, p.Description.String, p.PublishedAt)
	}
	return nil
}

// handleAgg will aggregate all posts from the feeds and write them to the database
// expects duration argument in time ex 1m
func handlerAgg(s *state, cmd command) error {
	s.ui.Header("Aggregate Feeds")
	if len(cmd.args) < 1 || len(cmd.args) > 1 {
		return fmt.Errorf("The agg command should only have one duration argument, recieved %d arguments.", len(cmd.args))
	}
	timeArg := cmd.args[0]
	duration, err := time.ParseDuration(timeArg)
	if err != nil {
		return err
	}

	s.ui.Item("We will begin downloading posts from your feeds starting in %v", duration)

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
		return fmt.Errorf("a username is required")
	}

	username := cmd.args[0]
	s.ui.Header("Register")

	// search for user first prior to creating new user
	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	})
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && string(pgErr.Code) == "23505" {
			s.ui.Error(fmt.Sprintf("User %q already exists in database", username))
			return fmt.Errorf("error user %q already exists", username)
		}
		return err
	}

	err = s.config.SetUser(user.Name)
	if err != nil {
		return fmt.Errorf("error setting username %s", err.Error())
	}

	s.ui.Item("The following user was created successfully id:%s name:%s", user.ID.String(), user.Name)
	return nil

}

// will login the passed in user
func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("a username is required")
	}
	username := cmd.args[0]

	s.ui.Header("Login")
	// find user in db
	userInDb, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("unable to find user in database %s", username)
	}

	err = s.config.SetUser(userInDb.Name)
	if err != nil {
		return err
	}
	s.ui.Item("Username has been set")
	return nil
}
