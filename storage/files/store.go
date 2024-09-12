package files

import (
	"database/sql"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type File struct {
	ID          uint64
	TxSeq       uint64    `gorm:"not null; unique"`
	NumErrors   uint64    `gorm:"not null"`
	NumNotSync  uint64    `gorm:"not null"`
	NumSynced   uint64    `gorm:"not null"`
	NumUploaded uint64    `gorm:"not null"`
	NumReplica  int       `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null; index"`
}

type Store struct {
	db *gorm.DB
}

func MustNewStore(config mysql.Config) *Store {
	db := config.MustOpenOrCreate(&File{})

	return &Store{
		db: db,
	}
}

func (s *Store) Upsert(files ...*File) error {
	return s.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(files).Error
}

func (s *Store) MaxTxSeq() (sql.NullInt64, error) {
	var maxTxSeq sql.NullInt64
	if err := s.db.Table("files").Select("max(tx_seq)").Scan(&maxTxSeq).Error; err != nil {
		return sql.NullInt64{}, err
	}
	return maxTxSeq, nil
}
