package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/generate"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/project"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/run"
)

var rootCmd = &cobra.Command{
	Use:   "gow",
	Short: "gow CLI",
	Long:  "gow is the CLI for GoWind framework.",
}

func init() {
	rootCmd.AddCommand(project.CmdProject)
	rootCmd.AddCommand(run.CmdRun)
	rootCmd.AddCommand(generate.CmdGenerate)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
