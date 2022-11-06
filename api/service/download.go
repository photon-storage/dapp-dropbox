package service

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/photon-storage/go-photon/crypto/codec"
	"github.com/photon-storage/go-photon/crypto/sha256"
	"github.com/photon-storage/go-photon/depot"
	pbd "github.com/photon-storage/photon-proto/depot"

	"github.com/photo-storage/dropbox/database/orm"
)

const DownloadLabel = "download"

// Download handles the /download request.
func (s *Service) Download(c *gin.Context) error {
	o := &orm.Object{}
	if err := s.db.Model(&orm.Object{}).
		Where("commit_tx_hash = ?", c.Query("hash")).
		First(o).Error; err != nil {
		return err
	}

	hash, err := hex.DecodeString(o.Hash)
	if err != nil {
		return err
	}

	commitTxHash, err := sha256.HashFromHex(o.CommitTxHash)
	if err != nil {
		return err
	}

	objResp, err := s.depotCli.ObjectStatus(c, &pbd.ObjectStatusRequest{
		Hash:         hash,
		CommitTxHash: commitTxHash.Bytes(),
	})
	if err != nil {
		return err
	}

	if objResp.Status != pbd.ObjectStatus_READABLE {
		return errObjectNotReadable
	}

	decoder, err := codec.NewMultikey(objResp.Decoder)
	if err != nil {
		return err
	}

	ohash, err := sha256.HashFromHex(o.Hash)
	if err != nil {
		return err
	}

	ehash, err := sha256.HashFromHex(o.EncodedHash)
	if err != nil {
		return err
	}

	df, err := depot.NewDownloadFile(
		commitTxHash,
		ohash,
		ehash,
		objResp.Size,
		objResp.EncodedSize,
		objResp.NumBlocks,
		objResp.BlocksPerChunk,
		objResp.NumChunks,
		decoder,
	)

	for i := uint32(0); i < df.NumChunks(); i++ {
		resp, err := s.depotCli.DownloadChunk(s.ctx, &pbd.DownloadChunkRequest{
			Hash:         df.OriginalHash().Bytes(),
			CommitTxHash: objResp.CommitTxHash,
			Index:        i,
		})
		if err != nil {
			return err
		}

		if err := df.SetChunk(i, resp.Chunk); err != nil {
			return err
		}
	}

	if err := df.Write(c.Writer); err != nil {
		return err
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", o.Name))
	c.Header("Content-Length", strconv.Itoa(int(o.Size)))
	c.Set(DownloadLabel, nil)
	return nil
}
