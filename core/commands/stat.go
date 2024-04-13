package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	humanize "gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	metrics "gx/ipfs/QmQbh3Rb7KM37As3vkHYnEFnzkVXNCP8EYGtHz6g2fXk14/go-libp2p-metrics"
	cmdkit "gx/ipfs/QmSNbH2A1evCCbJSDC6u3RV3GGDhgu6pRGbXHvrN89tMKf/go-ipfs-cmdkit"
	cmds "gx/ipfs/QmUgr8HrEkQqXfBPtj1A2UEg1V7cvhUhDsmL44wFPCJk5k/go-ipfs-cmds"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
)

var StatsCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Query IPFS statistics.",
		ShortDescription: `'ipfs stats' is a set of commands to help look at statistics
for your IPFS node.
`,
		LongDescription: `'ipfs stats' is a set of commands to help look at statistics
for your IPFS node.`,
	},

	Subcommands: map[string]*cmds.Command{
		"bw":      statBwCmd,
		"repo":    repoStatCmd,
		"bitswap": bitswapStatCmd,
	},
}

var statBwCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Print ipfs bandwidth information.",
		ShortDescription: `'ipfs stats bw' prints bandwidth information for the ipfs daemon.
It displays: TotalIn, TotalOut, RateIn, RateOut.
		`,
		LongDescription: `'ipfs stats bw' prints bandwidth information for the ipfs daemon.
It displays: TotalIn, TotalOut, RateIn, RateOut.

By default, overall bandwidth and all protocols are shown. To limit bandwidth
to a particular peer, use the 'peer' option along with that peer's multihash
id. To specify a specific protocol, use the 'proto' option. The 'peer' and
'proto' options cannot be specified simultaneously. The protocols that are
queried using this method are outlined in the specification:
https://github.com/libp2p/specs/blob/master/7-properties.md#757-protocol-multicodecs

Example protocol options:
  - /ipfs/id/1.0.0
  - /ipfs/bitswap
  - /ipfs/dht

Example:

    > ipfs stats bw -t /ipfs/bitswap
    Bandwidth
    TotalIn: 5.0MB
    TotalOut: 0B
    RateIn: 343B/s
    RateOut: 0B/s
    > ipfs stats bw -p QmepgFW7BHEtU4pZJdxaNiv75mKLLRQnPi1KaaXmQN4V1a
    Bandwidth
    TotalIn: 4.9MB
    TotalOut: 12MB
    RateIn: 0B/s
    RateOut: 0B/s
`,
	},
	Options: []cmdkit.Option{
		cmdkit.StringOption("peer", "p", "Specify a peer to print bandwidth for."),
		cmdkit.StringOption("proto", "t", "Specify a protocol to print bandwidth for."),
		cmdkit.BoolOption("poll", "Print bandwidth at an interval.").Default(false),
		cmdkit.StringOption("interval", "i", `Time interval to wait between updating output, if 'poll' is true.

    This accepts durations such as "300s", "1.5h" or "2h45m". Valid time units are:
    "ns", "us" (or "µs"), "ms", "s", "m", "h".`).Default("1s"),
	},

	Run: func(req cmds.Request, res cmds.ResponseEmitter) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		// Must be online!
		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmdkit.ErrClient)
			return
		}

		if nd.Reporter == nil {
			res.SetError(fmt.Errorf("bandwidth reporter disabled in config"), cmdkit.ErrNormal)
			return
		}

		pstr, pfound, err := req.Option("peer").String()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		tstr, tfound, err := req.Option("proto").String()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}
		if pfound && tfound {
			res.SetError(errors.New("please only specify peer OR protocol"), cmdkit.ErrClient)
			return
		}

		var pid peer.ID
		if pfound {
			checkpid, err := peer.IDB58Decode(pstr)
			if err != nil {
				res.SetError(err, cmdkit.ErrNormal)
				return
			}
			pid = checkpid
		}

		timeS, _, err := req.Option("interval").String()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}
		interval, err := time.ParseDuration(timeS)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		doPoll, _, err := req.Option("poll").Bool()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		for {
			if pfound {
				stats := nd.Reporter.GetBandwidthForPeer(pid)
				res.Emit(&stats)
			} else if tfound {
				protoId := protocol.ID(tstr)
				stats := nd.Reporter.GetBandwidthForProtocol(protoId)
				res.Emit(&stats)
			} else {
				totals := nd.Reporter.GetBandwidthTotals()
				res.Emit(&totals)
			}
			if !doPoll {
				return
			}
			select {
			case <-time.After(interval):
			case <-req.Context().Done():
				return
			}
		}

	},
	Type: metrics.Stats{},
	PostRun: cmds.PostRunMap{
		cmds.CLI: func(req cmds.Request, re cmds.ResponseEmitter) cmds.ResponseEmitter {
			reNext, res := cmds.NewChanResponsePair(req)

			go func() {
				defer re.Close()

				polling, _, err := res.Request().Option("poll").Bool()
				if err != nil {
					return
				}

				fmt.Fprintln(os.Stdout, "Total Up    Total Down  Rate Up     Rate Down")
				for {
					v, err := res.Next()
					if !cmds.HandleError(err, res, re) {
						break
					}

					bs := v.(*metrics.Stats)

					if !polling {
						printStats(os.Stdout, bs)
						return
					}

					fmt.Fprintf(os.Stdout, "%8s    ", humanize.Bytes(uint64(bs.TotalOut)))
					fmt.Fprintf(os.Stdout, "%8s    ", humanize.Bytes(uint64(bs.TotalIn)))
					fmt.Fprintf(os.Stdout, "%8s/s  ", humanize.Bytes(uint64(bs.RateOut)))
					fmt.Fprintf(os.Stdout, "%8s/s  \n", humanize.Bytes(uint64(bs.RateIn)))
				}
			}()

			return reNext
		},
	},
}

func printStats(out io.Writer, bs *metrics.Stats) {
	fmt.Fprintln(out, "Bandwidth")
	fmt.Fprintf(out, "TotalIn: %s\n", humanize.Bytes(uint64(bs.TotalIn)))
	fmt.Fprintf(out, "TotalOut: %s\n", humanize.Bytes(uint64(bs.TotalOut)))
	fmt.Fprintf(out, "RateIn: %s/s\n", humanize.Bytes(uint64(bs.RateIn)))
	fmt.Fprintf(out, "RateOut: %s/s\n", humanize.Bytes(uint64(bs.RateOut)))
}
