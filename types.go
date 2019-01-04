package main

type Config struct {
	Discord struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	} `json:"discord"`
}

type OAuth2CallBackResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type DiscordAPIMeResponse struct {
	Username      string `json:"username"`
	Locale        string `json:"locale"`
	PremiumType   int    `json:"premium_type"`
	Multifactor   bool   `json:"mfa_enabled"`
	Flags         int    `json:"flags"`
	Avatar        string `json:"avatar"`
	Discriminator string `json:"discriminator"`
	ID            string `json:"id"`
}

type PragmaTableInfoRow struct {
	cid          int
	name         string
	rowType      string
	notnull      bool
	defaultValue interface{}
	pk           int
}

type UserAccessRow struct {
	id   int
	user int
	path string
}

type UserRow struct {
	id        int
	snowflake string
	admin     bool
}
