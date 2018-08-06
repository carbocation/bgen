package bgen

import (
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
	FileSize           uint   `db:"file_size"`
	LastWriteTime      Time   `db:"last_write_time"`
	FirstThousandBytes []byte `db:"first_1000_bytes"`
	IndexCreationTime  Time   `db:"index_creation_time"`
}
