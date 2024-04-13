package providers

import (
	"context"
	"time"

	flags "github.com/ipfs/go-ipfs/flags"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	host "gx/ipfs/QmRS46AyqtpJBsf1zmQdeizSDEzo1qkWR7rdEuPFAv8237/go-libp2p-host"
	process "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	procctx "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess/context"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

var (
	provideKeysBufferSize = 2048
	// HasBlockBufferSize is the maximum numbers of CIDs that will get buffered
	// for providing
	HasBlockBufferSize = 256

	provideWorkerMax = 512
	provideTimeout   = time.Second * 15

	// maxProvidersPerRequest specifies the maximum number of providers desired
	// from the network. This value is specified because the network streams
	// results.
	// TODO: if a 'non-nice' strategy is implemented, consider increasing this value
	maxProvidersPerRequest = 3
	providerRequestTimeout = time.Second * 10

	sizeBatchRequestChan = 32
)

var log = logging.Logger("providers")

type blockRequest struct {
	Cid *cid.Cid
	Ctx context.Context
}

// Interface is an definition of providers interface to libp2p routing system
type Interface interface {
	Provide(*cid.Cid) error
	FindProviders(ctx context.Context, c *cid.Cid) error
	FindProvidersAsync(ctx context.Context, k *cid.Cid, max int) <-chan peer.ID

	Stat() (*Stat, error)
}

type providers struct {
	routing routing.ContentRouting
	process process.Process
	host    host.Host

	// newBlocks is a channel for newly added blocks to be provided to the
	// network.  blocks pushed down this channel get buffered and fed to the
	// provideKeys channel later on to avoid too much network activity
	newBlocks chan *cid.Cid
	// provideKeys directly feeds provide workers
	provideKeys chan *cid.Cid

	// findKeys sends keys to a worker to find and connect to providers for them
	findKeys chan *blockRequest
}

func init() {
	if flags.LowMemMode {
		HasBlockBufferSize = 64
		provideKeysBufferSize = 512
		provideWorkerMax = 16
	}
}

// NewProviders returns providers interface implementation based on
// libp2p routing
func NewProviders(parent context.Context, routing routing.ContentRouting, host host.Host) Interface {
	ctx, cancelFunc := context.WithCancel(parent)

	px := process.WithTeardown(func() error {
		return nil
	})

	p := &providers{
		routing: routing,
		process: px,
		host:    host,

		newBlocks:   make(chan *cid.Cid, HasBlockBufferSize),
		provideKeys: make(chan *cid.Cid, provideKeysBufferSize),

		findKeys: make(chan *blockRequest, sizeBatchRequestChan),
	}

	p.startWorkers(ctx, px)
	// bind the context and process.
	// do it over here to avoid closing before all setup is done.
	go func() {
		<-px.Closing() // process closes first
		cancelFunc()
	}()
	procctx.CloseAfterContext(px, ctx) // parent cancelled first

	return p
}

func (p *providers) Provide(b *cid.Cid) error {
	select {
	case p.newBlocks <- b:
	// send block off to be provided to the network
	case <-p.process.Closing():
		return p.process.Close()
	}
	return nil
}

func (p *providers) FindProviders(ctx context.Context, c *cid.Cid) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.findKeys <- &blockRequest{Ctx: ctx, Cid: c}:
		return nil
	}
}

// FindProvidersAsync returns a channel of providers for the given key
func (p *providers) FindProvidersAsync(ctx context.Context, k *cid.Cid, max int) <-chan peer.ID {

	// Since routing queries are expensive, give bitswap the peers to which we
	// have open connections. Note that this may cause issues if bitswap starts
	// precisely tracking which peers provide certain keys. This optimization
	// would be misleading. In the long run, this may not be the most
	// appropriate place for this optimization, but it won't cause any harm in
	// the short term.
	connectedPeers := p.host.Network().Peers()
	out := make(chan peer.ID, len(connectedPeers)) // just enough buffer for these connectedPeers
	for _, id := range connectedPeers {
		if id == p.host.ID() {
			continue // ignore self as provider
		}
		out <- id
	}

	go func() {
		defer close(out)
		providers := p.routing.FindProvidersAsync(ctx, k, max)
		for info := range providers {
			if info.ID == p.host.ID() {
				continue // ignore self as provider
			}
			p.host.Peerstore().AddAddrs(info.ID, info.Addrs, pstore.TempAddrTTL)
			select {
			case <-ctx.Done():
				return
			case out <- info.ID:
			}
		}
	}()
	return out
}
