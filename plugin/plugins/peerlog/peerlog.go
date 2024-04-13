package peerlog

import (
	"fmt"

	core "github.com/ipfs/go-ipfs/core"
	plugin "github.com/ipfs/go-ipfs/plugin"
	logging "github.com/ipfs/go-log"
	network "github.com/libp2p/go-libp2p-core/network"
)

var log = logging.Logger("plugin/peerlog")

// Log all the PeerIDs we see
//
// Usage:
//   GOLOG_FILE=~/peer.log IPFS_LOGGING_FMT=json ipfs daemon
// Output:
//   {"level":"info","ts":"2020-02-10T13:54:26.639Z","logger":"plugin/peerlog","caller":"peerlog/peerlog.go:51","msg":"connected","peer":"QmS2H72gdrekXJggGdE9SunXPntBqdkJdkXQJjuxcH8Cbt"}
//   {"level":"info","ts":"2020-02-10T13:54:59.095Z","logger":"plugin/peerlog","caller":"peerlog/peerlog.go:56","msg":"disconnected","peer":"QmS2H72gdrekXJggGdE9SunXPntBqdkJdkXQJjuxcH8Cbt"}
//
type peerLogPlugin struct{}

var _ plugin.PluginDaemonInternal = (*peerLogPlugin)(nil)

// Plugins is exported list of plugins that will be loaded
var Plugins = []plugin.Plugin{
	&peerLogPlugin{},
}

// Name returns the plugin's name, satisfying the plugin.Plugin interface.
func (*peerLogPlugin) Name() string {
	return "peerlog"
}

// Version returns the plugin's version, satisfying the plugin.Plugin interface.
func (*peerLogPlugin) Version() string {
	return "0.1.0"
}

// Init initializes plugin
func (*peerLogPlugin) Init(*plugin.Environment) error {
	return nil
}

func (*peerLogPlugin) Start(node *core.IpfsNode) error {
	// Ensure logs from this plugin get printed regardless of global IPFS_LOGGING value
	if err := logging.SetLogLevel("plugin/peerlog", "info"); err != nil {
		return fmt.Errorf("failed to set log level: %w", err)
	}
	var notifee network.NotifyBundle
	notifee.ConnectedF = func(net network.Network, conn network.Conn) {
		log.Infow("connected",
			"peer", conn.RemotePeer().Pretty(),
		)
	}
	notifee.DisconnectedF = func(net network.Network, conn network.Conn) {
		log.Infow("disconnected",
			"peer", conn.RemotePeer().Pretty(),
		)
	}
	node.PeerHost.Network().Notify(&notifee)
	return nil
}

func (*peerLogPlugin) Close() error {
	return nil
}
