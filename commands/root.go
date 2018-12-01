package commands

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Variables used in flags.
var (
	localOnly     bool
	filePath      string
	fileHash      string
	dlPath        string
	dbPath        string
	serviceName   string
	port          int
	verbose       bool
	listenAddress string
	listenTimout  time.Duration
	seedPartial   bool
)

var hostname, _ = os.Hostname()

var rootCmd = &cobra.Command{
	Use:   "fileshare",
	Short: "FileShare allows efficent file sharing in a local area network",
	Long:  `A Fast and Flexible P2P file sharing library `,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
			log.Debug("Debugging mode set")
		}
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)
	rootCmd.AddCommand(unpublishCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(seedCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(deleteCmd)
	// Add db flag, database is required for all commands to work
	rootCmd.PersistentFlags().StringVarP(&dbPath, "db", "d", "fileshare.db", "database path")
	rootCmd.MarkFlagFilename("db")
	// Add verbose flag
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// Execute main root command line
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
