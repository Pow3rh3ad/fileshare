package p2p

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jedib0t/go-pretty/progress"
	"github.com/jinzhu/gorm"
	"github.com/spf13/afero"
)

// A Request represents a file transfer request to be sent by a Client.
type Request struct {
	// dlDirectory is where the file will be saved
	dlDirectory string

	// Peers specifies available peers to download requested file from, peers may be added if more are discovered
	peers map[string]Client

	// Resolver specifies the discovery method to find P2PClients, The given resolver defines how peers are added
	// to the request
	resolver PeerResolver

	// db is the connection to the local database of fileshare holding all metadata
	db *gorm.DB

	// rwLock always locking on update on shared items
	rwLock sync.RWMutex

	// downloadMethod is the algorithim called every time a new fragment needs to be downloaded
	dlMethod DownloadMethod
}

// NewRequest creates a new request for download / listing files from remote peers
func NewRequest(dlPath string, db *gorm.DB, resolver PeerResolver, dlMethod DownloadMethod) *Request {
	return &Request{dlPath, make(map[string]Client), resolver, db, sync.RWMutex{}, dlMethod}
}

// List shows all available files in the network, in a specific point, if a peer is offline, his files won't show.
func (r *Request) List(ctx context.Context) []FileMetaData {
	// start discover
	r.startDiscover(ctx, true)
	r.rwLock.RLock()
	var allFiles []FileMetaData
	for _, client := range r.peers {
		files, _ := client.List(ctx)
		allFiles = append(allFiles, files...)
	}
	r.rwLock.RUnlock()
	return allFiles
}

// Download a remote file based on hash to local system from remote peers
func (r *Request) Download(ctx context.Context, fs afero.Fs, fileHash string) {
	fm, err := r.getFileMeta(ctx, fileHash)
	if err != nil {
		return
	}
	// Start Discovery once, before so to populate the initial peers
	r.startDiscover(ctx, true)
	if len(r.peers) == 0 {
		log.Errorf("No peers found, try again later")
		return
	}
	// Look for new peers whilst downloading
	go r.startDiscover(ctx, false)
	// Open file to save downloaded fragments
	f, err := fs.OpenFile(path.Join(r.dlDirectory, fm.Name), os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		log.Errorf("Open file failed. Reason: %s", err)
		return
	}
	if err := f.Truncate(fm.Size); err != nil {
		log.Errorf("Truncate file failed. Reason: %s", err)
		return
	}
	fm.Status = Downloading // Mark file as downloading
	defer r.db.Save(&fm)    // Make sure status will be saved, in any case of transfer / failure etc
	tracker, pw := createProgressBar(int64(FileChunkSize * int(fm.FragmentsCount)))
	fm.Status = r.transfer(ctx, f, fm, tracker, pw)
	time.Sleep(50 * time.Millisecond) // Sleep 50 ms to allow bar to fully update rendering
	if fm.Status == Seeding {
		log.Infof("Finished Downloading %s", fm.Name)
	}
}

func (r *Request) transfer(ctx context.Context, f afero.File, fm FileMetaData, tracker *progress.Tracker, pw progress.Writer) Status {
	out := make(chan DownloadResult, 5)
	// Update tracker based on fragments we already downloaded
	tracker.Increment(int64(len(fm.AvailableFragments) * FileChunkSize))
	time.Sleep(150 * time.Millisecond)
	// Download all missing fragment, every time a fragment is downloaded update meta file
	for {
		// Lock while algorithim is choosing, this preventing any peer changing whilst executing
		r.rwLock.RLock()
		peer, fragmentID, err := r.dlMethod.NextFragment(ctx, r.peers, fm)
		r.rwLock.RUnlock()
		if err != nil {
			pw.Stop()
			if len(fm.AvailableFragments) == fm.FragmentsCount {
				log.Info("No more fragments, seeding..")
				tracker.MarkAsDone()
				return Seeding
			}
			if ctx.Err() == context.Canceled {

				log.Warnf("Download was interrupted, restart the program")
			} else {
				log.Warnf("Fragments were missing, download is paused, try again later")
			}
			return Paused
		}
		go peer.Download(ctx, fm.Hash, fragmentID, out)
		// Wait for downloaded chunk / interrupt
		select {
		case dl := <-out:
			if !dl.Successful {
				log.Debugf("Download was unssuccesful for fragment(%d)@%s", dl.FragmentID, dl.PeerName)
				continue
			}
			fm.AvailableFragments = append(fm.AvailableFragments, Fragment{FragmentID: dl.FragmentID, HashID: fm.Hash})
			// Save fragment to db and write to file
			r.db.Save(&Fragment{FragmentID: dl.FragmentID, HashID: fm.Hash})
			f.WriteAt(dl.Data, int64(FileChunkSize*int(dl.FragmentID)))
			tracker.Increment(FileChunkSize)
			time.Sleep(50 * time.Millisecond)
		case <-ctx.Done():
			return Paused
		}
	}
}

func (r *Request) getFileMeta(ctx context.Context, fileHash string) (FileMetaData, error) {
	var fm FileMetaData
	if !r.db.Where("hash = ?", fileHash).First(&fm).RecordNotFound() {
		log.Info("Will resume pervious download request")
		r.db.Model(&fm).Related(&fm.AvailableFragments, "hash_id")
		return fm, nil
	}
	log.Infof("Meta file(hash=%s) missing, will create a new download", fileHash)
	// List all files from peers, and look for our file, any meta of that file will suffice
	ffm := r.List(ctx)
	for _, m := range ffm {
		if m.Hash != fileHash {
			continue
		}
		log.Debugf("Found file meta name=%s hash=%s fragments=%d ", m.Name, m.Hash, m.FragmentsCount)
		// This is probably a fresh download, so we will assume all fragments are missing
		m.AvailableFragments = make([]Fragment, 0)
		log.Infof("Current Fragments: %v", m.AvailableFragments)
		// FilePath will be our dlDirectory + the file name, that were the meta file will be save aswell
		m.FilePath = path.Join(r.dlDirectory, m.Name)
		m.Status = Downloading
		r.db.Save(&m)
		return m, nil
	}
	log.Errorf("File with hash %s wasn't found in network", fileHash)
	return fm, errors.New("Failed to find file in network")
}

func (r *Request) startDiscover(ctx context.Context, once bool) {
	if r.resolver == nil {
		log.Info("No discovery defined, discovery won't be enabled")
		return
	}
	ticker := time.NewTicker(3 * time.Second)
	log.Info("Started Discovery")
	select {
	case <-ticker.C:
		log.Debug("Looking for peers..")
		p2pClients, _ := r.resolver.Discover(ctx)
		for _, client := range p2pClients {
			log.Debugf("Found Client(%s)", client.Name())
			if _, ok := r.peers[client.Name()]; !ok {
				r.rwLock.Lock()
				log.Infof("Added Client(%s)", client.Name())
				r.peers[client.Name()] = client
				r.rwLock.Unlock()
			}
		}
		if once {
			log.Info("Discovery executed once")
			return
		}
	case <-ctx.Done():
		log.Info("Canceled discovery will stop")
	}
	log.Debug(r.peers)
}

func createProgressBar(total int64) (*progress.Tracker, progress.Writer) {
	// instantiate a Progress Writer and set up the options
	pw := progress.NewWriter()
	pw.SetAutoStop(true)
	pw.SetTrackerLength(25)
	pw.ShowOverallTracker(true)
	pw.ShowTime(true)
	pw.ShowTracker(true)
	pw.ShowValue(true)
	pw.SetMessageWidth(24)
	pw.SetNumTrackersExpected(1)
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleDefault)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsExample
	pw.Style().Options.PercentFormat = "%4.1f%%"

	// call Render() in async mode; if we are in debug level, we won't show tracker so prints will work correct
	if log.GetLevel() < log.DebugLevel {
		log.Info("Progress bar initialized")
		go func() {
			pw.Render()
			runtime.Gosched()
		}()
	}
	tracker := progress.Tracker{Message: fmt.Sprintf("Download Progress"), Total: total, Units: progress.UnitsBytes}
	pw.AppendTracker(&tracker)
	return &tracker, pw
}
