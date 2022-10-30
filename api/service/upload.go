package service

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"

	fieldparams "github.com/photon-storage/go-photon/config/fieldparams"
	"github.com/photon-storage/go-photon/crypto/sha256"
	"github.com/photon-storage/go-photon/depot"
	"github.com/photon-storage/go-photon/proto/consensus/domain"
	pbc "github.com/photon-storage/photon-proto/consensus"
	pbd "github.com/photon-storage/photon-proto/depot"

	"github.com/photo-storage/dropbox/database/orm"
)

const deadlineMod = uint64(300)

var (
	ErrSectorsPerBlockMismatch = errors.New("SectorsPerBlock setting is different from server")
	ErrBlocksPerChunkMismatch  = errors.New("BlocksPerChunk setting is different from server")
	ErrChunkCountMismatch      = errors.New("unexpected received chunks count")
)

// Upload handles the /upload request.
func (s *Service) Upload(c *gin.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	uf, err := depot.NewUploadFile(
		src,
		nil, /* no block signature */
		nil, /* no encoding */
	)
	if err != nil {
		return err
	}

	tx, hash, err := s.buildCommitTx(uf)
	if err != nil {
		return err
	}
	uf.SetTxHash(hash)

	initResp, err := s.depotCli.UploadInit(
		s.ctx,
		&pbd.UploadInitRequest{
			SignedTx:           tx,
			InitiatorSignature: tx.Signature,
			FromSignature:      tx.Signature,
		},
	)
	if err != nil {
		return err
	}

	if depot.SectorsPerBlock != initResp.SectorsPerBlock {
		return ErrSectorsPerBlockMismatch
	}

	if depot.BlocksPerChunk != initResp.BlocksPerChunk {
		return ErrBlocksPerChunkMismatch
	}

	received := uint32(0)
	for i := uint32(0); i < uf.NumChunks(); i++ {
		resp, err := s.depotCli.UploadChunk(s.ctx, &pbd.UploadChunkRequest{
			Chunk: uf.GetChunk(i),
		})
		if err != nil {
			return err
		}

		if resp.ReceivedChunks != received+1 {
			return ErrChunkCountMismatch
		}
		received++
	}

	return insertObject(s.db, file.Filename, hash.Hex(), uf)
}

func (s *Service) buildCommitTx(uf *depot.UploadFile) (*pbc.SignedTransaction, sha256.Hash, error) {
	sk := nextSk()
	pk := sk.PublicKey().Bytes()
	acct, err := s.nodeCli.GetAccount(
		s.ctx,
		&pbc.AccountRequest{Address: pk},
	)
	if err != nil {
		return nil, sha256.Zero, err
	}

	head, err := s.nodeCli.GetChainHead(s.ctx, &emptypb.Empty{})
	if err != nil {
		return nil, sha256.Zero, err
	}

	deadline := (uint64(head.HeadSlot)/deadlineMod + 2) * deadlineMod
	tx := &pbc.Transaction{
		Type:     uint32(pbc.TxType_OBJECT_COMMIT),
		From:     pk,
		ChainId:  uint32(1),
		Nonce:    acct.Nonce,
		GasPrice: uint64(1),
		GasLimit: fieldparams.ObjectCommitGas,
		TxDataObjectCommit: &pbc.TxDataObjectCommit{
			Owner:            pk,
			Depot:            s.depotPk,
			DepotDiscoveryId: s.depotDiscoveryID,
			Hash:             uf.OriginalHash().Bytes(),
			Size:             uf.OriginalSize(),
			EncodedHash:      uf.EncodedHash().Bytes(),
			EncodedSize:      uf.EncodedSize(),
			NumBlocks:        uf.NumBlocks(),
			Duration:         pbc.Slot(10000),
			Fee:              uint64(1),
			Pledge:           uint64(1),
			Deadline:         pbc.Slot(deadline),
		},
	}
	h, err := tx.HashTreeRoot()
	if err != nil {
		return nil, sha256.Zero, err
	}

	sig, err := domain.Tx.Sign(tx, sk)
	if err != nil {
		return nil, sha256.Zero, err
	}

	return &pbc.SignedTransaction{
		Tx:        tx,
		Signature: sig.Bytes(),
	}, h, nil
}

func insertObject(
	db *gorm.DB,
	name string,
	txHash string,
	uf *depot.UploadFile,
) error {
	return db.Model(&orm.Object{}).Create(&orm.Object{
		Name:         name,
		CommitTxHash: txHash,
		Hash:         uf.OriginalHash().Hex(),
		Size:         uf.OriginalSize(),
		EncodedHash:  uf.EncodedHash().Hex(),
		EncodedSize:  uf.EncodedSize(),
		Status:       orm.ObjectPending,
	}).Error
}
