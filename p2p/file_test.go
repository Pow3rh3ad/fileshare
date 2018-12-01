package p2p

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/afero"

	"github.com/stretchr/testify/assert"
)

// Test status is printable a human readable version
func TestStatusString(t *testing.T) {
	s := []Status{Status(0), Status(1), Status(2), Status(3), Status(4)}
	assert.Equal(t, "[New Paused Downloading Finished Seeding]", fmt.Sprintf("%s", s))
}

// Test FileMetaData struct and it's functions
func TestFileMetaData(t *testing.T) {
	fragments := []Fragment{Fragment{FragmentID: 0, HashID: "13c405d80e97aa7b46d3389180b19eb3"}, Fragment{FragmentID: 2, HashID: "13c405d80e97aa7b46d3389180b19eb3"}}
	fm := FileMetaData{"Test", "C:\\file\\path\test.exe", "TestPub", "13c405d80e97aa7b46d3389180b19eb3", 666, 2, fragments, 1}
	assert.True(t, fm.FragmentExists(0))
	assert.False(t, fm.FragmentExists(3))
}

func TestCreateDatabase(t *testing.T) {
	defer os.Remove("test.db")
	db, err := CreateDatabase("test.db", false)
	defer db.Close()
	assert.FileExists(t, "test.db")
	assert.Nil(t, err)
	_, err = CreateDatabase("/asdasdsa/asd/test.db", false)
	assert.NotNil(t, err)

}

func TestList(t *testing.T) {
	defer os.Remove("test.db")
	db, err := CreateDatabase("test.db", false)
	defer db.Close()
	assert.Nil(t, err)
	fs := afero.NewOsFs()
	files := List(fs, db)
	assert.Empty(t, files)
	fragments := []Fragment{Fragment{FragmentID: 0, HashID: "13c405d80e97aa7b46d3389180b19eb3"}, Fragment{FragmentID: 2, HashID: "13c405d80e97aa7b46d3389180b19eb3"}}
	fm := FileMetaData{"Test", "C:\\file\\path\test.exe", "TestPub", "13c405d80e97aa7b46d3389180b19eb3", 666, 2, fragments, 1}
	for _, f := range fragments {
		db.Save(&f)
	}
	// Save file meta data and it's fragments
	db.Save(&fm)
	files = List(fs, db)
	assert.Len(t, files, 1)
	assert.Equal(t, files[0], fm)
	fragments = []Fragment{Fragment{FragmentID: 1, HashID: "13c405d80e97aa7b46d3389180b19eb4"}, Fragment{FragmentID: 3, HashID: "13c405d80e97aa7b46d3389180b19eb4"}}
	fm2 := FileMetaData{"Test2", "C:\\file\\path\test.exe", "TestPub", "13c405d80e97aa7b46d3389180b19eb4", 666, 3, fragments, 1}
	for _, f := range fragments {
		db.Save(&f)
	}
	// Save file meta data and it's fragments
	db.Save(&fm2)
	files = List(fs, db)
	assert.Len(t, files, 2)
	assert.Equal(t, files[0], fm)
	assert.Equal(t, files[1], fm2)
}

func TestPublish(t *testing.T) {
	defer os.Remove("test.db")
	db, err := CreateDatabase("test.db", false)
	defer db.Close()
	assert.Nil(t, err)
	fs := afero.NewOsFs()
	var fm FileMetaData
	assert.True(t, db.Where("file_path = ?", "file.go").First(&fm).RecordNotFound())
	// Lets publish the file we are testing
	err = Publish(fs, db, "file.go")
	assert.Nil(t, err)
	// Now we'll check db that all is fine
	assert.False(t, db.Where("file_path = ?", "file.go").First(&fm).RecordNotFound())
	files := List(fs, db)
	assert.Len(t, files, 1)
	assert.Equal(t, files[0].Status, Status(Seeding))
	assert.Equal(t, files[0].FragmentsCount, 1)
}

func TestUnpublish(t *testing.T) {
	defer os.Remove("test.db")
	db, err := CreateDatabase("test.db", false)
	defer db.Close()
	assert.Nil(t, err)
	Unpublish(db, "somerandompaththatdoesntexist.go")
	// Now lets publish a file
	fs := afero.NewOsFs()
	var fm FileMetaData
	err = Publish(fs, db, "file.go")
	assert.Nil(t, err)
	assert.False(t, db.Where("file_path = ?", "file.go").First(&fm).RecordNotFound())
	assert.Equal(t, fm.Status, Status(Seeding))
	Unpublish(db, "file.go")
	assert.False(t, db.Where("file_path = ?", "file.go").First(&fm).RecordNotFound())
	assert.Equal(t, fm.Status, Status(Finished))
}

func TestCreateMetaFile(t *testing.T) {
	fs := afero.NewOsFs()
	fm, err := createMetaFile(fs, "file.go")
	assert.Nil(t, err)
	// File should be 1 fragment (size < 1mb)
	assert.Equal(t, 1, fm.FragmentsCount)
	assert.True(t, fm.Size < FileChunkSize)
	assert.Equal(t, "file.go", fm.Name)
	assert.Equal(t, "file.go", fm.FilePath)
	assert.Len(t, fm.AvailableFragments, 1)
	// file shouldn't exist
	fm, err = createMetaFile(fs, "fildde.go")
	assert.NotNil(t, err)
}
