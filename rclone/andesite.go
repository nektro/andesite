package main

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aymerick/raymond"
	etc "github.com/nektro/go.etc"
	"github.com/rclone/rclone/cmd"
	"github.com/rclone/rclone/cmd/serve"
	rFs "github.com/rclone/rclone/fs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/nektro/andesite/config"
	"github.com/nektro/andesite/db"
	"github.com/nektro/andesite/fs"
	"github.com/nektro/andesite/search"
	"github.com/nektro/andesite/web"
	"github.com/nektro/andesite/web/handler"

	. "github.com/nektro/go-util/alias"
	. "github.com/nektro/go-util/util"

	_ "github.com/nektro/andesite/statik"
)

func init() {
	flags := Command.Flags()

	flags.IntVar(&config.Config.Version, "version", config.RequiredConfigVersion, "Config version to use.")
	flags.IntVar(&config.Config.Port, "port", 8000, "Port to open server on")
	flags.StringVar(&config.Config.HTTPBase, "base", "/", "Http Origin Path")
	flags.StringVar(&config.Config.Public, "public", "", "Public root of files to serve")

	serve.Command.AddCommand(Command)
}

// Command definition for cobra
var Command = &cobra.Command{
	Use:   "andesite remote:path",
	Short: `Serve the remote over Andesite.`,
	Run: func(command *cobra.Command, args []string) {
		cmd.CheckArgs(1, 1, command, args)
		f := cmd.NewFsSrc(args)
		cmd.Run(false, true, command, func() error {

			s := newServer()

			handler.FS = &rcloneFs{
				f,
			}

			err := s.ListenAndServe()
			if err != nil {
				return err
			}

			return nil
		})
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		rFs.Logf(nil, "Gracefully shutting down...")

		rFs.Logf(nil, "Saving database to disk")
		etc.Database.Close()

		rFs.Logf(nil, "Done!")
	},
}

type rcloneFs struct {
	rFs.Fs
}

func (r *rcloneFs) Stat(name string) (os.FileInfo, error) {
	return r.Stat(name)
}

func (r *rcloneFs) Open(name string) (fs.FileHandle, error) {
	return r.Open(name)
}

func newServer() *http.Server {
	rFs.Logf(nil, "Initializing Andesite %s...", config.Version)
	flagDGS := pflag.String("discord-guild-id", "", "")
	flagDBT := pflag.String("discord-bot-token", "", "")
	etc.PreInit("andesite")

	etc.Init("andesite", &config.Config, "./files/", db.HelperOA2SaveInfo)

	//
	for i, item := range config.Config.Clients {
		if item.For == "discord" {
			if len(*flagDGS) > 0 {
				config.Config.Clients[i].Extra1 = *flagDGS
			}
			if len(*flagDBT) > 0 {
				config.Config.Clients[i].Extra2 = *flagDBT
			}
		}
	}

	//
	if config.Config.Version == 0 {
		config.Config.Version = 1
	}
	if config.Config.Version != config.RequiredConfigVersion {
		DieOnError(
			E(F("Current config.Config.json version '%d' does not match required version '%d'.", config.Config.Version, config.RequiredConfigVersion)),
			F("Visit https://github.com/nektro/andesite/blob/master/docs/config/v%d.md for more info.", config.RequiredConfigVersion),
		)
	}

	// database initialization
	db.Init()

	// database upgrade (removing db prefixes in favor of provider column)
	db.Upgrade()

	// initialize filesystem watching
	if config.Config.SearchOn {
		go search.InitFsWatcher()
	}

	// handlebars helpers
	raymond.RegisterHelper("url_name", func(x string) string {
		return strings.Replace(url.PathEscape(x), "%2F", "/", -1)
	})
	raymond.RegisterHelper("add_i", func(a, b int) int {
		return a + b
	})

	// http server setup and launch
	muxer := web.NewMuxer()

	if len(config.Config.Root) > 0 {
		web.RegisterPrivate(muxer)
	}

	if len(config.Config.Public) > 0 {
		web.RegisterPublic(muxer)
	}

	handlerWrapper := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Server", "nektro/andesite")
		})
	}

	DieOnError(Assert(IsPortAvailable(config.Config.Port), F("Binding to port %d failed.", config.Config.Port)), "It may be taken or you may not have permission to. Aborting!")
	p := strconv.Itoa(config.Config.Port)
	rFs.Logf(nil, "Initialization complete. Starting server on port "+p)

	return &http.Server{
		Addr:    ":" + p,
		Handler: handlerWrapper(muxer),
	}
}
