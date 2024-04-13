package namesys

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	pb "github.com/ipfs/go-ipfs/namesys/pb"
	peer "gx/ipfs/QmVf8hTAsLLFtn4WPCRNdnaF2Eag2qTBS6uR8AiHPZARXy/go-libp2p-peer"
	pstore "gx/ipfs/QmZhsmorLpD9kmQ4ynbAu4vbKv2goMUnXazwGA4gnWHDjB/go-libp2p-peerstore"
	ic "gx/ipfs/Qme1knMqwt1hKZbc1BmQFmnm9f36nyQGwXxPGVpVJ9rMK5/go-libp2p-crypto"

	record "gx/ipfs/QmPWjVzxHeJdrjp4Jr2R2sPxBrMbBgGPWQtKwCKHHCBF7x/go-libp2p-record"
	u "gx/ipfs/QmPdKqUcHGFdeSpvjVoaTRPPstGif9GBZb5Q56RVw9o69A/go-ipfs-util"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
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

// ErrSignature should be returned when an ipns record fails
// signature verification
var ErrSignature = errors.New("record signature verification failed")

// ErrBadRecord should be returned when an ipns record cannot be unmarshalled
var ErrBadRecord = errors.New("record could not be unmarshalled")

// ErrKeyFormat should be returned when an ipns record key is
// incorrectly formatted (not a peer ID)
var ErrKeyFormat = errors.New("record key could not be parsed into peer ID")

// ErrPublicKeyNotFound should be returned when the public key
// corresponding to the ipns record path cannot be retrieved
// from the peer store
var ErrPublicKeyNotFound = errors.New("public key not found in peer store")

var ErrPublicKeyMismatch = errors.New("public key in record did not match expected pubkey")

type IpnsValidator struct {
	KeyBook pstore.KeyBook
}

func (v IpnsValidator) Validate(key string, value []byte) error {
	ns, pidString, err := record.SplitKey(key)
	if err != nil || ns != "ipns" {
		return ErrInvalidPath
	}

	// Parse the value into an IpnsEntry
	entry := new(pb.IpnsEntry)
	err = proto.Unmarshal(value, entry)
	if err != nil {
		return ErrBadRecord
	}

	// Get the public key defined by the ipns path
	pid, err := peer.IDFromString(pidString)
	if err != nil {
		log.Debugf("failed to parse ipns record key %s into peer ID", pidString)
		return ErrKeyFormat
	}

	pubk, err := v.getPublicKey(pid, entry)
	if err != nil {
		return err
	}

	// Check the ipns record signature with the public key
	if ok, err := pubk.Verify(ipnsEntryDataForSig(entry), entry.GetSignature()); err != nil || !ok {
		log.Debugf("failed to verify signature for ipns record %s", pidString)
		return ErrSignature
	}

	// Check that record has not expired
	switch entry.GetValidityType() {
	case pb.IpnsEntry_EOL:
		t, err := u.ParseRFC3339(string(entry.GetValidity()))
		if err != nil {
			log.Debugf("failed parsing time for ipns record EOL in record %s", pidString)
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

func (v IpnsValidator) getPublicKey(pid peer.ID, entry *pb.IpnsEntry) (ic.PubKey, error) {
	if entry.PubKey != nil {
		pk, err := ic.UnmarshalPublicKey(entry.PubKey)
		if err != nil {
			log.Debugf("public key in ipns record failed to parse: ", err)
			return nil, fmt.Errorf("unmarshaling pubkey in record: %s", err)
		}

		expPid, err := peer.IDFromPublicKey(pk)
		if err != nil {
			return nil, fmt.Errorf("could not regenerate peerID from pubkey: %s", err)
		}

		if pid != expPid {
			return nil, ErrPublicKeyMismatch
		}

		return pk, nil
	}

	pubk := v.KeyBook.PubKey(pid)
	if pubk == nil {
		log.Debugf("public key with hash %s not found in peer store", pid)
		return nil, ErrPublicKeyNotFound
	}
	return pubk, nil
}

// IpnsSelectorFunc selects the best record by checking which has the highest
// sequence number and latest EOL
func (v IpnsValidator) Select(k string, vals [][]byte) (int, error) {
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
	var bestSeq uint64
	besti := -1

	for i, r := range recs {
		if r == nil || r.GetSequence() < bestSeq {
			continue
		}
		rt, err := u.ParseRFC3339(string(r.GetValidity()))
		if err != nil {
			log.Errorf("failed to parse ipns record EOL %s", r.GetValidity())
			continue
		}

		if besti == -1 || r.GetSequence() > bestSeq {
			bestSeq = r.GetSequence()
			besti = i
		} else if r.GetSequence() == bestSeq {
			bestt, _ := u.ParseRFC3339(string(recs[besti].GetValidity()))
			if rt.After(bestt) {
				besti = i
			} else if rt == bestt {
				if bytes.Compare(vals[i], vals[besti]) > 0 {
					besti = i
				}
			}
		}
	}
	if besti == -1 {
		return 0, errors.New("no usable records in given set")
	}

	return besti, nil
}
