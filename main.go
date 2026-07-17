package main

import (
	"sgwnotify/cmd"
	"sgwnotify/internal/sgwnotify"
)

func main() {
	if err := cmd.Execute(); err != nil {
		sgwnotify.NotifyError(err)
		sgwnotify.ExitWithError(err)
	}
}
