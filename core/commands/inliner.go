package commands

import (
	mh "gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	cid "gx/ipfs/Qmdu2AYUV7yMoVBQPxXNfe7FJcdx16kYtsx6jAPKWQYF1y/go-cid"
)

// Inliner is a cid.Builder that will use the id multihash when the
// size of the content is no more than limit
type Inliner struct {
	base  cid.Builder
	limit int
}

// GetCodec implements the cid.Builder interface
func (p Inliner) GetCodec() uint64 {
	return p.base.GetCodec()
}

// WithCodec implements the cid.Builder interface
func (p Inliner) WithCodec(c uint64) cid.Builder {
	return Inliner{p.base.WithCodec(c), p.limit}
}

// Sum implements the cid.Builder interface
func (p Inliner) Sum(data []byte) (*cid.Cid, error) {
	if len(data) > p.limit {
		return p.base.Sum(data)
	}
	return cid.V1Builder{Codec: p.base.GetCodec(), MhType: mh.ID}.Sum(data)
}
