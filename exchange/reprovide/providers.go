package reprovide

import (
	"context"

	pin "github.com/ipfs/go-ipfs/pin"

	"github.com/ipfs/go-ipfs/thirdparty/streaming-cid-set"
	ipld "gx/ipfs/QmUSyMZ8Vt4vTZr5HdDEgEfpwAXfQRuDdfCFTt7XBzhxpQ/go-ipld-format"
	merkledag "gx/ipfs/QmYxX4VfVcxmfsj8U6T5kVtFvHsSidy9tmPyPTW5fy7H3q/go-merkledag"
	cid "gx/ipfs/Qmdu2AYUV7yMoVBQPxXNfe7FJcdx16kYtsx6jAPKWQYF1y/go-cid"
	blocks "gx/ipfs/QmeFZ47hGe5T8nSUjwd6zf6ikzFWYEzWsb1e4Q2r6n1w9z/go-ipfs-blockstore"
)

// NewBlockstoreProvider returns key provider using bstore.AllKeysChan
func NewBlockstoreProvider(bstore blocks.Blockstore) KeyChanFunc {
	return func(ctx context.Context) (<-chan *cid.Cid, error) {
		return bstore.AllKeysChan(ctx)
	}
}

// NewPinnedProvider returns provider supplying pinned keys
func NewPinnedProvider(pinning pin.Pinner, dag ipld.DAGService, onlyRoots bool) KeyChanFunc {
	return func(ctx context.Context) (<-chan *cid.Cid, error) {
		set, err := pinSet(ctx, pinning, dag, onlyRoots)
		if err != nil {
			return nil, err
		}

		outCh := make(chan *cid.Cid)
		go func() {
			defer close(outCh)
			for c := range set.New {
				select {
				case <-ctx.Done():
					return
				case outCh <- c:
				}
			}

		}()

		return outCh, nil
	}
}

func pinSet(ctx context.Context, pinning pin.Pinner, dag ipld.DAGService, onlyRoots bool) (*streamingset.StreamingSet, error) {
	set := streamingset.NewStreamingSet()

	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		for _, key := range pinning.DirectKeys() {
			set.Visitor(ctx)(key)
		}

		for _, key := range pinning.RecursiveKeys() {
			set.Visitor(ctx)(key)

			if !onlyRoots {
				err := merkledag.EnumerateChildren(ctx, merkledag.GetLinksWithDAG(dag), key, set.Visitor(ctx))
				if err != nil {
					log.Errorf("reprovide indirect pins: %s", err)
					return
				}
			}
		}
	}()

	return set, nil
}
