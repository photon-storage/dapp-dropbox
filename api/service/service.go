package service

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/config/config"
	"github.com/photon-storage/go-photon/sak/io/rpc"
	pbc "github.com/photon-storage/photon-proto/consensus"
	pbd "github.com/photon-storage/photon-proto/depot"
)

// Service defines an instance of service that handles third-party requests.
type Service struct {
	ctx              context.Context
	db               *gorm.DB
	depotPk          []byte
	depotDiscoveryID []byte
	nodeCli          pbc.NodeClient
	depotCli         pbd.DepotClient
}

// New creates a new service instance.
func New(
	ctx context.Context,
	db *gorm.DB,
	nodeEndpoint string,
	configType config.ConfigType,
	depotBootstrap []string,
) (*Service, error) {
	nc, err := rpcDialConfig(nodeEndpoint).Dial(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "dial node failed")
	}

	// TODO(doris): Async find multi depot client.
	depotEndpoint, err := findDepot(ctx, configType, depotBootstrap)
	if err != nil {
		return nil, err
	}

	dc, err := rpcDialConfig(depotEndpoint).Dial(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "dial depot node failed")
	}

	depotCli := pbd.NewDepotClient(dc)
	depotState, err := depotCli.State(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	nodeCli := pbc.NewNodeClient(nc)
	go newTxStatusTask(ctx, db, nodeCli).run()
	go newCIDTask(ctx, db, depotCli).run()
	return &Service{
		ctx:              ctx,
		db:               db,
		depotPk:          depotState.GetPublicKey(),
		depotDiscoveryID: depotState.GetDiscoveryId(),
		nodeCli:          nodeCli,
		depotCli:         depotCli,
	}, nil
}

func rpcDialConfig(endpoint string) rpc.DialConfig {
	return rpc.DialConfig{
		Endpoint:    endpoint,
		NumRetries:  5,
		RetryDelay:  10 * time.Second,
		MaxRecvSize: 1 << 22,
	}
}

type pingResp struct {
	Pong string `json:"pong"`
}

func (s *Service) Ping(_ *gin.Context) (*pingResp, error) {
	return &pingResp{Pong: "pong"}, nil
}
