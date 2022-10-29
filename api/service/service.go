package service

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Service defines an instance of service that handles third-party requests.
type Service struct {
	db *gorm.DB
}

// New creates a new service instance.
func New(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

type pingResp struct {
	Pong string `json:"pong"`
}

func (s *Service) Ping(_ *gin.Context) (*pingResp, error) {
	return &pingResp{Pong: "pong"}, nil
}
