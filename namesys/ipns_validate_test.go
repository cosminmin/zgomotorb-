package namesys

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	opts "github.com/ipfs/go-ipfs/namesys/opts"
	pb "github.com/ipfs/go-ipfs/namesys/pb"
	path "github.com/ipfs/go-ipfs/path"

	record "gx/ipfs/QmPWjVzxHeJdrjp4Jr2R2sPxBrMbBgGPWQtKwCKHHCBF7x/go-libp2p-record"
	u "gx/ipfs/QmPdKqUcHGFdeSpvjVoaTRPPstGif9GBZb5Q56RVw9o69A/go-ipfs-util"
	testutil "gx/ipfs/QmPdxCaVp4jZ9RbxqZADvKH6kiCR5jHvdR5f2ycjAY6T2a/go-testutil"
	routing "gx/ipfs/QmUV9hDAAyjeGbxbXkJ2sYqZ6dTd1DXJ2REhYEkRm178Tg/go-libp2p-routing"
	ropts "gx/ipfs/QmUV9hDAAyjeGbxbXkJ2sYqZ6dTd1DXJ2REhYEkRm178Tg/go-libp2p-routing/options"
	peer "gx/ipfs/QmVf8hTAsLLFtn4WPCRNdnaF2Eag2qTBS6uR8AiHPZARXy/go-libp2p-peer"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	pstore "gx/ipfs/QmZhsmorLpD9kmQ4ynbAu4vbKv2goMUnXazwGA4gnWHDjB/go-libp2p-peerstore"
	mockrouting "gx/ipfs/Qmb1N7zdjG2FexpzWNj8T289u9QnQLEiSsTRadDGQxX32D/go-ipfs-routing/mock"
	ci "gx/ipfs/Qme1knMqwt1hKZbc1BmQFmnm9f36nyQGwXxPGVpVJ9rMK5/go-libp2p-crypto"
	ds "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dssync "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
)

func testValidatorCase(t *testing.T, priv ci.PrivKey, kbook pstore.KeyBook, key string, val []byte, eol time.Time, exp error) {
	t.Helper()

	match := func(t *testing.T, err error) {
		t.Helper()
		if err != exp {
			params := fmt.Sprintf("key: %s\neol: %s\n", key, eol)
			if exp == nil {
				t.Fatalf("Unexpected error %s for params %s", err, params)
			} else if err == nil {
				t.Fatalf("Expected error %s but there was no error for params %s", exp, params)
			} else {
				t.Fatalf("Expected error %s but got %s for params %s", exp, err, params)
			}
		}
	}

	testValidatorCaseMatchFunc(t, priv, kbook, key, val, eol, match)
}

func testValidatorCaseMatchFunc(t *testing.T, priv ci.PrivKey, kbook pstore.KeyBook, key string, val []byte, eol time.Time, matchf func(*testing.T, error)) {
	t.Helper()
	validator := IpnsValidator{kbook}

	data := val
	if data == nil {
		p := path.Path("/ipfs/QmfM2r8seH2GiRaC4esTjeraXEachRt8ZsSeGaWTPLyMoG")
		entry, err := CreateRoutingEntryData(priv, p, 1, eol)
		if err != nil {
			t.Fatal(err)
		}

		data, err = proto.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
	}

	matchf(t, validator.Validate(key, data))
}

func TestValidator(t *testing.T) {
	ts := time.Now()

	priv, id, _, _ := genKeys(t)
	priv2, id2, _, _ := genKeys(t)
	kbook := pstore.NewPeerstore()
	kbook.AddPubKey(id, priv.GetPublic())
	emptyKbook := pstore.NewPeerstore()

	testValidatorCase(t, priv, kbook, "/ipns/"+string(id), nil, ts.Add(time.Hour), nil)
	testValidatorCase(t, priv, kbook, "/ipns/"+string(id), nil, ts.Add(time.Hour*-1), ErrExpiredRecord)
	testValidatorCase(t, priv, kbook, "/ipns/"+string(id), []byte("bad data"), ts.Add(time.Hour), ErrBadRecord)
	testValidatorCase(t, priv, kbook, "/ipns/"+"bad key", nil, ts.Add(time.Hour), ErrKeyFormat)
	testValidatorCase(t, priv, emptyKbook, "/ipns/"+string(id), nil, ts.Add(time.Hour), ErrPublicKeyNotFound)
	testValidatorCase(t, priv2, kbook, "/ipns/"+string(id2), nil, ts.Add(time.Hour), ErrPublicKeyNotFound)
	testValidatorCase(t, priv2, kbook, "/ipns/"+string(id), nil, ts.Add(time.Hour), ErrSignature)
	testValidatorCase(t, priv, kbook, "//"+string(id), nil, ts.Add(time.Hour), ErrInvalidPath)
	testValidatorCase(t, priv, kbook, "/wrong/"+string(id), nil, ts.Add(time.Hour), ErrInvalidPath)
}

func mustMarshal(t *testing.T, entry *pb.IpnsEntry) []byte {
	t.Helper()
	data, err := proto.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEmbeddedPubKeyValidate(t *testing.T) {
	goodeol := time.Now().Add(time.Hour)
	kbook := pstore.NewPeerstore()

	pth := path.Path("/ipfs/QmfM2r8seH2GiRaC4esTjeraXEachRt8ZsSeGaWTPLyMoG")

	priv, _, _, ipnsk := genKeys(t)

	entry, err := CreateRoutingEntryData(priv, pth, 1, goodeol)
	if err != nil {
		t.Fatal(err)
	}

	testValidatorCase(t, priv, kbook, ipnsk, mustMarshal(t, entry), goodeol, ErrPublicKeyNotFound)

	pubkb, err := priv.GetPublic().Bytes()
	if err != nil {
		t.Fatal(err)
	}

	entry.PubKey = pubkb
	testValidatorCase(t, priv, kbook, ipnsk, mustMarshal(t, entry), goodeol, nil)

	entry.PubKey = []byte("probably not a public key")
	testValidatorCaseMatchFunc(t, priv, kbook, ipnsk, mustMarshal(t, entry), goodeol, func(t *testing.T, err error) {
		if !strings.Contains(err.Error(), "unmarshaling pubkey in record:") {
			t.Fatal("expected pubkey unmarshaling to fail")
		}
	})

	opriv, _, _, _ := genKeys(t)
	wrongkeydata, err := opriv.GetPublic().Bytes()
	if err != nil {
		t.Fatal(err)
	}

	entry.PubKey = wrongkeydata
	testValidatorCase(t, priv, kbook, ipnsk, mustMarshal(t, entry), goodeol, ErrPublicKeyMismatch)
}

func TestPeerIDPubKeyValidate(t *testing.T) {
	goodeol := time.Now().Add(time.Hour)
	kbook := pstore.NewPeerstore()

	pth := path.Path("/ipfs/QmfM2r8seH2GiRaC4esTjeraXEachRt8ZsSeGaWTPLyMoG")

	sk, pk, err := ci.GenerateEd25519Key(rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatal(err)
	}

	pid, err := peer.IDFromPublicKey(pk)
	if err != nil {
		t.Fatal(err)
	}

	ipnsk := "/ipns/" + string(pid)

	entry, err := CreateRoutingEntryData(sk, pth, 1, goodeol)
	if err != nil {
		t.Fatal(err)
	}

	dataNoKey, err := proto.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}

	testValidatorCase(t, sk, kbook, ipnsk, dataNoKey, goodeol, nil)
}

func TestResolverValidation(t *testing.T) {
	ctx := context.Background()
	rid := testutil.RandIdentityOrFatal(t)
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	peerstore := pstore.NewPeerstore()

	vstore := newMockValueStore(rid, dstore, peerstore)
	resolver := NewIpnsResolver(vstore)

	// Create entry with expiry in one hour
	priv, id, _, ipnsDHTPath := genKeys(t)
	ts := time.Now()
	p := path.Path("/ipfs/QmfM2r8seH2GiRaC4esTjeraXEachRt8ZsSeGaWTPLyMoG")
	entry, err := CreateRoutingEntryData(priv, p, 1, ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// Make peer's public key available in peer store
	err = peerstore.AddPubKey(id, priv.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	// Publish entry
	err = PublishEntry(ctx, vstore, ipnsDHTPath, entry)
	if err != nil {
		t.Fatal(err)
	}

	// Resolve entry
	resp, _, err := resolver.resolveOnce(ctx, id.Pretty(), opts.DefaultResolveOpts())
	if err != nil {
		t.Fatal(err)
	}
	if resp != p {
		t.Fatalf("Mismatch between published path %s and resolved path %s", p, resp)
	}

	// Create expired entry
	expiredEntry, err := CreateRoutingEntryData(priv, p, 1, ts.Add(-1*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// Publish entry
	err = PublishEntry(ctx, vstore, ipnsDHTPath, expiredEntry)
	if err != nil {
		t.Fatal(err)
	}

	// Record should fail validation because entry is expired
	_, _, err = resolver.resolveOnce(ctx, id.Pretty(), opts.DefaultResolveOpts())
	if err == nil {
		t.Fatal("ValidateIpnsRecord should have returned error")
	}

	// Create IPNS record path with a different private key
	priv2, id2, _, ipnsDHTPath2 := genKeys(t)

	// Make peer's public key available in peer store
	err = peerstore.AddPubKey(id2, priv2.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	// Publish entry
	err = PublishEntry(ctx, vstore, ipnsDHTPath2, entry)
	if err != nil {
		t.Fatal(err)
	}

	// Record should fail validation because public key defined by
	// ipns path doesn't match record signature
	_, _, err = resolver.resolveOnce(ctx, id2.Pretty(), opts.DefaultResolveOpts())
	if err == nil {
		t.Fatal("ValidateIpnsRecord should have failed signature verification")
	}

	// Publish entry without making public key available in peer store
	priv3, id3, pubkDHTPath3, ipnsDHTPath3 := genKeys(t)
	entry3, err := CreateRoutingEntryData(priv3, p, 1, ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	err = PublishEntry(ctx, vstore, ipnsDHTPath3, entry3)
	if err != nil {
		t.Fatal(err)
	}

	// Record should fail validation because public key is not available
	// in peer store or on network
	_, _, err = resolver.resolveOnce(ctx, id3.Pretty(), opts.DefaultResolveOpts())
	if err == nil {
		t.Fatal("ValidateIpnsRecord should have failed because public key was not found")
	}

	// Publish public key to the network
	err = PublishPublicKey(ctx, vstore, pubkDHTPath3, priv3.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	// Record should now pass validation because resolver will ensure
	// public key is available in the peer store by looking it up in
	// the DHT, which causes the DHT to fetch it and cache it in the
	// peer store
	_, _, err = resolver.resolveOnce(ctx, id3.Pretty(), opts.DefaultResolveOpts())
	if err != nil {
		t.Fatal(err)
	}
}

func genKeys(t *testing.T) (ci.PrivKey, peer.ID, string, string) {
	sr := u.NewTimeSeededRand()
	priv, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, sr)
	if err != nil {
		t.Fatal(err)
	}

	// Create entry with expiry in one hour
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	pubkDHTPath, ipnsDHTPath := IpnsKeysForID(pid)

	return priv, pid, pubkDHTPath, ipnsDHTPath
}

type mockValueStore struct {
	r         routing.ValueStore
	kbook     pstore.KeyBook
	Validator record.Validator
}

func newMockValueStore(id testutil.Identity, dstore ds.Datastore, kbook pstore.KeyBook) *mockValueStore {
	serv := mockrouting.NewServer()
	r := serv.ClientWithDatastore(context.Background(), id, dstore)
	return &mockValueStore{r, kbook, record.NamespacedValidator{
		"ipns": IpnsValidator{kbook},
		"pk":   record.PublicKeyValidator{},
	}}
}

func (m *mockValueStore) GetValue(ctx context.Context, k string, opts ...ropts.Option) ([]byte, error) {
	data, err := m.r.GetValue(ctx, k, opts...)
	if err != nil {
		return data, err
	}

	if err = m.Validator.Validate(k, data); err != nil {
		return nil, err
	}

	return data, err
}

func (m *mockValueStore) GetPublicKey(ctx context.Context, p peer.ID) (ci.PubKey, error) {
	pk := m.kbook.PubKey(p)
	if pk != nil {
		return pk, nil
	}

	pkkey := routing.KeyForPublicKey(p)
	val, err := m.GetValue(ctx, pkkey)
	if err != nil {
		return nil, err
	}

	pk, err = ci.UnmarshalPublicKey(val)
	if err != nil {
		return nil, err
	}

	return pk, m.kbook.AddPubKey(p, pk)
}

func (m *mockValueStore) PutValue(ctx context.Context, k string, d []byte, opts ...ropts.Option) error {
	return m.r.PutValue(ctx, k, d, opts...)
}
