#!/bin/bash
mv ./go.mod ./go.mod.bac
GOPATH=$PWD GOOS=linux GOARCH=arm go build -ldflags="-s -w -X main.Version=0.0.1" -o tftp-server
mv ./go.mod.bac ./go.mod