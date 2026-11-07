package main

import (
	"github.com/tx7do/go-wind-toolkit/gowind/internal/clientcmd"
)

func init() {
	rootCmd.AddCommand(clientcmd.CmdClient)
}
