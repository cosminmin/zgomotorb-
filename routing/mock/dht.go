package mockrouting

import (
	context "context"
	"github.com/ipfs/go-ipfs/thirdparty/testutil"
	dht "gx/ipfs/QmQcRLisUbREko56ThfgzdBorMGNfNjgqzvwuPPr1jFw6A/go-libp2p-kad-dht"
	ds "gx/ipfs/QmRWDav6mzWseLWeYfVd5fvUKiVe9xNH29YfMF438fG364/go-datastore"
	sync "gx/ipfs/QmRWDav6mzWseLWeYfVd5fvUKiVe9xNH29YfMF438fG364/go-datastore/sync"
	mocknet "gx/ipfs/QmRai5yZNL67pWCoznW7sBdFnqZrFULuJ5w8KhmRyhdgN4/go-libp2p/p2p/net/mock"
)

type mocknetserver struct {
	mn mocknet.Mocknet
}

func NewDHTNetwork(mn mocknet.Mocknet) Server {
	return &mocknetserver{
		mn: mn,
	}
}

func (rs *mocknetserver) Client(p testutil.Identity) Client {
	return rs.ClientWithDatastore(context.TODO(), p, ds.NewMapDatastore())
}

func (rs *mocknetserver) ClientWithDatastore(ctx context.Context, p testutil.Identity, ds ds.Datastore) Client {

	// FIXME AddPeer doesn't appear to be idempotent

	host, err := rs.mn.AddPeer(p.PrivateKey(), p.Address())
	if err != nil {
		panic("FIXME")
	}
	return dht.NewDHT(ctx, host, sync.MutexWrap(ds))
}

var _ Server = &mocknetserver{}
