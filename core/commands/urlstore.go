package commands

import (
	"fmt"
	"io"
	"net/http"

	cmdenv "github.com/ipfs/go-ipfs/core/commands/cmdenv"
	filestore "github.com/ipfs/go-ipfs/filestore"

	cmds "gx/ipfs/QmPTfgFTo9PFr1PvPKyKoeMgBvYPh6cX3aDP7DHKVbnCbi/go-ipfs-cmds"
	mh "gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	balanced "gx/ipfs/QmQjEpRiwVvtowhq69dAtB4jhioPVFXiCcWZm9Sfgn7eqc/go-unixfs/importer/balanced"
	ihelper "gx/ipfs/QmQjEpRiwVvtowhq69dAtB4jhioPVFXiCcWZm9Sfgn7eqc/go-unixfs/importer/helpers"
	trickle "gx/ipfs/QmQjEpRiwVvtowhq69dAtB4jhioPVFXiCcWZm9Sfgn7eqc/go-unixfs/importer/trickle"
	cmdkit "gx/ipfs/QmSP88ryZkHSRn1fnngAaV2Vcn63WUJzAavnRM9CVdU1Ky/go-ipfs-cmdkit"
	chunk "gx/ipfs/QmXzBbJo2sLf3uwjNTeoWYiJV7CjAhkiA4twtLvwJSSNdK/go-ipfs-chunker"
	cid "gx/ipfs/QmZFbDTY9jfSBms2MchvYM9oYRbAF19K7Pby47yDBfpPrb/go-cid"
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

The file is not pinned, so this command should be followed by an 'ipfs
pin add'.

This command is considered temporary until a better solution can be
found.  It may disappear or the semantics can change at any
time.
`,
	},
	Options: []cmdkit.Option{
		cmdkit.BoolOption(trickleOptionName, "t", "Use trickle-dag format for dag generation."),
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("url", true, false, "URL to add to IPFS"),
	},
	Type: BlockStat{},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) {
		url := req.Arguments[0]
		n, err := cmdenv.GetNode(env)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		if !filestore.IsURL(url) {
			res.SetError(fmt.Errorf("unsupported url syntax: %s", url), cmdkit.ErrNormal)
			return
		}

		cfg, err := n.Repo.Config()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		if !cfg.Experimental.UrlstoreEnabled {
			res.SetError(filestore.ErrUrlstoreNotEnabled, cmdkit.ErrNormal)
			return
		}

		useTrickledag, _ := req.Options[trickleOptionName].(bool)

		hreq, err := http.NewRequest("GET", url, nil)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		hres, err := http.DefaultClient.Do(hreq)
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}
		if hres.StatusCode != http.StatusOK {
			res.SetError(fmt.Errorf("expected code 200, got: %d", hres.StatusCode), cmdkit.ErrNormal)
			return
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
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		cmds.EmitOnce(res, BlockStat{
			Key:  root.Cid().String(),
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
