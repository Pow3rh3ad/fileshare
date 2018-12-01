package p2p

import (
	"context"
	"errors"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
)

// Priority is a type name for enum below
type Priority int

const (
	// LowestAvailability downloads fragments that have lowest availability
	LowestAvailability Priority = 0
	// HighestAvailability downloads fragments that have highest availability
	HighestAvailability = 1
)

type availableFragments struct {
	FragmentID int
	Peers      []string
}

// AvailabilityDownloader downloads the fragment with highest availability picking the peer with least requests
type AvailabilityDownloader struct {
	RefreshPeriod time.Duration
	Priority      Priority

	availabilityCount map[int][]availableFragments
	lastRefresh       time.Time
	peerRequests      map[string]int
}

// NewHighAvailabilityDownloader creates a new HighAvailabilityDownloader with a defined refresh period for fragment availability
func NewHighAvailabilityDownloader(refreshPeriod time.Duration) DownloadMethod {
	return &AvailabilityDownloader{refreshPeriod, LowestAvailability, make(map[int][]availableFragments), time.Now().Add(-refreshPeriod),
		make(map[string]int)}
}

// NextFragment - simple aglorithim for downloading from peers
func (h *AvailabilityDownloader) NextFragment(ctx context.Context, peers map[string]Client, fm FileMetaData) (Client, int, error) {
	if time.Since(h.lastRefresh) >= h.RefreshPeriod {
		log.Debug("Fragment Count refresh")
		h.countFragments(ctx, peers, fm)
		h.lastRefresh = time.Now()
	}
	// go from highest availability fragment
	for i := 0; i <= len(peers); i++ {
		for _, f := range h.availabilityCount[i] {
			if fm.FragmentExists(f.FragmentID) {
				continue
			}
			if len(f.Peers) == 0 {
				continue
			}
			// pick peer with lowest request count out of peers that have this fragment
			log.Debugf("Picking from peers %s", f.Peers)
			pn := h.pickPeer(f.Peers...)
			log.Debugf("Downloading fragment(id=%d)@%s", f.FragmentID, pn)
			return peers[pn], f.FragmentID, nil
		}
	}
	return nil, 0, errors.New("No Fragment found to download")
}

// countFragments implements a counting sort for each fragment and to what peers have it.
func (h *AvailabilityDownloader) countFragments(ctx context.Context, peers map[string]Client, fm FileMetaData) {
	// Sorts each fragment and how many peers have that fragment
	// 0: peera,
	// 1: peera, peerb
	// 2: peera, peerc
	// 3: peera, peerb, peerc
	c := make([][]string, fm.FragmentsCount)
	// Count for every peer, finding the "most available chunk" etc
	for _, peer := range peers {
		// if peer is dead for some reason, it will return 0 fragments that are avialable since it's down now.
		af := peer.FragmentsAvailable(ctx, fm.Hash)
		for _, i := range af {
			c[i] = append(c[i], peer.Name())
		}
	}
	// Sort by availability (by how many peers have this fragment)
	// 1 : [(0, peera),]
	// 2 : [(1, peera, peerb), (2, peera, peerc)]
	// 3 : [(3, peera, peerb, peerc)]
	fragmentAvailability := make(map[int][]availableFragments)
	for i, peers := range c {
		availability, ok := fragmentAvailability[len(peers)]
		if !ok {
			fragmentAvailability[len(peers)] = []availableFragments{availableFragments{i, peers}}
		} else {
			fragmentAvailability[len(peers)] = append(availability, availableFragments{i, peers})
		}
	}
	h.availabilityCount = fragmentAvailability
}

// pickPeer picks peer with lowest request count
func (h *AvailabilityDownloader) pickPeer(peers ...string) string {
	lowestPeerReq := math.MaxInt64
	var pickedPeer string
	for _, p := range peers {
		req, ok := h.peerRequests[p]
		// Peers was never requested
		if !ok {
			h.peerRequests[p] = 0
			req = 0
		}
		if req < lowestPeerReq {
			lowestPeerReq = req
			pickedPeer = p
		}
	}
	h.peerRequests[pickedPeer]++
	return pickedPeer
}
