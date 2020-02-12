package itypes

//
type UserAccess struct {
	ID   int64  `json:"id"`
	User int64  `json:"user" sqlite:"int"`
	Path string `json:"path" sqlite:"text"`
}

//
type Share struct {
	ID   int64  `json:"id"`
	Hash string `json:"hash" sqlite:"text"` // character(32)
	Path string `json:"path" sqlite:"text"`
}

//
type DiscordRoleAccess struct {
	ID        int64  `json:"id"`
	GuildID   string `json:"guild_snowflake" sqlite:"text"`
	RoleID    string `json:"role_snowflake" sqlite:"text"`
	Path      string `json:"path" sqlite:"text"`
	GuildName string `json:"guild_name" sqlite:"text"`
	RoleName  string `json:"role_name" sqlite:"text"`
}
