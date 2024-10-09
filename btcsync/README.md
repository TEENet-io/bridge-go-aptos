`btcsync` is responsible for:

1) Monitor BTC blockchain and discover interested actions.
    - RPC to get blocks/TXs/UTXOs
2) Pub-sub to push notification to different serivces.
    - EVM mint service.
    - action storage service.
3) If something goes wrong do the Refund action.
    - Do BTC refund tx.
    - do action storage service.

# Monitor the chain

A constant running monitor is scrubbing the bitcoin blockchain. It filter out each block and each tx. Then once an interested action is found in the Tx. It triggeres registered observers to process the action.