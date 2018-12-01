package p2p

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/jedib0t/go-pretty/table"
	"github.com/jinzhu/gorm"

	// Main of p2p package

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/spf13/afero"
)

// Status describes state of FileMetaData
type Status uint32

// These are the generalized file operations that can trigger a notification.
const (
	New         Status = 0
	Paused             = 1
	Downloading        = 2
	Finished           = 3
	Seeding            = 4
)

// Allow Status to get a human printable form
func (e Status) String() string {
	switch e {
	case New:
		return "New"
	case Paused:
		return "Paused"
	case Downloading:
		return "Downloading"
	case Finished:
		return "Finished"
	case Seeding:
		return "Seeding"
	default:
		return fmt.Sprintf("%d", int(e))
	}
}

// FileMetaData holds information on the file published for download
type FileMetaData struct {
	// Name of file
	Name string
	// FilePath is the full path of the file
	FilePath string
	// Who is publishing the file, usually the node we are downloading from
	Publisher string
	// Hash is the unique identifier of the file
	Hash string `gorm:"primary_key"`
	// Size of file
	Size int64
	// FragmentsCount specifies the amount of chunks the file is split
	FragmentsCount int
	// AvailbleFragments specifices all fragments available on the file
	AvailableFragments []Fragment `gorm:"foreignkey:Hash"`
	// Status of file PAUSED|DOWNLOADING|FINISHED|SEEDING
	Status Status
}

// FragmentExists checks if a fragment is available on this file
func (fm FileMetaData) FragmentExists(id int) bool {
	for _, f := range fm.AvailableFragments {
		if f.FragmentID == id {
			return true
		}
	}
	return false
}

// Fragment is a single part of a file
type Fragment struct {
	FragmentID int    `gorm:"primary_key;type:INTEGER; DEFAULT:0"`
	HashID     string `gorm:"primary_key"` // This is the hash of the file not the fragment
}

// CreateDatabase create a database connection (sqlite3 based)
func CreateDatabase(dbPath string, verbose bool) (*gorm.DB, error) {
	log.Infof("Creating database connection to %s?cache=shared&mode=rwc", dbPath)
	db, err := gorm.Open("sqlite3", fmt.Sprintf("%s?mode=rwc", dbPath))
	if err != nil {
		log.Errorf("Failed to open db. Reason: %s", err)
		return nil, err
	}
	db.Exec("PRAGMA foreign_keys = ON")
	db.LogMode(verbose)
	db.AutoMigrate(&FileMetaData{}, &Fragment{})
	return db, nil
}

// List Files available locally for downloading from other peers
func List(fs afero.Fs, db *gorm.DB) []FileMetaData {
	var files []FileMetaData
	db.Find(&files)
	log.Debugf("Found %d files", len(files))
	// Add fragments related to each file
	for i := range files {
		if ok, _ := afero.Exists(fs, files[i].FilePath); !ok {
			log.Warnf("File(%s) doesn't exist in given path, skipping..", files[i].FilePath)
		}
		log.Debugf("Finding available fragments for file %s", files[i].Name)
		db.Model(&files[i]).Related(&files[i].AvailableFragments, "hash_id")
	}
	return files
}

// Publish a file to be available for sharing, files that aren't published won't show in list or be availble in when seeding.
func Publish(fs afero.Fs, db *gorm.DB, filePath string) error {
	ok, _ := afero.Exists(fs, filePath)
	if !ok {
		log.Errorf("File to publish %s doesn't exist", filePath)
		return os.ErrNotExist
	}
	log.Info("Checking if meta file exists in database")
	var fm FileMetaData
	var err error
	if db.Where("file_path = ?", filePath).First(&fm).RecordNotFound() {
		log.Debug("Meta doesn't exist in database, building meta file")
		fm, err = createMetaFile(fs, filePath)
		if err != nil {
			return err
		}
		// Save fragments information
		for _, fragment := range fm.AvailableFragments {
			log.Debugf("Saving fragment id=%d", fragment.FragmentID)
			db.Save(&fragment)
		}
	} else {
		log.Debug("Meta file exists, only updating status")
	}
	// Don't update files that are paused or downloading.
	if fm.Status == Paused || fm.Status == Downloading {
		return nil
	}
	fm.Status = Seeding
	log.Info("Saving file meta data to database")
	err = db.Save(&fm).Error
	if err == nil {
		log.Info("File saved successfully")
	}
	return err
}

// Unpublish a file to become unavailble for downloading anymore
func Unpublish(db *gorm.DB, fileName string) {
	log.Infof("Unpublishing file(name=%s)", fileName)
	var fm FileMetaData
	if db.Where("name = ?", fileName).First(&fm).RecordNotFound() {
		log.Debug("Meta file doesn't exist, file is counted as unpublished")
		return
	}
	if fm.Status == Paused {
		log.Warn("file isn't completed, can't be marked as finished")
		return
	}
	log.Info("Changing file status to finished")
	fm.Status = Finished
	err := db.Save(&fm).Error
	if err != nil {
		log.Error("Failed to unpublish file")
	}
}

// Delete a file from the database
func Delete(db *gorm.DB, fileName string) error {
	log.Infof("Deleting file(name=%s)", fileName)
	var fm FileMetaData
	if db.Where("name = ?", fileName).First(&fm).RecordNotFound() {
		log.Debug("Meta file doesn't exist, file is counted as deleted")
		return nil
	}
	var fragments []Fragment
	db.Where("hash_id = ?", fm.Hash).Find(&fragments)
	tx := db.Begin()
	for _, f := range fragments {
		log.Debugf("Deleted Fragment hash=%s id=%d", f.HashID, f.FragmentID)
		if err := tx.Delete(&f).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	// Delete file
	if err := tx.Delete(&fm).Error; err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func createMetaFile(fs afero.Fs, filePath string) (FileMetaData, error) {
	f, err := fs.Open(filePath)
	if err != nil {
		log.Errorf("Failed to open file(%s). Reason: %s", filePath, err)
		return FileMetaData{}, err
	}
	// Make sure file will be closed after function ends
	defer f.Close()
	stats, err := f.Stat()
	if err != nil {
		log.Error("Failed to find file stats")
		return FileMetaData{}, err
	}
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return FileMetaData{}, err
	}
	fileHash := hex.EncodeToString(h.Sum(nil))
	fragmentCount := int(math.Ceil(float64(stats.Size()) / float64(FileChunkSize)))
	var availableFragments []Fragment
	for i := 0; i < int(fragmentCount); i++ {
		availableFragments = append(availableFragments, Fragment{FragmentID: i, HashID: fileHash})
	}
	host, err := os.Hostname()
	if err != nil {
		return FileMetaData{}, err
	}
	return FileMetaData{stats.Name(), filePath, host, fileHash, stats.Size(), fragmentCount, availableFragments, Finished}, nil
}

// PrintFiles prints an array of files in a human readable table
func PrintFiles(files []FileMetaData) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Client", "File", "Size", "FragmentCount", "Hash", "Status"})
	for i := 0; i < len(files); i++ {
		f := files[i]
		t.AppendRow([]interface{}{i, f.Publisher, f.Name, f.Size,
			fmt.Sprintf("%d/%d", len(f.AvailableFragments), f.FragmentsCount),
			f.Hash, f.Status})
	}
	t.AppendFooter(table.Row{"", "", "", "", "", "Total", len(files)})
	t.Render()
}
