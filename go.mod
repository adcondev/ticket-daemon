module github.com/adcondev/ticket-daemon

go 1.24.6

require (
	github.com/adcondev/poster v0.0.0
	github.com/coder/websocket v1.8.14
	github.com/google/uuid v1.6.0
	github.com/judwhite/go-svc v1.2.1
	golang.org/x/crypto v0.48.0
)

require (
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/yeqown/go-qrcode/v2 v2.2.5 // indirect
	github.com/yeqown/go-qrcode/writer/standard v1.3.0 // indirect
	github.com/yeqown/reedsolomon v1.0.0 // indirect
	golang.org/x/image v0.34.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)

replace github.com/adcondev/poster => ./mock_poster
