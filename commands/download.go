package commands

import (
	"context"
	"fileshare/p2p"
	"fileshare/p2p/rpc"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download file from remote nodes",
	Long:  "Downloads a file from remote nodes into specified download directory",
	Run:   downloadFile,
}

func init() {
	downloadCmd.Flags().StringVarP(&fileHash, "fileHash", "f", "", "hash of file to download")
	downloadCmd.Flags().StringVarP(&dlPath, "download", "p", "", "directory to download files to")
	downloadCmd.MarkFlagRequired("fileHash")
}

func downloadFile(cmd *cobra.Command, args []string) {

	db, err := p2p.CreateDatabase(dbPath, false)
	if err != nil {
		log.Errorf("Failed to create db. Reason: %s", err)
		return
	}
	request := p2p.NewRequest(dlPath, db, p2p.SimplePeerDiscovery{Payload: p2p.DiscoveryPayload{}, ClientFactory: rpc.NewClient},
		p2p.NewHighAvailabilityDownloader(time.Second*1))
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
			log.Info("Context done")
		}
	}()
	fs := afero.NewOsFs()
	request.Download(ctx, fs, fileHash)
}
