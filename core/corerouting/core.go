package corerouting

import (
	"errors"

	context "context"
	core "github.com/ipfs/go-ipfs/core"
	repo "github.com/ipfs/go-ipfs/repo"
	supernode "github.com/ipfs/go-ipfs/routing/supernode"
	gcproxy "github.com/ipfs/go-ipfs/routing/supernode/proxy"
	pstore "gx/ipfs/QmNUVzEjq3XWJ89hegahPvyfJbTXgTaom48pLb7YBD9gHQ/go-libp2p-peerstore"
	ds "gx/ipfs/QmRWDav6mzWseLWeYfVd5fvUKiVe9xNH29YfMF438fG364/go-datastore"
	routing "gx/ipfs/QmafuecpeZp3k3sHJ5mUARHd4795revuadECQMkmHB8LfW/go-libp2p-routing"
	"gx/ipfs/QmcyNeWPsoFGxThGpV8JnJdfUNankKhWCTrbrcFRQda4xR/go-libp2p-host"
)

// NB: DHT option is included in the core to avoid 1) because it's a sane
// default and 2) to avoid a circular dependency (it needs to be referenced in
// the core if it's going to be the default)

var (
	errHostMissing      = errors.New("supernode routing client requires a Host component")
	errIdentityMissing  = errors.New("supernode routing server requires a peer ID identity")
	errPeerstoreMissing = errors.New("supernode routing server requires a peerstore")
	errServersMissing   = errors.New("supernode routing client requires at least 1 server peer")
)

// SupernodeServer returns a configuration for a routing server that stores
// routing records to the provided datastore. Only routing records are store in
// the datastore.
func SupernodeServer(recordSource ds.Datastore) core.RoutingOption {
	return func(ctx context.Context, ph host.Host, dstore repo.Datastore) (routing.IpfsRouting, error) {
		server, err := supernode.NewServer(recordSource, ph.Peerstore(), ph.ID())
		if err != nil {
			return nil, err
		}
		proxy := &gcproxy.Loopback{
			Handler: server,
			Local:   ph.ID(),
		}
		ph.SetStreamHandler(gcproxy.ProtocolSNR, proxy.HandleStream)
		return supernode.NewClient(proxy, ph, ph.Peerstore(), ph.ID())
	}
}

// TODO doc
func SupernodeClient(remotes ...pstore.PeerInfo) core.RoutingOption {
	return func(ctx context.Context, ph host.Host, dstore repo.Datastore) (routing.IpfsRouting, error) {
		if len(remotes) < 1 {
			return nil, errServersMissing
		}

		proxy := gcproxy.Standard(ph, remotes)
		ph.SetStreamHandler(gcproxy.ProtocolSNR, proxy.HandleStream)
		return supernode.NewClient(proxy, ph, ph.Peerstore(), ph.ID())
	}
}
