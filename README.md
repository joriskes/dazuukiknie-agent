# Dazuukiknie Agent
Small tray app written in GO that keeps track of the active window automatically

## Install
Run `go get`

Install [go-winres](go install github.com/tc-hib/go-winres@latest) using `go install github.com/tc-hib/go-winres@latest`

Run `go-winres make` do make the syso files. Rerun this when you update something in the `winres` directory

## Build for dist
Build with the flags to signal Windows that it's not a command line app
`go build -ldflags="-H windowsgui"`


