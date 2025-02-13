package etherman_test

import (
	"testing"

	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/multisig"
)

// Bridge controlled core account, shall have some eth money already.
const CORE_ACCOUNT_ADDR = "0x85b427C84731bC077BA5A365771D2b64c5250Ac8"
const CORE_ACCOUNT_PRIV = "dbcec79f3490a6d5d162ca2064661b85c40c93672968bfbd906b952e38c3e8de"

func TestConnect(t *testing.T) {
	// 1) prepare the ethereum rpc to connect to
	url := "http://localhost:8545"

	// 2) prepare the schnorr signer
	ss, err := multisig.NewRandomLocalSchnorrSigner()
	if err != nil {
		t.Fatalf("failed to create schnorr wallet: %v", err)
	}

	// 3) prepare the core eth account controlled by the bridge
	core_account, err := etherman.StringToPrivateKey(CORE_ACCOUNT_PRIV)
	if err != nil {
		t.Fatalf("failed to create core eth account controlled by bridge: %v", err)
	}

	// 4) Create the real ethereum chain that is terraformed.
	realEth, err := etherman.NewRealEthChain(url, core_account, ss)
	if err != nil {
		t.Fatalf("failed to create real eth chain: %v", err)
	}
	t.Logf("Connected to the ethereum chain: %v", realEth.ChainId.Int64())
	t.Log("Bridge smart contract deployed")
	t.Logf("Bridge controlled eth account: %v", realEth.CoreAccount.From.Hex())
	// log the bridge contract address and twbtc contract address
	t.Logf("Bridge contract address: %v", realEth.BridgeContractAddress.Hex())
	t.Logf("Twbtc contract address: %v", realEth.TwbtcContractAddress.Hex())
}
