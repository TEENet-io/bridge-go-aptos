# Multi-Sig Schnorr Wallet

Multi-sign schnorr wallet is a multi-party wallet that can perform like a single private key backed wallet.

# Functionality

- Sign(): Given a message, sign it and return the signature.
- GetPublicKey(): Return the public key of the wallet. (for signature verification)

# Types

- LocalWallet: Locally hosted wallet. Backed by only a single private key.
- RemoteWallet: A wallet that is M-N, an sign messages, and is remotely hosted on server.

# Files

`interface.go` - Interface for wallet.
`net.go` - Operator to communicate with remote server. Do `Sign()` and `GetPubKey()` network requests.

`remote.go` - Remote wallet implementation. Based on `net.go`.
`local.go` - Local wallet implementation.

`utils.go` - Converters of primitive data types.