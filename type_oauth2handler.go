package main

type Oauth2Provider struct {
	authorizeURL string
	tokenURL     string
	meURL        string
	scope        string
	dbPrefix     string
	nameProp     string
	namePrefix   string
}

var (
	Oauth2Providers = map[string]Oauth2Provider{
		"discord": Oauth2Provider{
			"https://discordapp.com/api/oauth2/authorize",
			"https://discordapp.com/api/oauth2/token",
			"https://discordapp.com/api/users/@me",
			"identify",
			"",
			"username",
			"@",
		},
		"reddit": Oauth2Provider{
			"https://old.reddit.com/api/v1/authorize",
			"https://old.reddit.com/api/v1/access_token",
			"https://oauth.reddit.com/api/v1/me",
			"identity",
			"1:",
			"name",
			"u/",
		},
		"github": Oauth2Provider{
			"https://github.com/login/oauth/authorize",
			"https://github.com/login/oauth/access_token",
			"https://api.github.com/user",
			"read:user",
			"2:",
			"login",
			"@",
		},
		"twitch": Oauth2Provider{
			"https://id.twitch.tv/oauth2/authorize",
			"https://id.twitch.tv/oauth2/token",
			"https://api.twitch.tv/kraken/user",
			"user_read",
			"3:",
			"name",
			"",
		},
		"twitter": Oauth2Provider{
			"https://api.twitter.com/oauth/authorize",
			"https://api.twitter.com/oauth/access_token",
			"https://api.twitter.com/1.1/account/verify_credentials.json?skip_status=true&include_email=false",
			"",
			"4:",
			"screen_name",
			"",
		},
	}
)
