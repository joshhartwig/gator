package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/joshhartwig/gator/internal/database"
	"github.com/lib/pq"
)

// handlerHelp prints a list of all available command names to the standard output.
// It iterates over the command names stored in the commands struct and outputs each one.
// Returns an error if any occurs during execution, otherwise returns nil.
func (c *commands) handlerHelp(s *state, cmd command) error {
	for n := range c.names {
		fmt.Printf("- %s\n", n)
	}
	return nil
}

// handlerListFollows retrieves the list of feed follows from the database and outputs them using spew.Dump.
// It returns an error if the database operation fails.
func handlerListFollows(s *state, cmd command) error {
	follows, err := s.db.GetFeedFollows(context.Background())
	if err != nil {
		return err
	}

	spew.Dump(follows)
	return nil
}

// handlerAddFeed handles the addition of a new feed to the user's account.
// It expects the command arguments to contain at least a feed name and URL.
// The function fetches the feed to validate the URL, retrieves the current user from the database,
// and creates a new feed entry associated with the user. If successful, it prints the feed details.
// Returns an error if argument validation fails, feed fetching fails, user retrieval fails, or feed creation fails.
func handlerAddFeed(s *state, cmd command, user database.User) error {

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

	// // fetch user
	// user, err := s.db.GetUser(context.Background(), s.config.Current_User_Name)
	// if err != nil {
	// 	return err
	// }

	retFeed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})

	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    retFeed.UserID,
		FeedID:    retFeed.ID,
	})

	if err != nil {
		return err
	}

	fmt.Printf("Feed Name: %s Feed Url: %s Feed Follow:%s created\n", retFeed.Name, retFeed.Url, follow.FeedID)

	return nil

}

// handlerGetFeeds retrieves all feeds from the database and prints their details (Name, Url, UserName) to the standard output.
// It returns an error if fetching feeds from the database fails.
func handlerGetFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, f := range feeds {
		fmt.Printf("%s - %s - %s\n", f.Name, f.Url, f.UserName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 || len(cmd.args) > 2 {
		return fmt.Errorf("unsupported arg count: %d", len(cmd.args))
	}

	url := cmd.args[0]
	if !isValidURL(url) {
		return fmt.Errorf("invalid url: %s", cmd.args[0])
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
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
	err = s.db.DeleteFeedFollowForUser(context.Background(), database.DeleteFeedFollowForUserParams{
		UserID: user.ID,
		FeedID: feedIdToDelete,
	})

	if err != nil {
		return err
	}
	return nil
}

// handlerGetFollows retrieves the list of feed follows for the current user from the database
// and prints each followed feed's username and feed name to the standard output.
// It returns an error if the user or their feed follows cannot be retrieved.
func handlerGetFollows(s *state, cmd command, user database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}

	for _, f := range follows {
		fmt.Printf("%s - %s\n", f.UserName, f.FeedName)
	}
	return nil
}

// handleReset will delete everything in the users table
func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("reset complete\n")
	return nil
}

// handlerFollow takes a single url record and creates a new
// follow record for the current user
func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 || len(cmd.args) > 1 {
		return fmt.Errorf("should only have one argument in follow command but got %d\n", len(cmd.args))
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
		return err
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return err
	}

	fmt.Printf("found current user:%s and feed by url:%s", user.Name, feed.Name)

	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})

	fmt.Printf("New Following Created:\nid:%s\nname:%s\nusername:%s\n", follow.ID, follow.FeedName, follow.UserName)
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

// register attempts to create a new user with the provided username from the command arguments.
// If no username is provided, it returns an error. If the user already exists, it returns a specific error message.
// On successful creation, it sets the user in the configuration and prints the user details.
// Returns an error if any step fails.
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
