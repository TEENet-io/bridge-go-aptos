Assembler represents a single entity that assembles the Tx.

### Intro of Address Types

`P2PK` - The original address type used in Bitcoin. It is a public key that can be used to create a signature for a transaction. But then people find it too long and doesn't save block space. So start to think of new type of address.

`P2PKH` - Commonly used address type. The sucessor of `P2PK`. It is the hash of public key. So shorter. Its been used primarily for a long time until Segwit upgrade occurs.

`P2WPKH` - address type associated with Segwit upgrade. It is a new address type (upgrade of P2PKH) that is more efficient and private than the current P2PKH and P2PK addresses. 

`P2TR` - address type associated with Taproot upgrade. It is a single address that can be used for all types of transactions, including single-sig, multi-sig, and complex scripts.

### Signature and Address Types

Over a long time the `ECDSA signature` has been used in Bitcoin. It uses elliptic curve. Before the Taproot upgrade, the `Schnorr signature` is not suppported natively. So addresses of type `P2PKH` and `P2WPKH` are using `ECDSA signature`. After the Taproot upgrade, the `Schnorr signature` is supported natively. So addresses of type `P2TR` are using `Schnorr signature`.

### Transaction Forming
It requires tx assembler able to do the following two jobs in *SEQUENCE*:

1. `lock`, which is creating locking scripts for designated receivers to redeem in the future.
2. `unlock`, which is creating suitable signatures to unlock the UTXOs previously received.

Do the locking first, then do the unlocking. Otherwise the Tx will be invalid.

### Lock

This **doesn't require** any knowledge of private key or signature. It just depends on two params: bitcoin receiver's address and the receiver's btc network parameter. Then the lock process produces locking scripts to be the outputs of Tx. In the future, the receiver produces correct signatures to unlock the outputs to spend the money.

### Unlock

This **requires** the assembler to produce some valid signatures to unlock UTXOs. So the assembler should somehow know the private key or be able to collect a valid signature from external services.

This however, depends on different implentations of legacy signer, segwit signer and m-to-n schnorr signer.

### Code Design

`Signer`, whether backed by local or remote, the core function is can provide a public key (for signature verification) and provide a signing function (to produce signature).

`Operator`, can perform `unlock` interface defined actions. It is the key step to spend previously received UTXOS, it requries you to have either a private key to sign and produce a valid signature, or a remote service to provide a valid signature.

`Assembler`, main entity to craft a proper business logic over Tx. Like for example, simple transfer Tx or complex Tx that includes data script. It uses `lock` interface to create locking scripts  and `unlock` to produce valid signatures (require operator) under the hood.

### Files
```bash
interfaces.go # Defines Operator interface (includes lock and unlock).
native_operator.go # Implements the Operator interface (local single-private key signer).
schnorr_operator.go # Implements the Operator interface (local/remote schnorr signer).
assembler.go # Uses Operator to perform busines logic (assemble a proper BTC Tx).
```