# Andesite
![loc](https://tokei.rs/b1/github/nektro/andesite)
[![license](https://img.shields.io/github/license/nektro/andesite.svg)](https://github.com/nektro/andesite/blob/master/LICENSE)
[![issues](https://img.shields.io/github/issues/nektro/andesite.svg)](https://github.com/nektro/andesite/issues)
[![paypal](https://img.shields.io/badge/donate-paypal-blue.svg?logo=paypal)](https://www.paypal.me/nektro)
[![patreon](https://img.shields.io/badge/donate-patreon-orange.svg?logo=patreon)](https://www.patreon.com/nektro)

Share folders in an Open Directory without making your entire server public. Manages users with [Discord](https://discordapp.com/).

## Getting Started
These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

### Prerequisites
- A directory you wish to proxy through Andesite
- *For Development* The Go Language (https://golang.org/dl/)
- *For Development* GCC on your PATH (for the https://github.com/mattn/go-sqlite3 installation)

### Installing
Go to (https://discordapp.com/developers/applications/) and create an application on your Discord Developers dashboard. Fill out the name and picture as you see fit. This will be the information shown when Andesite authenticates users through Discord.

Obtain your newly created application's Client ID and Secret from the dashboard.

Create a folder in the root of the directory you will be serving with the name `.andesite`. If your file manager does not allow you to do this at first, you can open a command prompt/terminal and run `mkdir .andesite`.

In the `.andesite` folder make a `config.json` file and put the following data inside.
```
{
    "discord": {
        "id": "CLIENT_ID",
        "secret": "CLIENT_SECRET"
    }
}
```
and replace `CLIENT_ID` and `CLIENT_SECRET` with their respective values. Do not worry, this folder will remain entirely private, even to users with full access.

Run
```
$ go get github.com/nektro/andesite
```
and then make your way to `$GOPATH/src/github.com/nektro/andesite/`.

Once there, run:
```
$ go build
$ ./andesite
```

`go build` is used here instead of `go run main.go` because `go run` creates a new binary every time which, since this program is a server, will request a firewall exception on every run. Using `go build` overwrites the same binary `./andesite` over and over again as changes are made.

### Options
- -root **Required**
    - A relative or absolute path to which you wish for Andesite to serve
- -port
    - The port Andesite will broadcast on. (Default `8000`)
- -admin
    - The Discord Snowflake of a user to add as an admin. Only required once. Admin priviledge allows this user to change the path access of other users.

## Deployment
Works standalone or proxied through Nginx, etc to add additional headers. Pre-built binaries will be released as well.

## Built With
- The Go Programming Lanuage - https://golang.org/
- https://github.com/gorilla/sessions - Session management
- https://github.com/mattn/go-sqlite3 - SQLite handler
- Discord & OAuth2 - https://discordapp.com/ - User Authentication
- https://handlebarsjs.com/ - HTML templating
- https://github.com/aymerick/raymond - Handlebars template rendering
- https://sass-lang.com/ - Cascading CSS styles
- No JavaScript!

## Contributing
We take issues all the time right here on GitHub. We use labels extensively to show the progress through the fixing process. Question issues are okay but make sure to close the issue when it's been answered!

When making a pull request, please have it be associated with an issue and make a comment on the issue saying that you're working on it so everyone else knows what's going on :D

## Contact
- hello@nektro.net
- Meghan#2032 on discordapp.com
- @nektro on twitter.com and mastodon.social

## License
MIT

## Inspiration
Gib Romz
