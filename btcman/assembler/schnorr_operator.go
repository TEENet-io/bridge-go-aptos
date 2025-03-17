// Implements the interface of Operator
// 1) Uses a local/remote schnorr signature service as backbone.
// 2) Provides public key and signatures, but NOT private key.

package assembler

import (
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/TEENet-io/bridge-go/multisig_client"
)

type SchnorrOperator struct {
	ChainConfig *chaincfg.Params // which BTC chain it is on. (mainnet, testnet, regtest)
	mySigner    multisig_client.SchnorrSigner
	P2TR        *btcutil.AddressTaproot
}

func NewSchnorrOperator(signer multisig_client.SchnorrSigner, chainConfig *chaincfg.Params) (*SchnorrOperator, error) {
	// Convert Public Key to a P2TR address
	pk, err := signer.Pub()
	if err != nil {
		return nil, err
	}
	// from github gist.
	// https://github.com/babylonlabs-io/babylon/blob/main/crypto/bip322/bip322.go#L175
	// https://github.com/GoudanWoo/note-minter/blob/7068cfd21ddaa5dce6a42d528facead5ab3ed7c0/utils.go#L28
	// https://github.com/nayuta-ueno/taproot-redeem-go/blob/9cf43258ecb756d6eadba6096fa797acc740beb5/tx/p2trkey.go#L24
	// https://github.com/b-harvest/babylon/blob/d4c52f96bd2caca302c054ccc3814fa17bd40749/crypto/bip322/bip322.go#L188
	tapKey := txscript.ComputeTaprootKeyNoScript(pk)
	p2trAddr, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(tapKey), chainConfig)

	// from github gist.
	// p2trAddr, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(pk), chainConfig)

	// Below line is from chatgpt, which doesn' pass the test!
	// p2trAddr, err := btcutil.NewAddressTaproot(pk.SerializeCompressed(), chainConfig)
	if err != nil {
		return nil, err
	}
	return &SchnorrOperator{chainConfig, signer, p2trAddr}, nil
}

func (rso *SchnorrOperator) AppendPayToAddress(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error) {
	// Use a common function that already implemented.
	return AppendPayToAddress(tx, dst_chain_cfg, dst_addr, amount)
}

// Unlock operation generates the tx's inputs secion,
// with every previous output, make a SignatureScript (to unlock it).
// Warning: You should generate Locking Scripts (outputs) firstly on tx,
// then call this function to generate the inputs.
func (rso *SchnorrOperator) Unlock(tx *wire.MsgTx, prevOutputs []*utxo.UTXO) (*wire.MsgTx, error) {
	// Trick:
	// Both tx.TxIn[] and tx.TxOut[] shall be ready, then you can create sign scripts.
	// If they are not ready the sign will create wrong signature (won't pass the validation of node)
	// In following step the signature script is filled with nil
	for _, item := range prevOutputs {
		txIn := wire.NewTxIn(wire.NewOutPoint(item.TxHash, item.Vout), nil, nil)
		tx.AddTxIn(txIn)
	}
	// In following step signature script is filled with real stuff.
	for idx, item := range prevOutputs {
		script, err := rso.SignatureScript(tx, idx, item.PkScript, txscript.SigHashAll, true)
		if err != nil {
			return nil, err
		}
		tx.TxIn[idx].SignatureScript = script
	}
	return tx, nil
}

// We leave out the "privKey *btcec.PrivateKey" param from the original implementation.
func (rso *SchnorrOperator) SignatureScript(tx *wire.MsgTx, idx int, subscript []byte, hashType txscript.SigHashType, compress bool) ([]byte, error) {
	sig, err := rso.RawTxInSignature(tx, idx, subscript, hashType)
	if err != nil {
		return nil, err
	}

	// Use Schnorr signer's service to replace the original logic of getting the public key.
	pk, err := rso.mySigner.Pub()
	if err != nil {
		return nil, err
	}
	// end of replacement of logic
	var pkData []byte
	if compress {
		pkData = pk.SerializeCompressed()
	} else {
		pkData = pk.SerializeUncompressed()
	}

	return txscript.NewScriptBuilder().AddData(sig).AddData(pkData).Script()
}

// RawTxInSignature returns the serialized ECDSA signature for the input idx of
// the given transaction, with hashType appended to it.
func (rso *SchnorrOperator) RawTxInSignature(tx *wire.MsgTx, idx int, subScript []byte,
	hashType txscript.SigHashType) ([]byte, error) {

	hash, err := txscript.CalcSignatureHash(subScript, hashType, tx, idx)
	if err != nil {
		return nil, err
	}
	// Use schnorr signature signer's service to replace the original logic of signing.
	signature, err := rso.mySigner.Sign(hash)
	if err != nil {
		return nil, err
	}
	// end of replacement of logic

	return append(signature.Serialize(), byte(hashType)), nil
}
