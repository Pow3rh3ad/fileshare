package p2p_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"fileshare/p2p"
	"fileshare/p2p/mocks"

	"github.com/stretchr/testify/assert"
)

func TestOnePeerDownload(t *testing.T) {
	fragments := []p2p.Fragment{}
	fm := p2p.FileMetaData{"Test", "C:\\file\\path\test.exe", "TestPub", "13c405d80e97aa7b46d3389180b19eb3", 666, 2, fragments, 1}
	dl := p2p.NewHighAvailabilityDownloader(time.Second * 30)
	client := &mocks.Client{}
	client.On("Name").Return("testClient")
	client.On("FragmentsAvailable", mock.Anything, mock.AnythingOfType("string")).Return([]int{0, 1}).Times(3)
	peers := make(map[string]p2p.Client)
	peers[client.Name()] = client
	ctx := context.Background()
	c, i, err := dl.NextFragment(ctx, peers, fm)
	assert.Nil(t, err)
	assert.Equal(t, c, client)
	assert.Equal(t, 0, i)
	fm.AvailableFragments = append(fm.AvailableFragments, p2p.Fragment{0, ""})
	c, i, err = dl.NextFragment(ctx, peers, fm)
	assert.Nil(t, err)
	assert.Equal(t, 1, i)
	fm.AvailableFragments = append(fm.AvailableFragments, p2p.Fragment{1, ""})
	c, i, err = dl.NextFragment(ctx, peers, fm)
	assert.Error(t, err)
	// Test Case where we have 2 fragments but peer has only 1, we expect after 1 iteration that it will fail
	client.On("FragmentsAvailable", mock.Anything, mock.AnythingOfType("string")).Return([]int{1}).Twice()
	fm.AvailableFragments = []p2p.Fragment{}
	c, i, err = dl.NextFragment(ctx, peers, fm)
	assert.Nil(t, err)
	assert.Equal(t, c, client)
	assert.Equal(t, 0, i)
	fm.AvailableFragments = append(fm.AvailableFragments, p2p.Fragment{1, ""})
	c, i, err = dl.NextFragment(ctx, peers, fm)
	assert.Nil(t, err)
}

// Test download algorithim with two peers both have all fragments, so load should be evenly distributed
func TestTwoPeerDownload(t *testing.T) {
	peerCount := make(map[string]int)
	fragments := []p2p.Fragment{}
	// We will create a file with 4 fragments
	fm := p2p.FileMetaData{"Test", "C:\\file\\path\test.exe", "TestPub", "13c405d80e97aa7b46d3389180b19eb3", 666, 4, fragments, 1}
	dl := p2p.NewHighAvailabilityDownloader(time.Second * 30)
	client1 := &mocks.Client{}
	client1.On("Name").Return("testClient1")
	client1.On("FragmentsAvailable", mock.Anything, mock.AnythingOfType("string")).Return([]int{0, 1, 2, 3})
	client2 := &mocks.Client{}
	client2.On("Name").Return("testClient2")
	client2.On("FragmentsAvailable", mock.Anything, mock.AnythingOfType("string")).Return([]int{0, 1, 2, 3})
	peers := make(map[string]p2p.Client)
	peers[client1.Name()] = client1
	peers[client2.Name()] = client2
	ctx := context.Background()
	for fid := 0; fid < fm.FragmentsCount; fid++ {
		c, i, err := dl.NextFragment(ctx, peers, fm)
		assert.Nil(t, err)
		assert.Equal(t, fid, i)
		peerCount[c.Name()] += 1
		fm.AvailableFragments = append(fm.AvailableFragments, p2p.Fragment{fid, ""})
	}
	assert.Equal(t, peerCount[client1.Name()], peerCount[client2.Name()])
	_, _, err := dl.NextFragment(ctx, peers, fm)
	assert.Error(t, err)
}

func TestTwoPeerDownloadMissingFiles(t *testing.T) {
	peerCount := make(map[string]int)
	fragments := []p2p.Fragment{}
	// We will create a file with 4 fragments
	fm := p2p.FileMetaData{"Test", "C:\\file\\path\test.exe", "TestPub", "13c405d80e97aa7b46d3389180b19eb3", 666, 4, fragments, 1}
	dl := p2p.NewHighAvailabilityDownloader(time.Second * 30)
	client1 := &mocks.Client{}
	client1.On("Name").Return("testClient1")
	client1.On("FragmentsAvailable", mock.Anything, mock.AnythingOfType("string")).Return([]int{0, 1, 2, 3})
	client2 := &mocks.Client{}
	client2.On("Name").Return("testClient2")
	client2.On("FragmentsAvailable", mock.Anything, mock.AnythingOfType("string")).Return([]int{0, 3})
	peers := make(map[string]p2p.Client)
	peers[client1.Name()] = client1
	peers[client2.Name()] = client2
	ctx := context.Background()
	expectedFragment := []int{1, 2, 0, 3}
	for fid := 0; fid < fm.FragmentsCount; fid++ {
		c, i, err := dl.NextFragment(ctx, peers, fm)
		assert.Nil(t, err)
		assert.Equal(t, expectedFragment[fid], i)
		peerCount[c.Name()] += 1
		fm.AvailableFragments = append(fm.AvailableFragments, p2p.Fragment{i, ""})
	}
	assert.Equal(t, peerCount[client1.Name()], peerCount[client2.Name()])
	_, _, err := dl.NextFragment(ctx, peers, fm)
	assert.Error(t, err)
}
