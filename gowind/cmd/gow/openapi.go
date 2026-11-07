package main

import (
	"github.com/tx7do/go-wind-toolkit/gowind/internal/openapi"
)

func init() {
	rootCmd.AddCommand(openapi.CmdOpenAPI)
}
