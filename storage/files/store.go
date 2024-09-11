package files

import (
	"time"

	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Replica struct {
	ID        uint64
	TxSeq     uint64    `gorm:"not null; unique"`
	Replica   int       `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null; index"`
}

type Store struct {
	db *gorm.DB
}

func MustNewStore(config mysql.Config) *Store {
	db := config.MustOpenOrCreate(&Replica{})

	return &Store{
		db: db,
	}
}

func (s *Store) Upsert(replicas ...*Replica) error {
	return s.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(replicas).Error
}
