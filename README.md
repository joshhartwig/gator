![Gator Logo](gator.png)

# Gator

Gator command line RSS reader / Blog aggregator that is part of the Boot.dev course. The application allows you to follow as many RSS feeds as you want and simply allow the app to aggregate the feeds non stop. If you let it run, it will keep downloading and updating to the latests feeds. Additionally it allows for multiple users, and allows for you to follow other users feeds.

## About the course

I have been learning Go for over a year and I thought this was a great course and teaches some new design patterns commonly used in CLIs. Additionally, creating a service that runs non stop and constantly pulls data is a great foundation for other apps. If you are interested in learning Go and taking on challening new projects, I recommend <https://boot.dev>

## Features

- multiuser cli for monitoring rss feeds
- reads blogs and websites
- minimal library usage
  - spew (debugging)
  - sqlc for go SQL query generation
  - goose for sql migrations
  - uuid for uuids
  - libpq for Postgres

### Development

```bash
git clone https://github.com/joshhartwig/gator.git
cd gator
docker compose up -d
go run ./...
```

### Installation

Setup the database by downloading the Docker Compose file and creating a new Postgres database. Run the executable.

### Usage

```bash
# admin related commands
gator help # shows all commands
gator listfollows # lists shows all the various feed follows
gator feeds # shows all feeds in the database
gator users # shows all users in the database
gator reset # resets the database to a new state

# user commands
gator register 'ted' # creates a new user in the database and sets them as current
gator login 'ted' # logs in as if the user exists in database
gator addfeed 'hackernews' 'https://hackernews.com/feed' # adds a new feed with name and url
gator following # shows the feeds the current user is following
gator unfollow # pass in a url and remove a feed if found
gator browse # shows the most recent posts for the logged in user
```

## Contributing

Open issues or submit pull requests.

## License

MIT
