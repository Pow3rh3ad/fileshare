package commands

import (
	"context"
	"fileshare/p2p"
	"fileshare/p2p/rpc"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all files available in the network or locally",
	Run:   listFiles,
}

func init() {
	listCmd.Flags().BoolVarP(&localOnly, "local", "l", false, "list only local files")
}

func listFiles(cmd *cobra.Command, args []string) {
	db, err := p2p.CreateDatabase(dbPath, verbose)
	if err != nil {
		log.Errorf("Failed to create db. Reason: %s", err)
		return
	}
	var ff []p2p.FileMetaData
	if localOnly {
		log.Info("Showing local files")
		fs := afero.NewOsFs()
		ff = p2p.List(fs, db)
	} else {
		// Create a remote request
		log.Info("Show files available in network")
		request := p2p.NewRequest(
			"",
			db,
			p2p.SimplePeerDiscovery{Payload: p2p.DiscoveryPayload{}, ClientFactory: rpc.NewClient},
			p2p.NewHighAvailabilityDownloader(1*time.Second),
		)
		ff = request.List(context.Background())
	}
	p2p.PrintFiles(ff)
}
