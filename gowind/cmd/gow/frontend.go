package main

import (
	"github.com/tx7do/go-wind-toolkit/gowind/internal/frontend"
)

func init() {
	rootCmd.AddCommand(frontend.CmdFrontend)
}
