package p2p

import (
	"context"
)

// FileChunkSize defines file split size 1mb chunks
const FileChunkSize = 1 * (1 << 20) // 1 MB, change this to your requirement

// CreateClient defines a factory method that allows to create different types of clients rpc/http/torrent etc
type CreateClient func(name, addr string, port int) (Client, error)

// DownloadMethod is an algorithim for downloading a file from multiple peers
// the method recevies all available peers and all fragments that were already downloaded
// and returns the best fragment to download.
type DownloadMethod interface {
	// NextFragments returns the next fragment to download and from what client
	NextFragment(ctx context.Context, peers map[string]Client, fm FileMetaData) (Client, int, error)
}

// DownloadResult is returned by client when it finishes to download a file part.
type DownloadResult struct {
	FragmentID int
	PeerName   string
	Data       []byte
	Successful bool
}

// Client interface allows to download / list files from a remote node
type Client interface {
	// Name returns unique name of client
	Name() string
	// List Files available files in client
	List(ctx context.Context) ([]FileMetaData, error)
	// Download a fragment of a file from remote client
	Download(ctx context.Context, fileHash string, fragmentID int, out chan DownloadResult)
	// FragmentAvailable checks if fragment is available on remote client
	FragmentsAvailable(ctx context.Context, fileHash string) []int
	// Alive checks if connection is alive
	Alive() bool
}

// Server interface allows for remote file seeding with other Client's
type Server interface {
	// Seed allows remote peers to download files from node
	Seed(ctx context.Context, resolver PeerResolver, addr string, port int, seedPartial bool)
}

// PeerResolver discovers remote peers in the network and returns Clients available in the network
type PeerResolver interface {
	// Discovers clients in the network
	Discover(ctx context.Context) ([]Client, error)
	// Listen allows other peers to discover this node
	Listen(ctx context.Context, addr string)
}
