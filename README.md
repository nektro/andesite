# Andesite
![loc](https://tokei.rs/b1/github/nektro/andesite)
[![license](https://img.shields.io/github/license/nektro/andesite.svg)](https://github.com/nektro/andesite/blob/master/LICENSE)
[![discord](https://img.shields.io/discord/551971034593755159.svg)](https://discord.gg/P6Y4zQC)
[![CircleCI](https://circleci.com/gh/nektro/andesite.svg?style=svg)](https://circleci.com/gh/nektro/andesite)

Share folders in an Open Directory without making your entire server public. Manages users with OAuth2.

[![buymeacoffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/nektro)

## Getting Started
These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

### Prerequisites
- A directory you wish to proxy through Andesite
- The Go Language 1.7+ (https://golang.org/dl/)
- GCC on your PATH (for the https://github.com/mattn/go-sqlite3 installation)

### Installing
Go to the developers dashboard of your choosing for the authentication platform you wish to use from the list below.

| Identity Provider | Short Code | Developer Dashboard |
| --- | --- | --- |
| Discord | `discord` | https://discordapp.com/developers/applications/ |
| Reddit | `reddit` | https://www.reddit.com/prefs/apps |
| GitHub | `github` | https://github.com/settings/developers |
| Google | `google` | https://console.developers.google.com |
| Facebook | `facebook` | https://developers.facebook.com/apps/ |
| Microsoft | `microsoft` | https://apps.dev.microsoft.com/ |


Once there, create an application and obtain the Client ID and Client Secret. Here you can also fill out a picture and description that will be displayed during the authorization of users on your chosen Identity Provider. When prompted for the "Redirect URI" during the app setup process, the URL to use will be `http://andesite/callback`, replacing `andesite` with any origins you wish Andesite to be usable from, such as `example.com` or `localhost:800`.

Once you have finished the app creation process and obtained the Client ID and Client Secret, create a folder in your home directory at the path of `~/.config/andesite/`. All of Andesite's config and local save files will go here. This directory will be referred to as `.andesite` going forward.

In the `.andesite` folder make a `config.json` file and put the following data inside, replacing `AUTH` with whichever Identity Provider you chose, such as `discord`, `reddit`, etc. And `CLIENT_ID` and `CLIENT_SECRET` with their respective values. Do not worry, this folder will remain entirely private, even to users with full access.

```json
{
    "auth": "{AUTH}",
    "{AUTH}": {
        "id": "{CLIENT_ID}",
        "secret": "{CLIENT_SECRET}"
    }
}
```

> Note: You may currently only use one Identity Provider at a time!

Run
```
$ go get -u github.com/nektro/andesite
```
and then make your way to `$GOPATH/src/github.com/nektro/andesite/`.

Once there, run:
```
$ go build
$ ./andesite
```

### Options
There are a number of options that are also required and can be used to configure your Andesite instance from within your `config.json`. They are listed here.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `"root"` | `string` | **Required.** | A relative or absolute path to where the data root Andesite should serve from is. |
| `"port"` | `uint` | `8000` | The port to bind to. A webserver will be launched accessible from `localhost:{port}`. |
| `"themes"` | `[]string` | ` ` | A array of names to load themes from. Read more about themes below. |
| `"base"` | `string` | `/` | The root path Andesite will be served from. See [`deployment.md`](docs/deployment.md) for more info. |
| `"providers"` | `[]Provider` | ` ` | An array of custom OAuth2 providers that you may use as your `"auth"`. See [`provider.go`](https://github.com/nektro/go.oauth2/blob/master/provider.go) for more info. |
| `"custom"` | `[]OA2Config` | ` ` | An array of OA2 app configs, that can be used with providers created in `"providers"`. See [`providers.md`](docs/providers.md) for more info. |
| `"public"` | `string` | None. | Similar to `--root`, but served from `/public/` and no authorization is required to see files listings or download. Like a regular OD. |

### Discord Guild/Role Access Grant
Due to a limitation in the Discord API, in order to determine if a user has a role on a specific server, you must use a bot. To get started, go to https://discordapp.com/developers/applications/ and add a Bot user to your app and copy down the Bot Token. Now, to be able to give file/folder access to entire roles, add the to your config like this:

```json
"discord": {
    "id": "{CLIENT_ID}",
    "secret": "{CLIENT_SECRET}",
    "extra_1": "{GUILD_SNOWFLAKE}",
    "extra_2": "{BOT_TOKEN}"
}
```

The value for `{GUILD_SNOWFLAKE}` can be obtained from the URL of the server. So if the server you want to add access for is `https://discordapp.com/channels/184315303323238400/184315303323238400`, then the first ID, `184315303323238400`, is the snowflake you need. The other is for the channel you are currently in. This value is not needed.

Enabling these values will add a section to `http://andesite/admin` that you can input the role snowflakes and the path you are granting.

## Themes
Andesite supports making custom themes for the splash page and the various HTML templates throughout the program. Those are:
- `index.html` - [Default Source](./www/index.html)
    - The main page shown to all users at the root of the server.
- `response.hbs` - [Default Source](./www/response.hbs)
    - A generic page used to show errors and message to the user.
- `listing.hbs` - [Default Source](./www/listing.hbs)
    - The main directory listing page.
- `admin.hbs` - [Default Source](./www/admin.hbs)
    - The admin dashboard that allows editing the access of users

### Using A Theme
All or none of the files may be replaced when using a theme. To enable use of a theme, suppose the value passed to `--theme` was `example`. Doing this will tell Andesite to serve files from `.andesite/themes/example/`.

## Deployment
[![CircleCI](https://circleci.com/gh/nektro/andesite.svg?style=svg)](https://circleci.com/gh/nektro/andesite)

Pre-compiled binaries are published on Circle CI at https://circleci.com/gh/nektro/andesite. To download a binary, navigate to the most recent build and click on 'Artifacts'. Here there will be a list of files. Click on the one appropriate for your system.

Once downloaded, run the following with the values applicable to you.
```
$ ./andesite-{date}-{tag}-{os}-{arch}
```

If you decide to pass Andesite through a reverse proxy, be sure to check out the [documentation](./docs/deployment/) for more info.

## Built With
- The Go Programming Lanuage - https://golang.org/
- https://github.com/gorilla/sessions - Session management
- https://github.com/mattn/go-sqlite3 - SQLite handler
- Discord & OAuth2 - https://discordapp.com/ - User Authentication
- https://handlebarsjs.com/ - HTML templating
- https://github.com/aymerick/raymond - Handlebars template rendering
- https://github.com/gorilla/securecookie
- https://github.com/kataras/go-sessions - HTTP Session manager for fasthttp
- https://github.com/mitchellh/go-homedir
- https://github.com/nektro/go-util - Go utilities for simplifying common complex tasks
- https://github.com/spf13/pflag - Optimized flag handler that makes program flags POSIX compliant
- https://github.com/nektro/go.oauth2 - OAuth2 Client library for Go
- https://github.com/nektro/go.etc
- https://github.com/rakyll/statik/fs

## Contributing
We take issues all the time right here on GitHub. We use labels extensively to show the progress through the fixing process. Question issues are okay but make sure to close the issue when it has been answered! Off-topic and '+1' comments will be deleted. Please use post reactions for this purpose.

[![issues](https://img.shields.io/github/issues/nektro/andesite.svg)](https://github.com/nektro/andesite/issues)

When making a pull request, please have it be associated with an issue and make a comment on the issue saying that you're working on it so everyone else knows what's going on :D

[![pulls](https://img.shields.io/github/issues-pr/nektro/andesite.svg)](https://github.com/nektro/andesite/pulls)

## Donate
[![buymeacoffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/nektro)

## Contact
- hello@nektro.net
- Meghan#2032 on discordapp.com
- https://twitter.com/nektro

## License
Apache 2.0
