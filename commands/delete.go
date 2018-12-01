package commands

import (
	"fileshare/p2p"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "deletes a file meta, doesn't delete the physical file",
	Run:   deleteFile,
}

func init() {
	deleteCmd.Flags().StringVarP(&filePath, "fileName", "f", "", "name of file to delete meta")
	deleteCmd.MarkFlagRequired("fileName")
}

func deleteFile(cmd *cobra.Command, args []string) {
	db, err := p2p.CreateDatabase(dbPath, verbose)
	if err != nil {
		log.Errorf("Failed to create db. Reason: %s", err)
	}
	err = p2p.Delete(db, filePath)
	if err != nil {
		log.Errorf("Failed to delete file. Reason: %s", err)
	}
}
