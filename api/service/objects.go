package service

import (
	"github.com/docker/go-units"
	"github.com/gin-gonic/gin"

	"github.com/photo-storage/dropbox/api/pagination"
	"github.com/photo-storage/dropbox/database/orm"
)

type object struct {
	FileName     string `json:"file_name"`
	CommitTxHash string `json:"commit_tx_hash"`
	CID          string `json:"cid"`
	Status       string `json:"status"`
	Timestamp    uint64 `json:"timestamp"`
	Size         string `json:"size"`
}

// Objects handles the /objects request.
func (s *Service) Objects(
	_ *gin.Context,
	page *pagination.Query,
) (*pagination.Result, error) {
	objects := make([]*orm.Object, 0)
	if err := s.db.Model(&orm.Object{}).
		Offset(page.Start).
		Limit(page.Limit).
		Order("id desc").
		Find(&objects).
		Error; err != nil {
		return nil, err
	}

	os := make([]*object, len(objects))
	for i, o := range objects {
		os[i] = &object{
			FileName:     o.Name,
			CommitTxHash: o.CommitTxHash,
			CID:          o.Cid,
			Status:       o.Status.String(),
			Timestamp:    uint64(o.CreatedAt.Unix()),
			Size:         units.HumanSize(float64(o.Size)),
		}
	}

	count := int64(0)
	if err := s.db.Model(&orm.Object{}).Count(&count).Error; err != nil {
		return nil, err
	}

	return &pagination.Result{
		Data:  os,
		Total: count,
	}, nil
}
