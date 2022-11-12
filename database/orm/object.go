package orm

import "time"

// ObjectStatus represents the status of
// different life cycles of Object.
type ObjectStatus uint8

const (
	ObjectPending ObjectStatus = iota + 1
	ObjectCommitted
	ObjectFinalized
	ObjectFailed
)

var objectMap = map[ObjectStatus]string{
	ObjectPending:   "pending",
	ObjectCommitted: "committed",
	ObjectFinalized: "finalized",
	ObjectFailed:    "failed",
}

// Object is a gorm table definition represents the objects.
type Object struct {
	ID             uint64 `gorm:"primary_key"`
	Name           string
	OwnerPublicKey string
	DepotPublicKey string
	CommitTxHash   string
	Hash           string
	Size           uint64
	EncodedHash    string
	EncodedSize    uint64
	Cid            string
	Status         ObjectStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (o ObjectStatus) String() string {
	if v, ok := objectMap[o]; ok {
		return v
	}

	return "invalid"
}
