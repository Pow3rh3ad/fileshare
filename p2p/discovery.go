package p2p

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/schollz/peerdiscovery"
)

// DiscoveryPayload sent between peers to recive info about the other peer
type DiscoveryPayload struct {
	Name string
	Addr string
	Port int
}

// SimplePeerDiscovery is a simple discovery package using mdns to locate peers in the LAN network
type SimplePeerDiscovery struct {
	Payload       DiscoveryPayload
	ClientFactory CreateClient
}

// Discover remote nodes on the LAN network using peerdiscovery package
func (p SimplePeerDiscovery) Discover(ctx context.Context) ([]Client, error) {
	discoveries, err := peerdiscovery.Discover(peerdiscovery.Settings{Limit: 0, AllowSelf: false, TimeLimit: 2 * time.Second,
		Payload: encodePayload(p.Payload)})
	if err != nil {
		log.Errorf("Failed to discover. Reason: %s", err)
		return nil, err
	}
	if len(discoveries) == 0 {
		return nil, errors.New("Didn't find any peers")
	}
	var clients []Client
	for _, d := range discoveries {
		payload := decodePayloadFromBytes(d.Payload)
		// Skip entries that didn't return any payload
		if payload.Name == "" {
			continue
		}
		log.Debugf("Connecting to %s@%s:%d", payload.Name, payload.Addr, payload.Port)
		client, err := p.ClientFactory(payload.Name, payload.Addr, payload.Port)
		if err != nil {
			log.Errorf("Failed to create client %s@%s:%d. Reason: %s", payload.Name, payload.Addr, payload.Port, err)
			continue
		}
		clients = append(clients, client)
	}
	return clients, nil
}

// Listen allows to be discoverable for 1 hour
func (p SimplePeerDiscovery) Listen(ctx context.Context, address string) {
	log.Info("Node is now discoverable")
	peerdiscovery.Discover(peerdiscovery.Settings{TimeLimit: time.Hour * 1, Payload: encodePayload(p.Payload)})
	ctx.Done()
}

// encodePayload encodes payload into bytes to send over remote network
func encodePayload(dp DiscoveryPayload) []byte {
	var buffer bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&buffer) // Will write to network.
	// Encode (send) the value.
	err := enc.Encode(dp)
	if err != nil {
		log.Fatal("encode error:", err)
	}
	return buffer.Bytes()
}

// decodePayloadFromBytes decodes bytes to receive the base info of the remote peer
func decodePayloadFromBytes(buf []byte) DiscoveryPayload {
	buffer := bytes.NewBuffer(buf)
	dec := gob.NewDecoder(buffer) // Will write to network
	var dp DiscoveryPayload
	err := dec.Decode(&dp)
	if err != nil {
		log.Fatal("decode error:", err)
	}
	return dp
}
