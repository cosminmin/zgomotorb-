package commands

import (
	"errors"

	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
	cmds "gx/ipfs/QmfUm9hpamNWvkmeFNAakDKsDpvMx3tzUqMSsTEq7Ve4jD/go-ipfs-cmds"
)

var MountCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline:          "Not yet implemented on Windows.",
		ShortDescription: "Not yet implemented on Windows. :(",
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		return errors.New("Mount isn't compatible with Windows yet")
	},
}
