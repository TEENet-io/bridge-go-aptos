BTCactions define the actions on the BTC side that we are interested in.

These actions will trigger the whole system to make state-transition.

These actions are read from BTC blockchain so is **very reliable**.

## Actions Interested

see `types.go`

**Deposit Actions**

```
Tx: {
inputs: we don't care actually
---------------------------
output #1 = {value} {to us}
output #2 = OP_RETURN={RLP(evm_id, evm_addr)} {to us}
output #3 = {value} {to change receiver}
}

```

**Other Transfers Actions** (money transfer-in other than deposit)
```
Tx: {
inputs: we don't care
------------------------
output #? = {value} {to us}
}
```

## Post Action Phase

Use pub-sub model to push notifications to various services once an action is found on-chain.

The message sender is publisher. The message receiver is observer.

see `btcsync/` package.

To ensure a non-blocking manner, the pub and sub use fat channels / separate go routines to convey the message.
