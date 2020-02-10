package web

import (
	"net/http"

	"github.com/nektro/andesite/web/handler"
)

func NewMuxer() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/test", handler.HandleTest)

	return mux
}
