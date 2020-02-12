package itypes

type UserAccess struct {
	ID   int64  `json:"id"`
	User int64  `json:"user" sqlite:"int"`
	Path string `json:"path" sqlite:"text"`
}
