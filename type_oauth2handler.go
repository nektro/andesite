package main

type Oauth2Provider struct {
	oa2baseURL   string
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
			"https://discordapp.com/api",
			"/oauth2/authorize",
			"identify",
			"/oauth2/token",
			"/users/@me",
			"",
			"username",
			"@",
		},
		"reddit": Oauth2Provider{
			"",
			"https://old.reddit.com/api/v1/authorize",
			"identity",
			"https://old.reddit.com/api/v1/access_token",
			"https://oauth.reddit.com/api/v1/me",
			"1:",
			"name",
			"u/",
		},
	}
)
