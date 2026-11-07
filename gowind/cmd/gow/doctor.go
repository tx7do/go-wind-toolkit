package main

import (
	"github.com/tx7do/go-wind-toolkit/gowind/internal/doctor"
)

func init() {
	rootCmd.AddCommand(doctor.CmdDoctor)
}
