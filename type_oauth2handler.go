package main

type Oauth2Provider struct {
	authorizeURL string
	scope        string
	tokenURL     string
	meURL        string
	dbPrefix     string
	nameProp     string
	namePrefix   string
}

var (
	Oauth2Providers = map[string]Oauth2Provider{
		"discord": Oauth2Provider{
			"https://discordapp.com/api/oauth2/authorize",
			"identify",
			"https://discordapp.com/api/oauth2/token",
			"https://discordapp.com/api/users/@me",
			"",
			"username",
			"@",
		},
		"reddit": Oauth2Provider{
			"https://old.reddit.com/api/v1/authorize",
			"identity",
			"https://old.reddit.com/api/v1/access_token",
			"https://oauth.reddit.com/api/v1/me",
			"1:",
			"name",
			"u/",
		},
		"github": Oauth2Provider{
			"https://github.com/login/oauth/authorize",
			"read:user",
			"https://github.com/login/oauth/access_token",
			"https://api.github.com/user",
			"2:",
			"login",
			"@",
		},
	}
)
