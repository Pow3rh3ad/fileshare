package commands

import (
	"fileshare/p2p"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a file allowing Peers to download it",
	Run:   publishFile,
}

func init() {
	publishCmd.Flags().StringVarP(&filePath, "filePath", "f", "", "path of file to publish")
	publishCmd.MarkFlagRequired("filePath")
	rootCmd.MarkFlagFilename("filePath")
}

func publishFile(cmd *cobra.Command, args []string) {

	db, err := p2p.CreateDatabase(dbPath, verbose)
	if err != nil {
		log.Errorf("Failed to create db. Reason: %s", err)
		return
	}
	fs := afero.NewOsFs()
	err = p2p.Publish(fs, db, filePath)
	if err != nil {
		log.Errorf("Failed to publish file %s. Reason: %s", filePath, err)
		return
	}
}
