package namesys

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/ipfs/go-ipfs/namesys/pb"
	path "github.com/ipfs/go-ipfs/path"
	pin "github.com/ipfs/go-ipfs/pin"
	dshelp "github.com/ipfs/go-ipfs/thirdparty/ds-help"
	ft "github.com/ipfs/go-ipfs/unixfs"

	u "gx/ipfs/QmNiJuT8Ja3hMVpBHXv3Q6dwmperaQ6JjLtpMQgMCD7xvx/go-ipfs-util"
	ds "gx/ipfs/QmPpegoMqhAEqjncrzArm7KVWAkCm78rqL2DPuNjhPrshg/go-datastore"
	routing "gx/ipfs/QmRijoA6zGS98ELTDbGsLWPZbVotYsGbjp3RbXcKCYBeon/go-libp2p-routing"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	peer "gx/ipfs/Qma7H6RW8wRrfZpNSXwxYGcd1E149s42FpWNpDNieSVrnU/go-libp2p-peer"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	record "gx/ipfs/QmbsY8Pr6s3uZsKg7rzBtGDKeCtdoAhNaMTCXBUbvb1eCV/go-libp2p-record"
	dhtpb "gx/ipfs/QmbsY8Pr6s3uZsKg7rzBtGDKeCtdoAhNaMTCXBUbvb1eCV/go-libp2p-record/pb"
)

// ErrExpiredRecord should be returned when an ipns record is
// invalid due to being too old
var ErrExpiredRecord = errors.New("expired record")

// ErrUnrecognizedValidity is returned when an IpnsRecord has an
// unknown validity type.
var ErrUnrecognizedValidity = errors.New("unrecognized validity type")

// ErrInvalidPath should be returned when an ipns record path
// is not in a valid format
var ErrInvalidPath = errors.New("record path invalid")

const PublishPutValTimeout = time.Minute
const DefaultRecordTTL = 24 * time.Hour

// ipnsPublisher is capable of publishing and resolving names to the IPFS
// routing system.
type ipnsPublisher struct {
	routing routing.ValueStore
	ds      ds.Datastore
}

// NewRoutingPublisher constructs a publisher for the IPFS Routing name system.
func NewRoutingPublisher(route routing.ValueStore, ds ds.Datastore) *ipnsPublisher {
	if ds == nil {
		panic("nil datastore")
	}
	return &ipnsPublisher{routing: route, ds: ds}
}

// Publish implements Publisher. Accepts a keypair and a value,
// and publishes it out to the routing system
func (p *ipnsPublisher) Publish(ctx context.Context, k ci.PrivKey, value path.Path) error {
	log.Debugf("Publish %s", value)
	return p.PublishWithEOL(ctx, k, value, time.Now().Add(DefaultRecordTTL))
}

// PublishWithEOL is a temporary stand in for the ipns records implementation
// see here for more details: https://github.com/ipfs/specs/tree/master/records
func (p *ipnsPublisher) PublishWithEOL(ctx context.Context, k ci.PrivKey, value path.Path, eol time.Time) error {

	id, err := peer.IDFromPrivateKey(k)
	if err != nil {
		return err
	}

	_, ipnskey := IpnsKeysForID(id)

	// get previous records sequence number
	seqnum, err := p.getPreviousSeqNo(ctx, ipnskey)
	if err != nil {
		return err
	}

	// increment it
	seqnum++

	return PutRecordToRouting(ctx, k, value, seqnum, eol, p.routing, id)
}

func (p *ipnsPublisher) getPreviousSeqNo(ctx context.Context, ipnskey string) (uint64, error) {
	prevrec, err := p.ds.Get(dshelp.NewKeyFromBinary([]byte(ipnskey)))
	if err != nil && err != ds.ErrNotFound {
		// None found, lets start at zero!
		return 0, err
	}
	var val []byte
	if err == nil {
		prbytes, ok := prevrec.([]byte)
		if !ok {
			return 0, fmt.Errorf("unexpected type returned from datastore: %#v", prevrec)
		}
		dhtrec := new(dhtpb.Record)
		err := proto.Unmarshal(prbytes, dhtrec)
		if err != nil {
			return 0, err
		}

		val = dhtrec.GetValue()
	} else {
		// try and check the dht for a record
		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()

		rv, err := p.routing.GetValue(ctx, ipnskey)
		if err != nil {
			// no such record found, start at zero!
			return 0, nil
		}

		val = rv
	}

	e := new(pb.IpnsEntry)
	err = proto.Unmarshal(val, e)
	if err != nil {
		return 0, err
	}

	return e.GetSequence(), nil
}

// setting the TTL on published records is an experimental feature.
// as such, i'm using the context to wire it through to avoid changing too
// much code along the way.
func checkCtxTTL(ctx context.Context) (time.Duration, bool) {
	v := ctx.Value("ipns-publish-ttl")
	if v == nil {
		return 0, false
	}

	d, ok := v.(time.Duration)
	return d, ok
}

func PutRecordToRouting(ctx context.Context, k ci.PrivKey, value path.Path, seqnum uint64, eol time.Time, r routing.ValueStore, id peer.ID) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	namekey, ipnskey := IpnsKeysForID(id)
	entry, err := CreateRoutingEntryData(k, value, seqnum, eol)
	if err != nil {
		return err
	}

	ttl, ok := checkCtxTTL(ctx)
	if ok {
		entry.Ttl = proto.Uint64(uint64(ttl.Nanoseconds()))
	}

	errs := make(chan error, 2) // At most two errors (IPNS, and public key)

	// Attempt to extract the public key from the ID
	extractedPublicKey := id.ExtractPublicKey()

	go func() {
		errs <- PublishEntry(ctx, r, ipnskey, entry)
	}()

	// Publish the public key if a public key cannot be extracted from the ID
	if extractedPublicKey == nil {
		go func() {
			errs <- PublishPublicKey(ctx, r, namekey, k.GetPublic())
		}()

		if err := waitOnErrChan(ctx, errs); err != nil {
			return err
		}
	}

	return waitOnErrChan(ctx, errs)
}

func waitOnErrChan(ctx context.Context, errs chan error) error {
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func PublishPublicKey(ctx context.Context, r routing.ValueStore, k string, pubk ci.PubKey) error {
	log.Debugf("Storing pubkey at: %s", k)
	pkbytes, err := pubk.Bytes()
	if err != nil {
		return err
	}

	// Store associated public key
	timectx, cancel := context.WithTimeout(ctx, PublishPutValTimeout)
	defer cancel()
	return r.PutValue(timectx, k, pkbytes)
}

func PublishEntry(ctx context.Context, r routing.ValueStore, ipnskey string, rec *pb.IpnsEntry) error {
	timectx, cancel := context.WithTimeout(ctx, PublishPutValTimeout)
	defer cancel()

	data, err := proto.Marshal(rec)
	if err != nil {
		return err
	}

	log.Debugf("Storing ipns entry at: %s", ipnskey)
	// Store ipns entry at "/ipns/"+b58(h(pubkey))
	return r.PutValue(timectx, ipnskey, data)
}

func CreateRoutingEntryData(pk ci.PrivKey, val path.Path, seq uint64, eol time.Time) (*pb.IpnsEntry, error) {
	entry := new(pb.IpnsEntry)

	entry.Value = []byte(val)
	typ := pb.IpnsEntry_EOL
	entry.ValidityType = &typ
	entry.Sequence = proto.Uint64(seq)
	entry.Validity = []byte(u.FormatRFC3339(eol))

	sig, err := pk.Sign(ipnsEntryDataForSig(entry))
	if err != nil {
		return nil, err
	}
	entry.Signature = sig
	return entry, nil
}

func ipnsEntryDataForSig(e *pb.IpnsEntry) []byte {
	return bytes.Join([][]byte{
		e.Value,
		e.Validity,
		[]byte(fmt.Sprint(e.GetValidityType())),
	},
		[]byte{})
}

var IpnsRecordValidator = &record.ValidChecker{
	Func: ValidateIpnsRecord,
	Sign: true,
}

func IpnsSelectorFunc(k string, vals [][]byte) (int, error) {
	var recs []*pb.IpnsEntry
	for _, v := range vals {
		e := new(pb.IpnsEntry)
		err := proto.Unmarshal(v, e)
		if err == nil {
			recs = append(recs, e)
		} else {
			recs = append(recs, nil)
		}
	}

	return selectRecord(recs, vals)
}

func selectRecord(recs []*pb.IpnsEntry, vals [][]byte) (int, error) {
	var best_seq uint64
	best_i := -1

	for i, r := range recs {
		if r == nil || r.GetSequence() < best_seq {
			continue
		}

		if best_i == -1 || r.GetSequence() > best_seq {
			best_seq = r.GetSequence()
			best_i = i
		} else if r.GetSequence() == best_seq {
			rt, err := u.ParseRFC3339(string(r.GetValidity()))
			if err != nil {
				continue
			}

			bestt, err := u.ParseRFC3339(string(recs[best_i].GetValidity()))
			if err != nil {
				continue
			}

			if rt.After(bestt) {
				best_i = i
			} else if rt == bestt {
				if bytes.Compare(vals[i], vals[best_i]) > 0 {
					best_i = i
				}
			}
		}
	}
	if best_i == -1 {
		return 0, errors.New("no usable records in given set")
	}

	return best_i, nil
}

// ValidateIpnsRecord implements ValidatorFunc and verifies that the
// given 'val' is an IpnsEntry and that that entry is valid.
func ValidateIpnsRecord(r *record.ValidationRecord) error {
	if r.Namespace != "ipns" {
		return ErrInvalidPath
	}

	entry := new(pb.IpnsEntry)
	err := proto.Unmarshal(r.Value, entry)
	if err != nil {
		return err
	}

	// NOTE/FIXME(#4613): We're not checking the DHT signature/author here.
	// We're going to remove them in a followup commit and then check the
	// *IPNS* signature. However, to do that, we need to ensure we *have*
	// the public key and:
	//
	// 1. Don't want to fetch it from the network when handling PUTs.
	// 2. Do want to fetch it from the network when handling GETs.
	//
	// Therefore, we'll need to either:
	//
	// 1. Pass some for of offline hint to the validator (e.g., using a context).
	// 2. Ensure we pre-fetch the key when performing gets.
	//
	// This PR is already *way* too large so we're punting that fix to a new
	// PR.
	//
	// This is not a regression, it just restores the current (bad)
	// behavior.

	// Check that record has not expired
	switch entry.GetValidityType() {
	case pb.IpnsEntry_EOL:
		t, err := u.ParseRFC3339(string(entry.GetValidity()))
		if err != nil {
			log.Debug("failed parsing time for ipns record EOL")
			return err
		}
		if time.Now().After(t) {
			return ErrExpiredRecord
		}
	default:
		return ErrUnrecognizedValidity
	}
	return nil
}

// InitializeKeyspace sets the ipns record for the given key to
// point to an empty directory.
// TODO: this doesnt feel like it belongs here
func InitializeKeyspace(ctx context.Context, pub Publisher, pins pin.Pinner, key ci.PrivKey) error {
	emptyDir := ft.EmptyDirNode()

	// pin recursively because this might already be pinned
	// and doing a direct pin would throw an error in that case
	err := pins.Pin(ctx, emptyDir, true)
	if err != nil {
		return err
	}

	err = pins.Flush()
	if err != nil {
		return err
	}

	return pub.Publish(ctx, key, path.FromCid(emptyDir.Cid()))
}

func IpnsKeysForID(id peer.ID) (name, ipns string) {
	namekey := "/pk/" + string(id)
	ipnskey := "/ipns/" + string(id)

	return namekey, ipnskey
}
