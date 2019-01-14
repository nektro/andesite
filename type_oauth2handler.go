package main

type Oauth2Provider struct {
	oa2baseURL   string
	authorizeURL string
	scope        string
	tokenURL     string
	meURL        string
	dbPrefix     string
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
		},
	}
)
