package commands

import (
	"context"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"fileshare/p2p"
	"fileshare/p2p/rpc"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seeds all avilable files, and becomes discoverable for other remote nodes.",
	Run:   seedFiles,
}

func init() {
	seedCmd.Flags().StringVarP(&serviceName, "name", "n", hostname, "name of service")
	seedCmd.Flags().IntVarP(&port, "port", "p", 7979, "port to listen")
	seedCmd.Flags().DurationVarP(&listenTimout, "time", "t", 60*time.Minute, "how long to listen, i.e 10s 30m 1h")
	seedCmd.Flags().StringVarP(&listenAddress, "address", "a", "", "address to listen")
	seedCmd.Flags().BoolVarP(&seedPartial, "allowPartial", "s", false, "allow partial files / paused files to be seeded")
	seedCmd.MarkFlagRequired("address")
}

func seedFiles(cmd *cobra.Command, args []string) {
	service := rpc.NewNode(serviceName, dbPath, verbose)
	ctx, cancel := context.WithTimeout(context.Background(), listenTimout)
	// Seed will cancel and stop after listen timeout expires
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	// Run seed in a goroutine
	r := &p2p.SimplePeerDiscovery{Payload: p2p.DiscoveryPayload{Name: serviceName, Addr: listenAddress, Port: port}, ClientFactory: rpc.NewClient}
	go service.Seed(ctx, r, listenAddress, port, seedPartial)
	select {
	case <-c:
		cancel()
	case <-ctx.Done():
		log.Info("Timeout deadline reached, stopping seed") // prints "context deadline exceeded"
	}
}
