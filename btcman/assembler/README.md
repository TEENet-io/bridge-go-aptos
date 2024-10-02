Assembler represents a single entity that can assemble the Tx.

It requires assembler able to do the following two jobs in *SEQUENCE*:

1. `lock`, which is creating locking scripts for designated receivers to redeem in the future.
2. `unlock`, which is creating suitable signatures to unlock the UTXOs previously received.

### Lock

This **doesn't require** any knowledge of private key. It just depends on two params: bitcoin receiver's address and the receiver's btc network parameter. Then the lock process produces locking scripts to be the outputs of Tx. In the future, the receiver produces correct signatures to unlock the outputs to spend the money.

### Unlock

This **requires** the assembler to produce some valid signatures to unlock UTXOs. So the assembler should somehow know the private key or be able to collect a valid signature from external services.

This however, depends on different implentations of legacy signer, segwit signer and m-to-n schnorr signer.

```bash
interfaces.go # Defines lock and unlock
legacy.go # A legacy signer (single private key) implementation
schnorr.go # A m-to-n schnorr signers' group implementation (aggregated), not implemented yet.
```