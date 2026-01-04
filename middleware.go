package main

import (
	"context"
	"errors"

	"github.com/joshhartwig/gator/internal/database"
)

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, c command) error {
		user, err := s.db.GetUser(context.Background(), s.config.Current_User_Name)
		if err != nil {
			return errors.New("gator: error - unable to find user in database, register a new user or use an existing account")
		}
		err = handler(s, c, user)
		if err != nil {
			return err
		}
		return nil
	}
}
