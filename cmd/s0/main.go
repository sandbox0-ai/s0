package main

import (
	"s0/internal/commands"
)

var (
	// Version is set by ldflags during build.
	Version = "dev"
)

func main() {
	commands.Execute(Version)
}
