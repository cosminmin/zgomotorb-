# The go-ipfs config file

The go-ipfs config file is a JSON document located at `$IPFS_PATH/config`. It
is read once at node instantiation, either for an offline command, or when
starting the daemon. Commands that execute on a running daemon do not read the
config file at runtime.

#### Profiles

Configuration profiles allow to tweak configuration quickly. Profiles can be
applied with `--profile` flag to `ipfs init` or with the `ipfs config profile
apply` command. When a profile is applied a backup of the configuration file
will be created in `$IPFS_PATH`.

Available profiles:

- `server`

  Recommended for nodes with public IPv4 address (servers, VPSes, etc.),
  disables host and content discovery in local networks.

- `local-discovery`

  Sets default values to fields affected by `server` profile, enables
  discovery in local networks.

- `test`

  Reduces external interference, useful for running ipfs in test environments.
  Note that with these settings node won't be able to talk to the rest of the
  network without manual bootstrap.

- `default-networking`

  Restores default network settings. Inverse profile of the `test` profile.

- `badgerds`

  Replaces default datastore configuration with experimental badger datastore.
  If you apply this profile after `ipfs init`, you will need to convert your
  datastore to the new configuration. You can do this using
  [ipfs-ds-convert](https://github.com/ipfs/ipfs-ds-convert)

  WARNING: badger datastore is experimental. Make sure to backup your data
  frequently.

- `default-datastore`

  Restores default datastore configuration.

- `lowpower`

  Reduces daemon overhead on the system. May affect node functionality,
  performance of content discovery and data fetching may be degraded.

- `randomports`

  Generate random port for swarm.

## Table of Contents

- [`Addresses`](#addresses)
    - [`Addresses.API`](#addressesapi)
    - [`Addresses.Gateway`](#addressesgateway)
    - [`Addresses.Swarm`](#addressesswarm)
    - [`Addresses.Announce`](#addressesannounce)
    - [`Addresses.NoAnnounce`](#addressesnoannounce)
- [`API`](#api)
    - [`API.HTTPHeaders`](#apihttpheaders)
- [`Bootstrap`](#bootstrap)
- [`Datastore`](#datastore)
    - [`Datastore.StorageMax`](#datastorestoragemax)
    - [`Datastore.StorageGCWatermark`](#datastorestoragegcwatermark)
    - [`Datastore.GCPeriod`](#datastoregcperiod)
    - [`Datastore.HashOnRead`](#datastorehashonread)
    - [`Datastore.BloomFilterSize`](#datastorebloomfiltersize)
    - [`Datastore.Spec`](#datastorespec)
- [`Discovery`](#discovery)
    - [`Discovery.MDNS`](#discoverymdns)
        - [`Discovery.MDNS.Enabled`](#discoverymdnsenabled)
        - [`Discovery.MDNS.Interval`](#discoverymdnsinterval)
- [`Routing`](#routing)
    - [`Routing.Type`](#routingtype)
- [`Gateway`](#gateway)
    - [`Gateway.NoFetch`](#gatewaynofetch)
    - [`Gateway.HTTPHeaders`](#gatewayhttpheaders)
    - [`Gateway.RootRedirect`](#gatewayrootredirect)
    - [`Gateway.Writable`](#gatewaywritable)
    - [`Gateway.PathPrefixes`](#gatewaypathprefixes)
- [`Identity`](#identity)
    - [`Identity.PeerID`](#identitypeerid)
    - [`Identity.PrivKey`](#identityprivkey)
- [`Ipns`](#ipns)
    - [`Ipns.RepublishPeriod`](#ipnsrepublishperiod)
    - [`Ipns.RecordLifetime`](#ipnsrecordlifetime)
    - [`Ipns.ResolveCacheSize`](#ipnsresolvecachesize)
- [`Mounts`](#mounts)
    - [`Mounts.IPFS`](#mountsipfs)
    - [`Mounts.IPNS`](#mountsipns)
    - [`Mounts.FuseAllowOther`](#mountsfuseallowother)
- [`Reprovider`](#reprovider)
    - [`Reprovider.Interval`](#reproviderinterval)
    - [`Reprovider.Strategy`](#reproviderstrategy)
- [`Swarm`](#swarm)
    - [`Swarm.AddrFilters`](#swarmaddrfilters)
    - [`Swarm.DisableBandwidthMetrics`](#swarmdisablebandwidthmetrics)
    - [`Swarm.DisableNatPortMap`](#swarmdisablenatportmap)
    - [`Swarm.DisableRelay`](#swarmdisablerelay)
    - [`Swarm.EnableRelayHop`](#swarmenablerelayhop)
    - [`Swarm.EnableAutoRelay`](#swarmenableautorelay)
    - [`Swarm.EnableAutoNATService`](#swarmenableautonatservice)
    - [`Swarm.ConnMgr`](#swarmconnmgr)
        - [`Swarm.ConnMgr.Type`](#swarmconnmgrtype)
        - [`Swarm.ConnMgr.LowWater`](#swarmconnmgrlowwater)
        - [`Swarm.ConnMgr.HighWater`](#swarmconnmgrhighwater)
        - [`Swarm.ConnMgr.GracePeriod`](#swarmconnmgrgraceperiod)

## `Addresses`

Contains information about various listener addresses to be used by this node.

### `Addresses.API`

Multiaddr or array of multiaddrs describing the address to serve the local HTTP
API on.

Supported Transports:

* tcp/ip{4,6} - `/ipN/.../tcp/...`
* unix - `/unix/path/to/socket`

Default: `/ip4/127.0.0.1/tcp/5001`

### `Addresses.Gateway`

Multiaddr or array of multiaddrs describing the address to serve the local
gateway on.

Supported Transports:

* tcp/ip{4,6} - `/ipN/.../tcp/...`
* unix - `/unix/path/to/socket`

Default: `/ip4/127.0.0.1/tcp/8080`

### `Addresses.Swarm`

Array of multiaddrs describing which addresses to listen on for p2p swarm
connections.

Supported Transports:

* tcp/ip{4,6} - `/ipN/.../tcp/...`
* websocket - `/ipN/.../tcp/.../ws`
* quic - `/ipN/.../udp/.../quic`

Default:
```json
[
  "/ip4/0.0.0.0/tcp/4001",
  "/ip6/::/tcp/4001"
]
```

### `Addresses.Announce`

If non-empty, this array specifies the swarm addresses to announce to the
network. If empty, the daemon will announce inferred swarm addresses.

Default: `[]`

### `Addresses.NoAnnounce`
Array of swarm addresses not to announce to the network.

Default: `[]`

## `API`
Contains information used by the API gateway.

### `API.HTTPHeaders`
Map of HTTP headers to set on responses from the API HTTP server.

Example:
```json
{
	"Foo": ["bar"]
}
```

Default: `null`

## `Bootstrap`

Bootstrap is an array of multiaddrs of trusted nodes to connect to in order to
initiate a connection to the network.

Default: The ipfs.io bootstrap nodes

## `Datastore`

Contains information related to the construction and operation of the on-disk
storage system.

### `Datastore.StorageMax`

A soft upper limit for the size of the ipfs repository's datastore. With `StorageGCWatermark`,
is used to calculate whether to trigger a gc run (only if `--enable-gc` flag is set).

Default: `10GB`

### `Datastore.StorageGCWatermark`

The percentage of the `StorageMax` value at which a garbage collection will be
triggered automatically if the daemon was run with automatic gc enabled (that
option defaults to false currently).

Default: `90`

### `Datastore.GCPeriod`

A time duration specifying how frequently to run a garbage collection. Only used
if automatic gc is enabled.

Default: `1h`

### `Datastore.HashOnRead`

A boolean value. If set to true, all block reads from disk will be hashed and
verified. This will cause increased CPU utilization.

Default: `false`

### `Datastore.BloomFilterSize`

A number representing the size in bytes of the blockstore's [bloom
filter](https://en.wikipedia.org/wiki/Bloom_filter). A value of zero represents
the feature being disabled.

This site generates useful graphs for various bloom filter values:
<https://hur.st/bloomfilter/?n=1e6&p=0.01&m=&k=7> You may use it to find a
preferred optimal value, where `m` is `BloomFilterSize` in bits. Remember to
convert the value `m` from bits, into bytes for use as `BloomFilterSize` in the
config file. For example, for 1,000,000 blocks, expecting a 1% false positive
rate, you'd end up with a filter size of 9592955 bits, so for `BloomFilterSize`
we'd want to use 1199120 bytes. As of writing, [7 hash
functions](https://github.com/ipfs/go-ipfs-blockstore/blob/547442836ade055cc114b562a3cc193d4e57c884/caching.go#L22)
are used, so the constant `k` is 7 in the formula.


Default: `0`

### `Datastore.Spec`

Spec defines the structure of the ipfs datastore. It is a composable structure,
where each datastore is represented by a json object. Datastores can wrap other
datastores to provide extra functionality (eg metrics, logging, or caching).

This can be changed manually, however, if you make any changes that require a
different on-disk structure, you will need to run the [ipfs-ds-convert
tool](https://github.com/ipfs/ipfs-ds-convert) to migrate data into the new
structures.

For more information on possible values for this configuration option, see
docs/datastores.md

Default:
```
{
  "mounts": [
	{
	  "child": {
		"path": "blocks",
		"shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",
		"sync": true,
		"type": "flatfs"
	  },
	  "mountpoint": "/blocks",
	  "prefix": "flatfs.datastore",
	  "type": "measure"
	},
	{
	  "child": {
		"compression": "none",
		"path": "datastore",
		"type": "levelds"
	  },
	  "mountpoint": "/",
	  "prefix": "leveldb.datastore",
	  "type": "measure"
	}
  ],
  "type": "mount"
}
```

## `Discovery`

Contains options for configuring ipfs node discovery mechanisms.

### `Discovery.MDNS`

Options for multicast dns peer discovery.

#### `Discovery.MDNS.Enabled`

A boolean value for whether or not mdns should be active.

Default: `true`

#### `Discovery.MDNS.Interval`

A number of seconds to wait between discovery checks.

## `Routing`

Contains options for content routing mechanisms.

### `Routing.Type`

Content routing mode. Can be overridden with daemon `--routing` flag. When set
to `dhtclient`, the node won't join the DHT but can still use it to find
content.

Valid modes are:
  - `dht` (default)
  - `dhtclient`
  - `none`
  
**Example:**

```json
{
  "Routing": {
    "Type": "dhtclient"
  }
}
```  
  

## `Gateway`

Options for the HTTP gateway.

### `Gateway.NoFetch`

When set to true, the gateway will only serve content already in the local repo
and will not fetch files from the network.

Default: `false`

### `Gateway.HTTPHeaders`

Headers to set on gateway responses.

Default:
```json
{
	"Access-Control-Allow-Headers": [
		"X-Requested-With"
	],
	"Access-Control-Allow-Methods": [
		"GET"
	],
	"Access-Control-Allow-Origin": [
		"*"
	]
}
```

### `Gateway.RootRedirect`

A url to redirect requests for `/` to.

Default: `""`

### `Gateway.Writable`

A boolean to configure whether the gateway is writeable or not.

Default: `false`


### `Gateway.PathPrefixes`

Array of acceptable url paths that a client can specify in X-Ipfs-Path-Prefix
header.

The X-Ipfs-Path-Prefix header is used to specify a base path to prepend to links
in directory listings and for trailing-slash redirects. It is intended to be set
by a frontend http proxy like nginx.

Example: We mount `blog.ipfs.io` (a dnslink page) at `ipfs.io/blog`.

**.ipfs/config**
```json
"Gateway": {
  "PathPrefixes": ["/blog"],
```

**nginx_ipfs.conf**
```nginx
location /blog/ {
  rewrite "^/blog(/.*)$" $1 break;
  proxy_set_header Host blog.ipfs.io;
  proxy_set_header X-Ipfs-Gateway-Prefix /blog;
  proxy_pass http://127.0.0.1:8080;
}
```

Default: `[]`

## `Identity`

### `Identity.PeerID`

The unique PKI identity label for this configs peer. Set on init and never read,
its merely here for convenience. Ipfs will always generate the peerID from its
keypair at runtime.

### `Identity.PrivKey`

The base64 encoded protobuf describing (and containing) the nodes private key.

## `Ipns`

### `Ipns.RepublishPeriod`

A time duration specifying how frequently to republish ipns records to ensure
they stay fresh on the network. If unset, we default to 4 hours.

### `Ipns.RecordLifetime`

A time duration specifying the value to set on ipns records for their validity
lifetime.

If unset, we default to 24 hours.

### `Ipns.ResolveCacheSize`

The number of entries to store in an LRU cache of resolved ipns entries. Entries
will be kept cached until their lifetime is expired.

Default: `128`

## `Mounts`

FUSE mount point configuration options.

### `Mounts.IPFS`

Mountpoint for `/ipfs/`.

### `Mounts.IPNS`

Mountpoint for `/ipns/`.

### `Mounts.FuseAllowOther`

Sets the FUSE allow other option on the mountpoint.

## `Reprovider`

### `Reprovider.Interval`

Sets the time between rounds of reproviding local content to the routing
system. If unset, it defaults to 12 hours. If set to the value `"0"` it will
disable content reproviding.

Note: disabling content reproviding will result in other nodes on the network
not being able to discover that you have the objects that you have. If you want
to have this disabled and keep the network aware of what you have, you must
manually announce your content periodically.

### `Reprovider.Strategy`

Tells reprovider what should be announced. Valid strategies are:
  - "all" (default) - announce all stored data
  - "pinned" - only announce pinned data
  - "roots" - only announce directly pinned keys and root keys of recursive pins

## `Swarm`

Options for configuring the swarm.

### `Swarm.AddrFilters`

An array of addresses (multiaddr netmasks) to not dial. By default, IPFS nodes
advertise _all_ addresses, even internal ones. This makes it easier for nodes on
the same network to reach each other. Unfortunately, this means that an IPFS
node will try to connect to one or more private IP addresses whenever dialing
another node, even if this other node is on a different network. This may may
trigger netscan alerts on some hosting providers or cause strain in some setups.

The `server` configuration profile fills up this list with sensible defaults,
preventing dials to all non-routable IP addresses (e.g., `192.168.0.0/16`) but
you should always check settings against your own network and/or hosting
provider.


### `Swarm.DisableBandwidthMetrics`

A boolean value that when set to true, will cause ipfs to not keep track of
bandwidth metrics. Disabling bandwidth metrics can lead to a slight performance
improvement, as well as a reduction in memory usage.

### `Swarm.DisableNatPortMap`

Disable automatic NAT port forwarding.

When not disabled (default), go-ipfs asks NAT devices (e.g., routers), to open
up an external port and forward it to the port go-ipfs is running on. When this
works (i.e., when your router supports NAT port forwarding), it makes the local
go-ipfs node accessible from the public internet.

### `Swarm.DisableRelay`

Disables the p2p-circuit relay transport.

### `Swarm.EnableRelayHop`

Enables HOP relay for the node.

If this is enabled, the node will act as an intermediate (Hop Relay) node in
relay circuits for connected peers.

### `Swarm.EnableAutoRelay`

Enables automatic relay for this node.

If the node is a HOP relay (`EnableRelayHop` is true) then it will advertise
itself as a relay through the DHT. Otherwise, the node will test its own NAT
situation (dialability) using passively discovered AutoNAT services. If the node
is not publicly reachable, then it will seek HOP relays advertised through the
DHT and override its public address(es) with relay addresses.

### `Swarm.EnableAutoNATService`

Enables the AutoNAT service for this node.

The service allows peers to discover their NAT situation by requesting dial
backs to their public addresses. This should only be enabled on publicly
reachable nodes.

### `Swarm.ConnMgr`

The connection manager determines which and how many connections to keep and can
be configured to keep.

#### `Swarm.ConnMgr.Type`

Sets the type of connection manager to use, options are: `"none"` (no connection
management) and `"basic"`.

#### Basic Connection Manager

##### `Swarm.ConnMgr.LowWater`

LowWater is the minimum number of connections to maintain.

##### `Swarm.ConnMgr.HighWater`

HighWater is the number of connections that, when exceeded, will trigger a
connection GC operation.

##### `Swarm.ConnMgr.GracePeriod`

GracePeriod is a time duration that new connections are immune from being closed
by the connection manager.

The "basic" connection manager tries to keep between `LowWater` and `HighWater`
connections. It works by:

1. Keeping all connections until `HighWater` connections is reached.
2. Once `HighWater` is reached, it closes connections until `LowWater` is
   reached.
3. To prevent thrashing, it never closes connections established within the
   `GracePeriod`.

**Example:**

```json
{
  "Swarm": {
    "ConnMgr": {
      "Type": "basic",
      "LowWater": 100,
      "HighWater": 200,
      "GracePeriod": "30s"
    }
  }
}
```
