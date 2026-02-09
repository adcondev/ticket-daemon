module github.com/adcondev/ticket-daemon

go 1.24.6

require (
	github.com/google/uuid v1.6.0
	github.com/judwhite/go-svc v1.2.1
	nhooyr.io/websocket v1.8.17
)

require golang.org/x/sys v0.1.0 // indirect

replace github.com/adcondev/poster => ../poster
