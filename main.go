package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joshhartwig/gator/internal/config"
	"github.com/joshhartwig/gator/internal/database"
	ui "github.com/joshhartwig/gator/internal/ui"

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
	ui     *ui.Renderer
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

	fmt.Println(prefix) // print out gator start message
	cfg, err := config.Read()
	if err != nil {
		log.Panic("error reading config file")
	}

	db, err := sql.Open("postgres", cfg.DB_URL)
	if err != nil {
		log.Panic("error opening database")
	}

	queries := database.New(db)

	st := state{
		db:     queries,
		config: &cfg,
		ui:     ui.New(os.Stdout),
	}

	// create a new commands struct and register login with handler
	cmds := commands{names: make(map[string]func(*state, command) error)}

	cmds.register("help", cmds.handlerHelp)
	cmds.register("listfollows", handlerListFollows)
	cmds.register("feeds", handlerGetFeeds)
	cmds.register("users", handlerListUsers)
	cmds.register("reset", handlerReset)

	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("agg", handlerAgg)
	cmds.register("browse", middlewareLoggedIn(handlerBrowsePosts))
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerGetFollows))
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))

	// get os orgs
	args := os.Args
	if len(args) < 2 {
		st.ui.Error("Gator requires two or more arguments initially, see 'help' for more assistance")
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
