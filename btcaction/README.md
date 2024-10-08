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
**Withdraw Actions**

```
Tx: {
input #1 - #n {from us}
---------------------------
output #1 = {value} {to user}
output #2 = {value} {to us}
}
```

**Unknown Transfers Actions** (non-of-the-two actions above)
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
