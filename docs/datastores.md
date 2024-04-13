# Datastore Configuration Options

This document describes the different possible values for the `Datastore.Spec`
field in the ipfs configuration file.

## flatfs

Stores each key value pair as a file on the filesystem.

The shardFunc is prefixed with `/repo/flatfs/shard/v1` then followed by a descriptor of the sharding strategy. Some example values are:
- `/repo/flatfs/shard/v1/next-to-last/2`
  - Shards on the two next to last characters of the key
- `/repo/flatfs/shard/v1/prefix/2`
  - Shards based on the two character prefix of the key

```json
{
	"type": "flatfs",
	"path": "<relative path within repo for flatfs root>",
	"shardFunc": "<a descriptor of the sharding scheme>",
	"sync": true|false
}
```

NOTE: flatfs must only be used as a block store (mounted at `/blocks`) as it only partially implements the datastore interface. You can mount flatfs for /blocks only using the mount datastore (described below).

## levelds
Uses a leveldb database to store key value pairs.

```json
{
	"type": "levelds",
	"path": "<location of db inside repo>",
	"compression": "none" | "snappy",
}
```

## badgerds

Uses [badger](https://github.com/dgraph-io/badger) as a key value store.

* `syncWrites`: Flush every write to disk before continuing. Setting this to false is safe as go-ipfs will automatically flush writes to disk before and after performing critical operations like pinning. However, you can set this to true to be extra-safe (at the cost of a 2-3x slowdown when adding files).
* `truncate`: Truncate the DB if a partially written sector is found (defaults to true). There is no good reason to set this to false unless you want to manually recover partially written (and unpinned) blocks if go-ipfs crashes half-way through a adding a file.
* `vlogFileSize`: Sets the maximum size of a single value log file to the size specified, or to the default if unspecified.

```json
{
	"type": "badgerds",
	"path": "<location of badger inside repo>",
	"syncWrites": true|false,
	"truncate": true|false,
	"vlogFileSize": <max size of a value log>
}
```

## badger2ds

Uses [badger2](https://github.com/dgraph-io/badger) as a key value store.

* `syncWrites`: Flush every write to disk before continuing. Setting this to false is safe as go-ipfs will automatically flush writes to disk before and after performing critical operations like pinning. However, you can set this to true to be extra-safe (at the cost of a 2-3x slowdown when adding files).
* `truncate`: Truncate the DB if a partially written sector is found (defaults to true). There is no good reason to set this to false unless you want to manually recover partially written (and unpinned) blocks if go-ipfs crashes half-way through a adding a file.
* `compression`: Configure compression. When compression is enabled, every block is compressed using the specified algorithm. This option doesn't affect existing tables. Only the newly created tables are compressed. Compression can be configured to use the snappy algorithm or ZSTD with level 1, 2 or 3.  An unspecified value uses the default configuration.
* `blockCacheSize`: Specifies how much data cache should hold in memory. If compression is disabled, adding a cache leads to unnecessary overhead which may affect read performance. A value of 0 means no block cache, and an unspecified value uses the default configuration.
* `vlogFileSize`: Sets the maximum size of a single value log file to the size specified, or to the default if unspecified.

```json
{
	"type": "badger2ds",
	"path": "<location of badger2 inside repo>",
	"syncWrites": true|false,
	"truncate": true|false,
	"compression": "none"|"snappy"|"zstd1"|"zstd2"|"zstd3",
	"blockCacheSize": <size of block cache>,
	"vlogFileSize": <max size of a value log>
}
```

## mount

Allows specified datastores to handle keys prefixed with a given path.
The mountpoints are added as keys within the child datastore definitions.

```json
{
	"type": "mount",
	"mounts": [
		{
			// Insert other datastore definition here, but add the following key:
			"mountpoint": "/path/to/handle"
		},
		{
			// Insert other datastore definition here, but add the following key:
			"mountpoint": "/path/to/handle"
		},
	]
}
```

## measure

This datastore is a wrapper that adds metrics tracking to any datastore.

```json
{
	"type": "measure",
	"prefix": "sometag.datastore",
	"child": { datastore being wrapped }
}
```

