package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goback",
	Short: "Interact with the GoBack backup solution",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start backup deamon",
	Example: `  goback start
  goback start -config /etc/goback`,
	RunE: func(cmd *cobra.Command, args []string) error {
		println("Starting GoBack")

		return nil
	},
}

var decryptCmd = &cobra.Command{
	Use:     "decrypt <archive> [spec]",
	Short:   "Decrypt a archive",
	Example: "goback decrypt data.enc",
	Args:    cobra.RangeArgs(1, 2), // Require 1 arg, allow up to 2
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

		// Optional spec argument
		spec := ""
		if len(args) > 1 {
			spec = args[1]
		}

		println(file, spec)

		return nil
	},
}

var configPath string

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to configuration file (optional)")

	rootCmd.AddCommand(decryptCmd)
	// TODO: Also add configPath here when not using default location?
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
