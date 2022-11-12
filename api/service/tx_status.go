package service

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/photon-storage/go-common/log"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photo-storage/dropbox/database/orm"
)

type txStatusTask struct {
	ctx     context.Context
	db      *gorm.DB
	nodeCli pbc.NodeClient
}

func newTxStatusTask(
	ctx context.Context,
	db *gorm.DB,
	nodeCli pbc.NodeClient,
) *txStatusTask {
	return &txStatusTask{
		ctx:     ctx,
		db:      db,
		nodeCli: nodeCli,
	}
}

func (t *txStatusTask) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := t.updateObjectTxStatus(); err != nil {
				log.Error("update tx status failed", "error", err)
			}

		case <-t.ctx.Done():
			return
		}
	}
}

func (t *txStatusTask) updateObjectTxStatus() error {
	os := make([]*orm.Object, 0)
	if err := t.db.Model(&orm.Object{}).
		Where("status in (?,?)", orm.ObjectPending, orm.ObjectCommitted).
		Find(&os).
		Error; err != nil {
		return err
	}

	for _, o := range os {
		hash, err := hex.DecodeString(o.CommitTxHash)
		if err != nil {
			return err
		}

		tx, err := t.nodeCli.GetTransaction(
			t.ctx,
			&pbc.GetTransactionRequest{Hash: hash},
		)

		if err != nil {
			if status.Convert(err).Code() == codes.NotFound {
				if o.CreatedAt.Add(time.Hour).Before(time.Now()) {
					if err := updateTxStatus(t.db, o.ID, orm.ObjectFailed); err != nil {
						return err
					}
				}

				continue
			}

			return errors.Wrapf(err, "tx hash: %s", o.CommitTxHash)
		}

		switch o.Status {
		case orm.ObjectPending:
			status := orm.ObjectCommitted
			if tx.Finalized {
				status = orm.ObjectFinalized
			}

			if err := updateTxStatus(t.db, o.ID, status); err != nil {
				return err
			}

		case orm.ObjectCommitted:
			if tx.Finalized {
				if err := updateTxStatus(t.db, o.ID, orm.ObjectFinalized); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func updateTxStatus(db *gorm.DB, id uint64, status orm.ObjectStatus) error {
	return db.Model(&orm.Object{}).
		Where("id = ?", id).
		Update("status", status).
		Error
}
