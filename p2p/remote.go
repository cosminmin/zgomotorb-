package p2p

import (
	"context"

	net "gx/ipfs/QmSTaEYUgDe1r581hxyd2u9582Hgp3KX4wGwYbRqz2u9Qh/go-libp2p-net"
	manet "gx/ipfs/QmV6FjemM1K8oXjrvuq3wuVWWoU2TLDPmNnKrxHzY3v6Ai/go-multiaddr-net"
	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
)

var maPrefix = "/" + ma.ProtocolWithCode(ma.P_IPFS).Name + "/"

// remoteListener accepts libp2p streams and proxies them to a manet host
type remoteListener struct {
	p2p *P2P

	// Application proto identifier.
	proto protocol.ID

	// Address to proxy the incoming connections to
	addr ma.Multiaddr
}

// ForwardRemote creates new p2p listener
func (p2p *P2P) ForwardRemote(ctx context.Context, proto protocol.ID, addr ma.Multiaddr) (Listener, error) {
	listener := &remoteListener{
		p2p: p2p,

		proto: proto,
		addr:  addr,
	}

	if err := p2p.ListenersP2P.Register(listener); err != nil {
		return nil, err
	}

	return listener, nil
}

func (l *remoteListener) handleStream(remote net.Stream) {
	local, err := manet.Dial(l.addr)
	if err != nil {
		remote.Reset()
		return
	}

	peer := remote.Conn().RemotePeer()

	peerMa, err := ma.NewMultiaddr(maPrefix + peer.Pretty())
	if err != nil {
		remote.Reset()
		return
	}

	stream := &Stream{
		Protocol: l.proto,

		OriginAddr: peerMa,
		TargetAddr: l.addr,
		peer:       peer,

		Local:  local,
		Remote: remote,

		Registry: l.p2p.Streams,
	}

	l.p2p.Streams.Register(stream)
}

func (l *remoteListener) Protocol() protocol.ID {
	return l.proto
}

func (l *remoteListener) ListenAddress() ma.Multiaddr {
	addr, err := ma.NewMultiaddr(maPrefix + l.p2p.identity.Pretty())
	if err != nil {
		panic(err)
	}
	return addr
}

func (l *remoteListener) TargetAddress() ma.Multiaddr {
	return l.addr
}

func (l *remoteListener) close() {}

func (l *remoteListener) key() string {
	return string(l.proto)
}
