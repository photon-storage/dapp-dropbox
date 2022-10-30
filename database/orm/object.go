package orm

import "time"

// ObjectStatus represents the status of
// different life cycles of Object.
type ObjectStatus uint8

const (
	ObjectPending = iota + 1
	ObjectCommitted
	ObjectFinalized
	ObjectFailed
)

// Object is a gorm table definition represents the objects.
type Object struct {
	ID           uint64 `gorm:"primary_key"`
	Name         string
	CommitTxHash string
	Hash         string
	Size         uint64
	EncodedHash  string
	EncodedSize  uint64
	Cid          string
	Status       ObjectStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
