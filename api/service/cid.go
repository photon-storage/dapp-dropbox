package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/photon-storage/go-common/log"
	"github.com/photon-storage/go-photon/crypto/sha256"
	pbd "github.com/photon-storage/photon-proto/depot"

	"github.com/photo-storage/dropbox/database/orm"
)

type cidTask struct {
	ctx      context.Context
	db       *gorm.DB
	depotCli pbd.DepotClient
}

func newCIDTask(
	ctx context.Context,
	db *gorm.DB,
	depotCli pbd.DepotClient,
) *cidTask {
	return &cidTask{
		ctx:      ctx,
		db:       db,
		depotCli: depotCli,
	}
}

func (c *cidTask) run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.fetchCID(); err != nil {
				log.Error("fetch object cid failed", "error", err)
			}

		case <-c.ctx.Done():
			return
		}
	}
}

func (c *cidTask) fetchCID() error {
	os := make([]*orm.Object, 0)
	if err := c.db.Model(&orm.Object{}).
		Where("cid = ? and status in (?,?)",
			"",
			orm.ObjectCommitted,
			orm.ObjectFinalized,
		).
		Limit(10).
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

		objResp, err := c.depotCli.ObjectStatus(c.ctx, &pbd.ObjectStatusRequest{
			Hash:         ohash.Bytes(),
			CommitTxHash: commitTxHash.Bytes(),
		})
		if err != nil {
			return err
		}

		if len(objResp.Cid) == 0 {
			continue
		}

		if err := c.db.Model(&orm.Object{}).
			Where("id = ?", o.ID).
			Update("cid", string(objResp.Cid)).
			Error; err != nil {
			return err
		}
	}

	return nil
}
