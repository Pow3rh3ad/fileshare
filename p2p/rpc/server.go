package rpc

import (
	"errors"
	"fileshare/p2p"
	fmt "fmt"
	"io"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/jinzhu/gorm"
	"github.com/spf13/afero"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Node is a mini-rpc server that satisifies the p2p.Client interface
type Node struct {
	ServiceName string
	fs          afero.Fs
	db          *gorm.DB
	seedPartial bool
}

// NewNode creates a new Node to serve incoming requests on the network
func NewNode(name, dbPath string, verbose bool) *Node {
	db, err := p2p.CreateDatabase(dbPath, verbose)
	if err != nil {
		log.Fatalf("Failed to create Database. Reason: %s", err)
		return nil
	}
	return &Node{name, afero.NewOsFs(), db, false}
}

// ================================================================================================================= //
// *										Server interface implementation										   * //
// ================================================================================================================= //

// Seed listens and becomes discoverable for incoming list/downlaod requests on the share network
func (r *Node) Seed(ctx context.Context, resolver p2p.PeerResolver, addr string, port int, seedPartial bool) {
	// seed paused / partial files if requested by user
	if seedPartial {
		r.seedPartial = true
	}
	log.Infof("Listening on %s:%d", addr, port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	RegisterFileServiceServer(server, r)
	// Only listen so other peers will be able to discover this node
	go resolver.Listen(ctx, addr)
	// start listening
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// ================================================================================================================= //
// *										gRPC interface implementation										   * //
// ================================================================================================================= //

// RemoteList satisfies P2PClient's List request
func (r *Node) RemoteList(ctx context.Context, l *ListRequest) (*ListReply, error) {

	log.Infof("Received list request")
	ff := p2p.List(r.fs, r.db)
	reply := ListReply{}
	for _, f := range ff {
		// Skip finished files (unpublished) and files that aren't seeding or partial unless requested to be allowed
		if f.Status == p2p.Finished || (f.Status != p2p.Seeding && !r.seedPartial) {
			log.Warn("Skipped File(%s), status is %s", f.Name, f.Status)
			continue
		}
		var fargments []int32
		for _, fragment := range f.AvailableFragments {
			fargments = append(fargments, int32(fragment.FragmentID))
		}
		m := MetaData{Name: f.Name,
			Publisher:          r.ServiceName,
			Hash:               f.Hash,
			Size:               f.Size,
			FragmentCount:      int32(f.FragmentsCount),
			AvailableFragments: fargments,
			Status:             Status(f.Status),
		}
		log.Infof("Added file %s (hash=%s, fragments=%d/%d, size=%d, status=%s)",
			m.Name, f.Hash, len(m.AvailableFragments), m.FragmentCount, m.Size, f.Status)
		reply.Files = append(reply.Files, &m)
	}
	return &reply, nil
}

// RemoteDownload satisfies P2PClients download request
func (r *Node) RemoteDownload(ctx context.Context, request *DownloadRequest) (*DownloadReply, error) {
	log.Infof("Received download request for fragment file(hash=%s, fragment=%d)", request.FileHash, request.RequestedFragment)
	var fm p2p.FileMetaData
	if r.db.Where("hash = ?", request.FileHash).First(&fm).RecordNotFound() {
		return nil, errors.New("File Not found")
	}
	// Skip finished files (unpublished) and files that aren't seeding or partial unless requested to be allowed
	if fm.Status == p2p.Finished || (fm.Status != p2p.Seeding && !r.seedPartial) {
		log.Warn("Skipped File(%s), status is %s", fm.Name, fm.Status)
		return nil, errors.New("File not available")
	}
	f, err := r.fs.Open(fm.FilePath)
	if err != nil {
		log.Errorf("Failed to open file %s. Reason: %s", fm.FilePath, err)
		return nil, err
	}
	defer f.Close()
	log.Debugf("Opened file %s", fm.FilePath)
	buffer := make([]byte, p2p.FileChunkSize)
	n, err := f.ReadAt(buffer, int64(request.RequestedFragment*p2p.FileChunkSize))
	log.Debugf("Read %d from %d", n, int64(request.RequestedFragment*p2p.FileChunkSize))
	if err != nil && err != io.EOF {
		log.Errorf("Failed to read. Reason: %s", err)
		return nil, err
	}
	return &DownloadReply{FragmentID: request.RequestedFragment, Data: buffer}, nil
}

// RemoteFragmentsAvailable checks if fragment is available in the server
func (r *Node) RemoteFragmentsAvailable(ctx context.Context, request *FragmentRequest) (*FragmentReply, error) {
	log.Debug("Fragment requested for file(hash=%s)", request.FileHash)
	var fragments []p2p.Fragment
	r.db.Where("hash_id = ?", request.FileHash).Find(&fragments)
	log.Debugf("Found %d fragments for file(hash=%s) found", len(fragments), request.FileHash)
	var fragmentIDs []int32
	for _, f := range fragments {
		fragmentIDs = append(fragmentIDs, int32(f.FragmentID))
	}
	return &FragmentReply{AvailableFragments: fragmentIDs}, nil
}
