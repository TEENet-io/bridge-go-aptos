# Synchronizer's job

A synchronizer shall:

1. Monitor blockchain blocks, and capture `Mint`/`RedeemRequest`/`RedeepPrepare` Event.
2. Notify `state` database about the captured events.

# For Developers

Implement `SyncWorker` interface (see `interface.go`).

# Files

`syncer.go` - Main function body of Sychronizer.

`interface.go` - Interfaces of chain's worker. Shall implement those to work with Synchronizer.