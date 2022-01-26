SET VERSION=0.0.1

go build -ldflags="-s -w -X main.Version=%VERSION%"
go build -ldflags="-s -w -X main.Version=%VERSION%" client/tftp-client.go