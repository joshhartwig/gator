package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/joshhartwig/gator/internal/config"
)

type state struct {
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
		return errors.New("Unable to find the command")
	}

	err := f(s, cmd)
	if err != nil {
		return err
	}
	return nil
}

// register a handler for a command name
func (c *commands) register(name string, f func(*state, command) error) {
	// look up the func in the map by name
	// if we cannot find it, register it
	_, ok := c.names[name]
	if !ok {
		c.names[name] = f
	}
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Panic("error reading config file")
	}

	st := state{
		config: &cfg,
	}

	cmds := commands{names: make(map[string]func(*state, command) error)}
	cmds.register("login", handlerLogin)

	args := os.Args
	if len(args) < 2 {
		fmt.Printf("not enough arguments passed in, need at least two, got %d", len(args))
		os.Exit(1)
	}

}

func handlerLogin(s *state, cmd command) error {
	// check to see if we have any args
	if len(cmd.args) == 0 {
		return errors.New("the login handler expects a single argument, the username")
	}
	username := cmd.args[0]

	err := s.config.SetUser(username)
	if err != nil {
		return err
	}
	fmt.Printf("The username:%s has been set \n", username)
	return nil
}
