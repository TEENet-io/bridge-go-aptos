Defines what a "synchronizer" of a ETH/APTOS/other chain should do.

According to design document, a chain synchronizer shall:

1. Monitor blockchain and capture `Mint`/`RedeemRequest`/`RedeepPrepare` Event.
2. Use an instance of `state`, and notify `state` about the events.

What you can do

Implement `SyncWorker` interface.