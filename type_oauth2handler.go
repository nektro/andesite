package main

type Oauth2Provider struct {
	authorizeURL string
	tokenURL     string
	meURL        string
	scope        string
	nameProp     string
	dbPrefix     string
	namePrefix   string
}

var (
	Oauth2Providers = map[string]Oauth2Provider{
		"discord": Oauth2Provider{
			"https://discordapp.com/api/oauth2/authorize",
			"https://discordapp.com/api/oauth2/token",
			"https://discordapp.com/api/users/@me",
			"identify",
			"username",
			"",
			"@",
		},
		"reddit": Oauth2Provider{
			"https://old.reddit.com/api/v1/authorize",
			"https://old.reddit.com/api/v1/access_token",
			"https://oauth.reddit.com/api/v1/me",
			"identity",
			"name",
			"1:",
			"u/",
		},
		"github": Oauth2Provider{
			"https://github.com/login/oauth/authorize",
			"https://github.com/login/oauth/access_token",
			"https://api.github.com/user",
			"read:user",
			"login",
			"2:",
			"@",
		},
		"google": Oauth2Provider{
			"https://accounts.google.com/o/oauth2/v2/auth",
			"https://www.googleapis.com/oauth2/v4/token",
			"https://www.googleapis.com/oauth2/v1/userinfo?alt=json",
			"profile",
			"name",
			"3:",
			"",
		},
		"facebook": Oauth2Provider{
			"https://graph.facebook.com/oauth/authorize",
			"https://graph.facebook.com/oauth/access_token",
			"https://graph.facebook.com/me",
			"",
			"name",
			"4:",
			"",
		},
	}
)
