# Office Games

Office games is an online tracker for games in the office (or anywhere really, this is just what I'm going to use it for). Users can create an "Office" which has it's own ELO ranking system for it's players. Whenever a game is played, users can log the result on the site and see the ranks update.

## Test Accounts

| Username | Password |
| --- | --- |
| johndoe | password |
| janedoe | password |

## Development

### Environment Variables

At the time of writing, there are three required environment variables...

```
DB_URL=<POSTGRES_DB_URL>
JWT_SECRET=<JWT_SECRET>
EMAIL_PASSWORD=<EMAIL_PASSWORD>
```

### Running the App Locally

#### Standard

```sh
# Generating Go code from Templ templates
templ generate

# Build and run
go build
./office-games

# or just run
go run main.go
```

#### Air

In the project root, there the configuration file for [Air](https://github.com/air-verse/air) (live reloading for Go apps). Assuming you have it installed, you can just run `Air` in the root and it will generate and build the necessary files.

#### Docker

**Build**

```sh
docker build . -t RowMur/office-games
```

**Run**

At the time of writing, the docker setup does not setup a database in the container. This means if the `.env` file you're using for local dev includes a reference to `localhost` (e.g. for database URL), these endpoints won't be resolved. Changing any reference to `localhost` with `host.docker.internal` will do the trick but that then breaks local dev outside of docker. Perhaps create a second `.env` file for now with the references changed.

```sh
docker run -p8080:8080  --env-file ./.env -t RowMur/office-games
```
