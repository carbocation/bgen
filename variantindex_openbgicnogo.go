// +build !cgo

package bgen

// If cgo is not enabled, we will use the modernc.org/sqlite non-cgo sqlite
// driver. It is slower than the sqlite3 cgo driver.

import (
	"strings"

	"github.com/jmoiron/sqlx"

	_ "modernc.org/sqlite"
)

const whichSQLiteDriver = "sqlite"

func OpenBGI(path string) (*BGIIndex, error) {
	bgi := &BGIIndex{
		Metadata: &BGIMetadata{},
	}

	// URI filenames have to begin with 'file:'; see
	// https://www.sqlite.org/c3ref/open.html . It seems that sqlite3 permitted
	// URI filenames without the file: prefix, but that is not standard.
	if !strings.HasPrefix(path, "file:") {
		path = "file:" + path
	}

	db, err := sqlx.Connect("sqlite", path)
	if err != nil {
		return nil, err
	}
	bgi.DB = db

	// Not all index files have metadata; ignore any error
	_ = bgi.DB.Get(bgi.Metadata, "SELECT * FROM Metadata LIMIT 1")

	return bgi, nil
}
