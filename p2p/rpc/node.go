package rpc

import (
	"errors"
	"fileshare/p2p"
	"fmt"

	"google.golang.org/grpc/connectivity"

	log "github.com/sirupsen/logrus"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

// P2PClient of rpc package allows connection to rpc based servers
type P2PClient struct {
	name, addr string
	conn       *grpc.ClientConn
	client     FileServiceClient
}

// NewClient creates a new rpc client to connect to a remote peer
func NewClient(name string, addr string, port int) (p2p.Client, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", addr, port), grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
		return nil, errors.New("Failed to create client")
	}
	client := NewFileServiceClient(conn)
	return &P2PClient{fmt.Sprintf("%s@%d", name, port), addr, conn, client}, nil
}

// List remote files on remote node
func (p *P2PClient) List(ctx context.Context) ([]p2p.FileMetaData, error) {
	response, err := p.client.RemoteList(ctx, &ListRequest{FullDetails: false})
	if err != nil {
		return nil, err
	}
	var files []p2p.FileMetaData
	for _, f := range response.GetFiles() {
		var fragments []p2p.Fragment
		for _, id := range f.AvailableFragments {
			fragments = append(fragments, p2p.Fragment{FragmentID: int(id), HashID: f.Hash})
		}
		files = append(files, p2p.FileMetaData{Name: f.Name, FilePath: "", Publisher: p.Name(),
			Hash: f.Hash, Size: f.Size, FragmentsCount: int(f.FragmentCount),
			AvailableFragments: fragments, Status: p2p.Status(f.Status)})
	}
	return files, nil
}

// Name of client
func (p *P2PClient) Name() string {
	return p.name
}

// Download a fragment of a file from remote client
func (p *P2PClient) Download(ctx context.Context, fileHash string, fragmentID int, out chan p2p.DownloadResult) {
	reply, err := p.client.RemoteDownload(ctx, &DownloadRequest{FileHash: fileHash, RequestedFragment: uint32(fragmentID)})
	if err == nil {
		out <- p2p.DownloadResult{FragmentID: fragmentID, PeerName: p.Name(), Data: reply.Data, Successful: true}
		return
	}
	if ctx.Err() == context.Canceled {
		return
	}
	log.Debug("Failed to download fragment. Reason: %s", err)
	out <- p2p.DownloadResult{FragmentID: fragmentID, PeerName: p.Name(), Data: nil, Successful: false}
}

// FragmentsAvailable checks if fragment is available on remote client
func (p *P2PClient) FragmentsAvailable(ctx context.Context, fileHash string) []int {
	reply, err := p.client.RemoteFragmentsAvailable(ctx, &FragmentRequest{FileHash: fileHash})
	if err != nil {
		log.Debugf("Fail to check available fragments. Reason: %s", err)
		return make([]int, 0)
	}
	var fragmentIDs []int
	for _, i := range reply.AvailableFragments {
		fragmentIDs = append(fragmentIDs, int(i))
	}
	return fragmentIDs
}

// Alive checks if rpc client is alive and ready for commands
func (p *P2PClient) Alive() bool {
	return p.conn.GetState() != connectivity.Shutdown
}
