package commands

import (
	"fmt"
	"io"
	"net/http"

	cmdenv "github.com/ipfs/go-ipfs/core/commands/cmdenv"
	filestore "github.com/ipfs/go-ipfs/filestore"
	pin "github.com/ipfs/go-ipfs/pin"

	chunk "gx/ipfs/QmR4QQVkBZsZENRjYFVi8dEtPL3daZRNKk24m4r6WKJHNm/go-ipfs-chunker"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	cmds "gx/ipfs/QmaAP56JAwdjwisPTu4yx17whcjTr6y5JCSCF77Y1rahWV/go-ipfs-cmds"
	balanced "gx/ipfs/Qmbvw7kpSM2p6rbQ57WGRhhqNfCiNGW6EKH4xgHLw4bsnB/go-unixfs/importer/balanced"
	ihelper "gx/ipfs/Qmbvw7kpSM2p6rbQ57WGRhhqNfCiNGW6EKH4xgHLw4bsnB/go-unixfs/importer/helpers"
	trickle "gx/ipfs/Qmbvw7kpSM2p6rbQ57WGRhhqNfCiNGW6EKH4xgHLw4bsnB/go-unixfs/importer/trickle"
	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
)

var urlStoreCmd = &cmds.Command{
	Subcommands: map[string]*cmds.Command{
		"add": urlAdd,
	},
}

var urlAdd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Add URL via urlstore.",
		LongDescription: `
Add URLs to ipfs without storing the data locally.

The URL provided must be stable and ideally on a web server under your
control.

The file is added using raw-leaves but otherwise using the default
settings for 'ipfs add'.

This command is considered temporary until a better solution can be
found.  It may disappear or the semantics can change at any
time.
`,
	},
	Options: []cmdkit.Option{
		cmdkit.BoolOption(trickleOptionName, "t", "Use trickle-dag format for dag generation."),
		cmdkit.BoolOption(pinOptionName, "Pin this object when adding.").WithDefault(true),
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("url", true, false, "URL to add to IPFS"),
	},
	Type: &BlockStat{},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		url := req.Arguments[0]
		n, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		if !filestore.IsURL(url) {
			return fmt.Errorf("unsupported url syntax: %s", url)
		}

		cfg, err := n.Repo.Config()
		if err != nil {
			return err
		}

		if !cfg.Experimental.UrlstoreEnabled {
			return filestore.ErrUrlstoreNotEnabled
		}

		useTrickledag, _ := req.Options[trickleOptionName].(bool)
		dopin, _ := req.Options[pinOptionName].(bool)

		hreq, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		hres, err := http.DefaultClient.Do(hreq)
		if err != nil {
			return err
		}
		if hres.StatusCode != http.StatusOK {
			return fmt.Errorf("expected code 200, got: %d", hres.StatusCode)
		}

		if dopin {
			// Take the pinlock
			defer n.Blockstore.PinLock().Unlock()
		}

		chk := chunk.NewSizeSplitter(hres.Body, chunk.DefaultBlockSize)
		prefix := cid.NewPrefixV1(cid.DagProtobuf, mh.SHA2_256)
		dbp := &ihelper.DagBuilderParams{
			Dagserv:    n.DAG,
			RawLeaves:  true,
			Maxlinks:   ihelper.DefaultLinksPerBlock,
			NoCopy:     true,
			CidBuilder: &prefix,
			URL:        url,
		}

		layout := balanced.Layout
		if useTrickledag {
			layout = trickle.Layout
		}

		root, err := layout(dbp.New(chk))
		if err != nil {
			return err
		}

		c := root.Cid()
		if dopin {
			n.Pinning.PinWithMode(c, pin.Recursive)
			if err := n.Pinning.Flush(); err != nil {
				return err
			}
		}

		return cmds.EmitOnce(res, &BlockStat{
			Key:  c.String(),
			Size: int(hres.ContentLength),
		})
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, bs *BlockStat) error {
			_, err := fmt.Fprintln(w, bs.Key)
			return err
		}),
	},
}
