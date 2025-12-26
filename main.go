package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joshhartwig/gator/internal/config"
	"github.com/joshhartwig/gator/internal/database"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

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
		fmt.Printf("gator: succesfully registered the %s command\n", name)
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
	cmds.register("admin", handlerAdmin)

	// get os orgs
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("gator: error not enough arguments passed in, need at least two, got %d\n", len(args))
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

func handlerAdmin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("gator: a username is required\n")
	}
	adminCommand := cmd.args[0]

	if adminCommand == "delete" {
		err := deleteUsers(s)
		if err != nil {
			return err
		}
		fmt.Printf("gator: admin delete users complete")
	}
	return nil
}

func deleteUsers(s *state) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return err
	}
	return nil
}
