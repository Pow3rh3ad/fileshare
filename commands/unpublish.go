package commands

import (
	"fileshare/p2p"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var unpublishCmd = &cobra.Command{
	Use:   "unpublish",
	Short: "Unpublish a file, peers won't be able to download file anymore",
	Run:   unpublishFile,
}

func init() {
	unpublishCmd.Flags().StringVarP(&filePath, "fileName", "f", "", "name of file to unpublish")
	unpublishCmd.MarkFlagRequired("fileName")
	rootCmd.MarkFlagFilename("fileName")
}

func unpublishFile(cmd *cobra.Command, args []string) {
	db, err := p2p.CreateDatabase(dbPath, verbose)
	if err != nil {
		log.Errorf("Failed to create db. Reason: %s", err)
	}
	p2p.Unpublish(db, filePath)
}
