package service

import (
	"encoding/hex"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/photon-storage/go-common/log"
	"github.com/photon-storage/go-photon/crypto/sha256"
	pbc "github.com/photon-storage/photon-proto/consensus"
	pbd "github.com/photon-storage/photon-proto/depot"

	"github.com/photo-storage/dropbox/database/orm"
)

type job func() error

type task struct {
	name string
	job  job
}

func newTask(name string, job job) *task {
	return &task{
		name: name,
		job:  job,
	}
}

func (s *Service) scheduler(interval int, task *task) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := task.job(); err != nil {
				log.Error("run job failed",
					"name", task.name,
					"error", err,
				)
			}

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) updateObjectTxStatus() error {
	os := make([]*orm.Object, 0)
	if err := s.db.Model(&orm.Object{}).
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

		tx, err := s.nodeCli.GetTransaction(
			s.ctx,
			&pbc.GetTransactionRequest{Hash: hash},
		)

		if err != nil {
			if status.Convert(err).Code() == codes.NotFound {
				if o.CreatedAt.Add(time.Hour).Before(time.Now()) {
					if err := updateTxStatus(s.db, o.ID, orm.ObjectFailed); err != nil {
						return err
					}
				}

				continue
			}

			return errors.Wrapf(err, "tx hash: %s", o.CommitTxHash)
		}

		if o.Status == orm.ObjectPending {
			status := orm.ObjectCommitted
			if tx.Finalized {
				status = orm.ObjectFinalized
			}

			if err := updateTxStatus(s.db, o.ID, status); err != nil {
				return err
			}
		}

		if o.Status == orm.ObjectCommitted && tx.Finalized {
			if err := updateTxStatus(s.db, o.ID, orm.ObjectFinalized); err != nil {
				return err
			}
		}
	}

	return nil
}

func updateTxStatus(db *gorm.DB, id uint64, status orm.ObjectStatus) error {
	return db.Model(&orm.Object{}).Where("id = ?", id).
		Update("status", status).
		Error
}

func (s *Service) fetchCID() error {
	os := make([]*orm.Object, 0)
	if err := s.db.Model(&orm.Object{}).
		Where("cid = ?", "").
		Find(&os).
		Error; err != nil {
		return err
	}

	for _, o := range os {
		ohash, err := sha256.HashFromHex(o.Hash)
		if err != nil {
			return err
		}

		commitTxHash, err := sha256.HashFromHex(o.CommitTxHash)
		if err != nil {
			return err
		}

		objResp, err := s.depotCli.ObjectStatus(s.ctx, &pbd.ObjectStatusRequest{
			Hash:         ohash.Bytes(),
			CommitTxHash: commitTxHash.Bytes(),
		})
		if err != nil {
			return err
		}

		if objResp.Status != pbd.ObjectStatus_READABLE || len(objResp.Cid) == 0 {
			continue
		}

		if err := s.db.Model(&orm.Object{}).
			Where("id = ?", o.ID).
			Update("cid", string(objResp.Cid)).
			Error; err != nil {
			return err
		}
	}

	return nil
}
