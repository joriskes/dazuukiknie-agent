# Dazuukiknie Agent
Small tray app written in GO that keeps track of the active window automatically

## Install
Run `go get`

## Build for dist
`CGO_ENABLED=1 go build -o my-tray-app`

## Build issues
If you get this on building:
```
# github.com/getlantern/systray
# [pkg-config --cflags  -- ayatana-appindicator3-0.1]
Package ayatana-appindicator3-0.1 was not found in the pkg-config search path.
Perhaps you should add the directory containing `ayatana-appindicator3-0.1.pc'
to the PKG_CONFIG_PATH environment variable
Package 'ayatana-appindicator3-0.1', required by 'virtual:world', not found
```
Install `libayatana-appindicator3-dev`
(`sudo apt-get install libayatana-appindicator3-dev`)