package commands

import (
	"testing"

	"github.com/ipfs/go-ipfs/namesys"
	tu "gx/ipfs/QmPdxCaVp4jZ9RbxqZADvKH6kiCR5jHvdR5f2ycjAY6T2a/go-testutil"
)

func TestKeyTranslation(t *testing.T) {
	pid := tu.RandPeerIDFatal(t)
	a, b := namesys.IpnsKeysForID(pid)

	pkk, err := escapeDhtKey("/pk/" + pid.Pretty())
	if err != nil {
		t.Fatal(err)
	}

	ipnsk, err := escapeDhtKey("/ipns/" + pid.Pretty())
	if err != nil {
		t.Fatal(err)
	}

	if pkk != a {
		t.Fatal("keys didnt match!")
	}

	if ipnsk != b {
		t.Fatal("keys didnt match!")
	}
}
