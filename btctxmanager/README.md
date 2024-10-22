# BTC withdraw

- It loops and check the state/statedb to find records of Redeem that needs to be withdrawed on BTC side.
- It withdraws real BTC.
- It creates a BTC withdraw action and keeps monitoring this action.
- Once the withdraw is mined, it publishes to the observer (then in turn publishes to state/statedb)