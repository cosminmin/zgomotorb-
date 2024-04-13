// Package iface defines IPFS Core API which is a set of interfaces used to
// interact with IPFS nodes.
package iface

import (
	"context"
	"errors"
	"io"
	"time"

	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	cid "gx/ipfs/QmcZfnkapfECQGcLZaf9B79NRg7cRa9EnZh4LSbkCzwNvY/go-cid"
	ipld "gx/ipfs/Qme5bWv7wtjUNGsK2BNGVUFPKiuxWrsqrtvYwCLRw8YFES/go-ipld-format"
)

// Path is a generic wrapper for paths used in the API. A path can be resolved
// to a CID using one of Resolve functions in the API.
type Path interface {
	// String returns the path as a string.
	String() string
	// Cid returns cid referred to by path
	Cid() *cid.Cid
	// Root returns cid of root path
	Root() *cid.Cid
	// Resolved returns whether path has been fully resolved
	Resolved() bool
}

// TODO: should we really copy these?
//       if we didn't, godoc would generate nice links straight to go-ipld-format
type Node ipld.Node
type Link ipld.Link

type Reader interface {
	io.ReadSeeker
	io.Closer
}

// IpnsEntry specifies the interface to IpnsEntries
type IpnsEntry interface {
	// Name returns IpnsEntry name
	Name() string
	// Value returns IpnsEntry value
	Value() Path
}

// Key specifies the interface to Keys in KeyAPI Keystore
type Key interface {
	// Key returns key name
	Name() string
	// Path returns key path
	Path() Path
}

// CoreAPI defines an unified interface to IPFS for Go programs.
type CoreAPI interface {
	// Unixfs returns an implementation of Unixfs API.
	Unixfs() UnixfsAPI
	// Dag returns an implementation of Dag API.
	Dag() DagAPI
	// Name returns an implementation of Name API.
	Name() NameAPI
	// Key returns an implementation of Key API.
	Key() KeyAPI

	// ObjectAPI returns an implementation of Object API
	Object() ObjectAPI

	// ResolvePath resolves the path using Unixfs resolver
	ResolvePath(context.Context, Path) (Path, error)

	// ResolveNode resolves the path (if not resolved already) using Unixfs
	// resolver, gets and returns the resolved Node
	ResolveNode(context.Context, Path) (Node, error)
}

// UnixfsAPI is the basic interface to immutable files in IPFS
type UnixfsAPI interface {
	// Add imports the data from the reader into merkledag file
	Add(context.Context, io.Reader) (Path, error)

	// Cat returns a reader for the file
	Cat(context.Context, Path) (Reader, error)

	// Ls returns the list of links in a directory
	Ls(context.Context, Path) ([]*Link, error)
}

// DagAPI specifies the interface to IPLD
type DagAPI interface {
	// Put inserts data using specified format and input encoding.
	// Unless used with WithCodec or WithHash, the defaults "dag-cbor" and
	// "sha256" are used.
	Put(ctx context.Context, src io.Reader, opts ...options.DagPutOption) (Path, error)

	// WithInputEnc is an option for Put which specifies the input encoding of the
	// data. Default is "json", most formats/codecs support "raw"
	WithInputEnc(enc string) options.DagPutOption

	// WithCodec is an option for Put which specifies the multicodec to use to
	// serialize the object. Default is cid.DagCBOR (0x71)
	WithCodec(codec uint64) options.DagPutOption

	// WithHash is an option for Put which specifies the multihash settings to use
	// when hashing the object. Default is based on the codec used
	// (mh.SHA2_256 (0x12) for DagCBOR). If mhLen is set to -1, default length for
	// the hash will be used
	WithHash(mhType uint64, mhLen int) options.DagPutOption

	// Get attempts to resolve and get the node specified by the path
	Get(ctx context.Context, path Path) (Node, error)

	// Tree returns list of paths within a node specified by the path.
	Tree(ctx context.Context, path Path, opts ...options.DagTreeOption) ([]Path, error)

	// WithDepth is an option for Tree which specifies maximum depth of the
	// returned tree. Default is -1 (no depth limit)
	WithDepth(depth int) options.DagTreeOption
}

// NameAPI specifies the interface to IPNS.
//
// IPNS is a PKI namespace, where names are the hashes of public keys, and the
// private key enables publishing new (signed) values. In both publish and
// resolve, the default name used is the node's own PeerID, which is the hash of
// its public key.
//
// You can use .Key API to list and generate more names and their respective keys.
type NameAPI interface {
	// Publish announces new IPNS name
	Publish(ctx context.Context, path Path, opts ...options.NamePublishOption) (IpnsEntry, error)

	// WithValidTime is an option for Publish which specifies for how long the
	// entry will remain valid. Default value is 24h
	WithValidTime(validTime time.Duration) options.NamePublishOption

	// WithKey is an option for Publish which specifies the key to use for
	// publishing. Default value is "self" which is the node's own PeerID.
	// The key parameter must be either PeerID or keystore key alias.
	//
	// You can use KeyAPI to list and generate more names and their respective keys.
	WithKey(key string) options.NamePublishOption

	// Resolve attempts to resolve the newest version of the specified name
	Resolve(ctx context.Context, name string, opts ...options.NameResolveOption) (Path, error)

	// WithRecursive is an option for Resolve which specifies whether to perform a
	// recursive lookup. Default value is false
	WithRecursive(recursive bool) options.NameResolveOption

	// WithLocal is an option for Resolve which specifies if the lookup should be
	// offline. Default value is false
	WithLocal(local bool) options.NameResolveOption

	// WithCache is an option for Resolve which specifies if cache should be used.
	// Default value is true
	WithCache(cache bool) options.NameResolveOption
}

// KeyAPI specifies the interface to Keystore
type KeyAPI interface {
	// Generate generates new key, stores it in the keystore under the specified
	// name and returns a base58 encoded multihash of it's public key
	Generate(ctx context.Context, name string, opts ...options.KeyGenerateOption) (Key, error)

	// WithType is an option for Generate which specifies which algorithm
	// should be used for the key. Default is options.RSAKey
	//
	// Supported key types:
	// * options.RSAKey
	// * options.Ed25519Key
	WithType(algorithm string) options.KeyGenerateOption

	// WithSize is an option for Generate which specifies the size of the key to
	// generated. Default is -1
	//
	// value of -1 means 'use default size for key type':
	//  * 2048 for RSA
	WithSize(size int) options.KeyGenerateOption

	// Rename renames oldName key to newName. Returns the key and whether another
	// key was overwritten, or an error
	Rename(ctx context.Context, oldName string, newName string, opts ...options.KeyRenameOption) (Key, bool, error)

	// WithForce is an option for Rename which specifies whether to allow to
	// replace existing keys.
	WithForce(force bool) options.KeyRenameOption

	// List lists keys stored in keystore
	List(ctx context.Context) ([]Key, error)

	// Remove removes keys from keystore. Returns ipns path of the removed key
	Remove(ctx context.Context, name string) (Path, error)
}

// ObjectAPI specifies the interface to MerkleDAG and contains useful utilities
// for manipulating MerkleDAG data structures.
type ObjectAPI interface {
	// New creates new, empty (by default) dag-node.
	New(context.Context, ...options.ObjectNewOption) (Node, error)

	// WithType is an option for New which allows to change the type of created
	// dag node.
	//
	// Supported types:
	// * 'empty' - Empty node
	// * 'unixfs-dir' - Empty UnixFS directory
	WithType(string) options.ObjectNewOption

	// Put imports the data into merkledag
	Put(context.Context, io.Reader, ...options.ObjectPutOption) (Path, error)

	// WithInputEnc is an option for Put which specifies the input encoding of the
	// data. Default is "json".
	//
	// Supported encodings:
	// * "protobuf"
	// * "json"
	WithInputEnc(e string) options.ObjectPutOption

	// WithDataType specifies the encoding of data field when using Josn or XML
	// input encoding.
	//
	// Supported types:
	// * "text" (default)
	// * "base64"
	WithDataType(t string) options.ObjectPutOption

	// Get returns the node for the path
	Get(context.Context, Path) (Node, error)

	// Data returns reader for data of the node
	Data(context.Context, Path) (io.Reader, error)

	// Links returns lint or links the node contains
	Links(context.Context, Path) ([]*Link, error)

	// Stat returns information about the node
	Stat(context.Context, Path) (*ObjectStat, error)

	// AddLink adds a link under the specified path. child path can point to a
	// subdirectory within the patent which must be present (can be overridden
	// with WithCreate option).
	AddLink(ctx context.Context, base Path, name string, child Path, opts ...options.ObjectAddLinkOption) (Path, error)

	// WithCreate is an option for AddLink which specifies whether create required
	// directories for the child
	WithCreate(create bool) options.ObjectAddLinkOption

	// RmLink removes a link from the node
	RmLink(ctx context.Context, base Path, link string) (Path, error)

	// AppendData appends data to the node
	AppendData(context.Context, Path, io.Reader) (Path, error)

	// SetData sets the data contained in the node
	SetData(context.Context, Path, io.Reader) (Path, error)
}

// ObjectStat provides information about dag nodes
type ObjectStat struct {
	// Cid is the CID of the node
	Cid *cid.Cid

	// NumLinks is number of links the node contains
	NumLinks int

	// BlockSize is size of the raw serialized node
	BlockSize int

	// LinksSize is size of the links block section
	LinksSize int

	// DataSize is the size of data block section
	DataSize int

	// CumulativeSize is size of the tree (BlockSize + link sizes)
	CumulativeSize int
}

var ErrIsDir = errors.New("object is a directory")
var ErrOffline = errors.New("can't resolve, ipfs node is offline")
