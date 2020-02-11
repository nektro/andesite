module github.com/nektro/andesite/rclone

go 1.13

replace github.com/nektro/andesite => ../.

require (
	github.com/aymerick/raymond v2.0.2+incompatible
	github.com/nektro/andesite v0.0.0-00010101000000-000000000000
	github.com/nektro/go-util v0.0.0-20200203203911-b451a8e1cfbf
	github.com/nektro/go.etc v0.0.0-20200210225604-b3480a899bc9
	github.com/nektro/go.oauth2 v0.0.0-20200210165244-d662d0610bec // indirect
	github.com/rclone/rclone v1.51.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
)
