package bgen

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type BGIIndex struct {
	DB       *sqlx.DB
	Metadata *BGIMetadata
}

func (b *BGIIndex) Close() error {
	return b.DB.Close()
}

func OpenBGI(path string) (*BGIIndex, error) {
	bgi := &BGIIndex{
		Metadata: &BGIMetadata{},
	}

	db, err := sqlx.Connect("sqlite3", path)
	if err != nil {
		return nil, err
	}
	bgi.DB = db

	// Not all index files have metadata; ignore any error
	_ = bgi.DB.Get(bgi.Metadata, "SELECT * FROM Metadata LIMIT 1")

	return bgi, nil
}

// VariantIndex conforms to the data found in the rows of the SQLite table
// "Variant" from BGEN Index (.bgi) files, and can be easily parsed with sqlx.
type VariantIndex struct {
	Chromosome        string
	Position          uint32
	RSID              string `db:"rsid"`
	NAlleles          uint16 `db:"number_of_alleles"`
	Allele1           Allele
	Allele2           Allele
	FileStartPosition uint `db:"file_start_position"`
	SizeInBytes       uint `db:"size_in_bytes"`
}

// BGIMetadata conforms to the data found in the rows of the SQLite table
// "Metadata" from more recent versions of BGEN.
type BGIMetadata struct {
	Filename           string
	FileSize           uint     `db:"file_size"`
	LastWriteTime      BGENTime `db:"last_write_time"`
	FirstThousandBytes []byte   `db:"first_1000_bytes"`
	IndexCreationTime  BGENTime `db:"index_creation_time"`
}

// BGENTime exists to facilitate time parsing from the Metadata, because BGEN
// uses both unixtime and text strings to represent time. Derived from
// https://github.com/mattn/go-sqlite3/issues/190#issuecomment-343341834f
type BGENTime time.Time

func (t *BGENTime) Scan(v interface{}) error {
	switch which := v.(type) {
	case int64:
		vt := time.Unix(which, 0)
		*t = BGENTime(vt)
		return nil
	case int:
		vt := time.Unix(int64(which), 0)
		*t = BGENTime(vt)
		return nil
	case []byte:
		// Should be more strictly to check this type.
		vt, err := time.Parse("2006-01-02 15:04:05", string(which))
		if err != nil {
			return err
		}
		*t = BGENTime(vt)
		return nil
	}

	return fmt.Errorf("No appropriate type could be found to decode %v", v)
}
