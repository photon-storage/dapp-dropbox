package service

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/pkg/errors"

	"github.com/photon-storage/go-common/log"
	"github.com/photon-storage/go-photon/chain/p2p/peers/scorers"
	"github.com/photon-storage/go-photon/p2p"
)

var ErrTimeout = errors.New("find node time out")

func findDepot(ctx context.Context, bootstrap []string) (string, error) {
	pf, err := newPeerFinder(ctx, bootstrap)
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(time.Minute)
	timedOut := false
	var endpoint p2p.RPCEndpoint
	pf.Run(func(n *enode.Node) bool {
		if time.Now().After(deadline) {
			timedOut = true
			return true
		}

		var role p2p.Role
		if err := n.Load(&role); err != nil {
			log.Error("Load role", "err", err)
			return false
		}

		if role != p2p.RoleDepot {
			time.Sleep(10 * time.Millisecond)
			return false
		}

		if err := n.Load(&endpoint); err != nil {
			log.Error("load endpoint", "err", err)
		}

		return true
	})

	if timedOut {
		return "", ErrTimeout
	}

	return string(endpoint), nil
}

func newPeerFinder(
	ctx context.Context,
	bootstrap []string,
) (*p2p.PeerFinder, error) {
	scorer := scorers.NewService(ctx, scorers.Config{})
	h, err := p2p.NewHost(
		ctx,
		p2p.HostConfig{
			NetworkID: p2p.NewID(),
			Role:      p2p.RoleObserver,
			TCPPort:   10003,
			MaxPeers:  100,
		},
		scorer,
	)
	if err != nil {
		return nil, err
	}

	return p2p.NewPeerFinder(ctx, h, p2p.PeerFinderConfig{
		UDPPort:        10004,
		BootstrapNodes: bootstrap,
	})
}
