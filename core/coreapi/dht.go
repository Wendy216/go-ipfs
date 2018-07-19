package coreapi

import (
	"context"
	"errors"
	"fmt"

	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	caopts "github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	blockservice "gx/ipfs/QmQrLuAVriwcPQpqn15GU7gjZGKKa45Hmdj9JCG3Cc45CC/go-blockservice"
	offline "gx/ipfs/QmTMhpt5sH5yup2RU5qeFDMZcDf4RaHqba4ztEp7yrzese/go-ipfs-exchange-offline"
	pstore "gx/ipfs/QmYLXCWN2myozZpx8Wx4UjrRuQuhY3YtWoMi6SHaXii6aM/go-libp2p-peerstore"
	dag "gx/ipfs/QmYxX4VfVcxmfsj8U6T5kVtFvHsSidy9tmPyPTW5fy7H3q/go-merkledag"
	peer "gx/ipfs/QmcZSzKEM5yDfpZbeEEZaVmaZ1zXm6JWTbrQZSB8hCVPzk/go-libp2p-peer"
	cid "gx/ipfs/Qmdu2AYUV7yMoVBQPxXNfe7FJcdx16kYtsx6jAPKWQYF1y/go-cid"
	blockstore "gx/ipfs/QmeFZ47hGe5T8nSUjwd6zf6ikzFWYEzWsb1e4Q2r6n1w9z/go-ipfs-blockstore"
	routing "gx/ipfs/QmfGeECX7CxJAFXwGKx2cn7csnxAtwJw8e3XtoNVf6bqcv/go-libp2p-routing"
)

type DhtAPI struct {
	*CoreAPI
	*caopts.DhtOptions
}

func (api *DhtAPI) FindPeer(ctx context.Context, p peer.ID) (pstore.PeerInfo, error) {
	pi, err := api.node.Routing.FindPeer(ctx, peer.ID(p))
	if err != nil {
		return pstore.PeerInfo{}, err
	}

	return pi, nil
}

func (api *DhtAPI) FindProviders(ctx context.Context, p coreiface.Path, opts ...caopts.DhtFindProvidersOption) (<-chan pstore.PeerInfo, error) {
	settings, err := caopts.DhtFindProvidersOptions(opts...)
	if err != nil {
		return nil, err
	}

	rp, err := api.ResolvePath(ctx, p)
	if err != nil {
		return nil, err
	}

	numProviders := settings.NumProviders
	if numProviders < 1 {
		return nil, fmt.Errorf("number of providers must be greater than 0")
	}

	pchan := api.node.Routing.FindProvidersAsync(ctx, rp.Cid(), numProviders)
	return pchan, nil
}

func (api *DhtAPI) Provide(ctx context.Context, path coreiface.Path, opts ...caopts.DhtProvideOption) error {
	settings, err := caopts.DhtProvideOptions(opts...)
	if err != nil {
		return err
	}

	if api.node.Routing == nil {
		return errors.New("cannot provide in offline mode")
	}

	rp, err := api.ResolvePath(ctx, path)
	if err != nil {
		return err
	}

	c := rp.Cid()

	has, err := api.node.Blockstore.Has(c)
	if err != nil {
		return err
	}

	if !has {
		return fmt.Errorf("block %s not found locally, cannot provide", c)
	}

	if settings.Recursive {
		err = provideKeysRec(ctx, api.node.Routing, api.node.Blockstore, []*cid.Cid{c})
	} else {
		err = provideKeys(ctx, api.node.Routing, []*cid.Cid{c})
	}
	if err != nil {
		return err
	}

	return nil
}

func provideKeys(ctx context.Context, r routing.IpfsRouting, cids []*cid.Cid) error {
	for _, c := range cids {
		err := r.Provide(ctx, c, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func provideKeysRec(ctx context.Context, r routing.IpfsRouting, bs blockstore.Blockstore, cids []*cid.Cid) error {
	provided := cid.NewSet()
	for _, c := range cids {
		kset := cid.NewSet()

		dserv := dag.NewDAGService(blockservice.New(bs, offline.Exchange(bs)))

		err := dag.EnumerateChildrenAsync(ctx, dag.GetLinksDirect(dserv), c, kset.Visit)
		if err != nil {
			return err
		}

		for _, k := range kset.Keys() {
			if provided.Has(k) {
				continue
			}

			err = r.Provide(ctx, k, true)
			if err != nil {
				return err
			}
			provided.Add(k)
		}
	}

	return nil
}
