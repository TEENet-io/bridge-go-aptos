Assembler represents a single entity that assembles the Tx.

It requires assembler able to do the following two jobs in *SEQUENCE*:

1. `lock`, which is creating locking scripts for designated receivers to redeem in the future.
2. `unlock`, which is creating suitable signatures to unlock the UTXOs previously received.

### Lock

This **doesn't require** any knowledge of private key. It just depends on two params: bitcoin receiver's address and the receiver's btc network parameter. Then the lock process produces locking scripts to be the outputs of Tx. In the future, the receiver produces correct signatures to unlock the outputs to spend the money.

### Unlock

This **requires** the assembler to produce some valid signatures to unlock UTXOs. So the assembler should somehow know the private key or be able to collect a valid signature from external services.

This however, depends on different implentations of legacy signer, segwit signer and m-to-n schnorr signer.

### Files
```bash
interfaces.go # Defines lock and unlock
locker_impl.go # Implements the lock interface
legacy.go # A legacy assembler (single private key) implements unlock interface + useful functions
```

### Structure

`Signer`, whether backed by local or remote, the core function is can provide a public key (for signature verification) and provide a signing function (to produce signature).

`Operator`, can perform `unlock` interface defined actions. It is the key step to spend previously received UTXOS, it requries you to have either a private key to sign and produce a valid signature, or a remote service to provide a valid signature.

`Assembler`, main entity to craft a proper business logic Tx. It uses `lock` to create locking scripts and `unlock` to produce valid signatures (require operator).