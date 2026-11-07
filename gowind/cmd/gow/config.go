package main

import (
	"github.com/tx7do/go-wind-toolkit/gowind/internal/configcmd"
)

func init() {
	rootCmd.AddCommand(configcmd.CmdConfig)
}
