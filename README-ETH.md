# ETH side: Core Components & Actions

### sim_ether_man

- has: a full simulated eth chain instance as backend.
- has: eth accounts + eth coins.
- has: a schnorr signer, the pub_key is used in smart contract creation, signer itself is used in signing txs.
- did: deployed smart contracts (once) to the internal simulated eth chain.
- has: an "Etherman" instance. this operates on the eth chain.

*Notice*

1) priv keys are predefined, and then **imported to** sim_ether_man.
2) On creation of simulated eth chain, the "bind.TransactOpts" accounts (entities able to sign txs) are created from priv keys && eth are pre-allocated to these accounts at the same time (function as genesis).


do: use "Etherman" instance to Mint(), Approve(TWBTC), Trigger RedeemRequest(), Trigger RedeemPrepare().

Technically, this sim_ether_man is a wrapper.
In the end-to-end test, it actively does:

- [user action] `Approve(TWBTC)` which approves the usage of TWBTC token (first step of redeem, allow bridge to use twbtc from user), then call `Etherman.Approve()` underhood.
- [user action] `GetAuth(idx)` to get one EOA entity (an eth user that can sign tx), this entity will sign the request tx later as a user.
- [user action] `Request2()` to call `Etherman.RedeemRequest()` underhood (send the tx as user, to really kickstart the redeem process).

In summary, in the end-to-end test, the sim_ether_man only does the user actions.

Under the the hood, it also prepares params for the above functions.

- [server action] sim_ether_man exposed `Mint()` function uses `GenMintParams()`, which uses a schnorr signer to sign [btctxid, amount, receiver], the signature is provided to the bridge's `mint()` smart contract call and is VERIFIED by the smart contract. The smart contract call local go binding is `Etherman.Mint()`. This `Mint()` function is however **NOT** called in the end-to-end test. Because only bridge can do mint operation. It is implicitly handled in a similar manner by the ethtxmanager in a monitor loop in a real life function (see below).
- [server action] In real life, the `ethtxmanager.mint()` (see mint.go) is using a async way to request for signature. Then call the `Etherman.Mint()` smart contract method on chain and provide such signature as a part of the params.
- [server action] the sim_ether_man exposed `Prepare()` function uses `GenPrepareParams()`, which uses schnorr signer to sign [requestTxID, requester, Amount, Receiver, outputTxids, outputIds], of course the btc outputs (UTXOs) selected are pure random. The signature is also provided to the smart contract call and is VERIFIED by smart contract method. However, `Prepare()` is **NOT** used in the end-to-end test. The reason is similar, only the bridge invokes this method on chain. So it is implicitly called  by the ethtxmanager in a monitor loop in a real life situation (see below).
- [server action] In real life, the `ethtxmanager.handleRedeemPrepareTx()` calls smart contract `etherman.RedeemPrepare()` method (See prepare.go). It is triggered automatically by capturing the event log of RedeemRequest, in a public function `prepareRedeem()`. Similarily, it asks for a btc wallet to yield some usable UTXOs, and call the schnorr sign service to generate a signature as part of the parameters that is provided to the smart contract call.

So, although sim_ether_man has `Mint()` and `Prepare()` as public exposed functions, these two functions were never explicitly called by the end-to-end test. Because ethtxmanager is doing these two jobs in a separate go routine using a different yet similar set of code.

### state_db (pure db)

- is: sqlite db == file

### state

- has: state_db
- has: evm_chain_id

- do: operates on statedb.

### eth_tx_manager_db (pure db)

- is: sqlite db == file

### eth_synchronizer (read info from evm chain, change state)

- has: an instance of "Etherman"
- has: state
- has: evm_chain_id

- do: use "Etherman" to scan for 1) finalized blocks, 2) minted, requested, prepared event logs.
- do: operates on "state", when event logs are captured, push info into state.

### eth_tx_manager (operates on evm chain, send tx)

- has: an instance of "Etherman"
- has: state_db
- has: eth_tx_manager_db
- has: Schnorr signer (for sign `mint()` and `prepare()`)
- has: BTC wallet (for query UTXOs before `prepare()`)

- do: use "Etherman" to find out "isMinted", "IsPrepared" events. Do dirty jobs like "Mint", "RedeemPrepare".
- do: use "schnorr signer" to sign Mint Tx & Prepare Tx as bridge.
- do: use "btc wallet" to query and lock some BTC UTXO (redeem process: prepare stage).
- do: use "eth_tx_manager_db" to track monitored tx.
- do: use "state_db" to find "not minted yet" db records, "user requested but not prepared" db records.


## Summary:
1) the deployment of smart contracts is BEFORE the eth side core components are running.
2) the ETH and accounts are already there.
3) Etherman (regardless of simulated or not), is the component to interact with the evm chain.
4) Etherman is used by synchronizer and tx manager. (also by sim_ether_man wrapper).


# Rewrite: simEtherman -> realEtherman

Before launch:

- [x] (external) Geth chain.
- [x] get chain id from chain.
- [x] (external) ETH accounts + money.
- [x] bridge controlled ETH accounts' priv + address
- [x] (external) Schnorr signer (either local or remote).
- [x] Schnorr signer => fetch pub_key.
- [x] (once) Deploy smart contracts (with pub_key), this isn't dont via Etherman, you should hand-craft it, after deployment, you should have TWBTC contract Address + Bridge Contract Address.
- [x] Bridge contract address + TWBTC address.

Launch:

- Create an "Etherman" instance (Ethereum RPC client + one Ethereum account controlled by bridge + bridge smart contract address + TWBTC contract address).
- state_db (file)
- state (controls state_db)
- eth_tx_manager_db (file)
- eth_synchronizer (Etherman instance + state)
- eth_tx_manager (Etherman instance + state_db + eth_tx_manager_db + Schnorr signer + BTC wallet)

Roles (separate entities):

Bridge = 
- One ETH account
- Etherman
- eth_synchrorizer
- eth_tx_manager
- db + state

+ BTC_SIDE components.
+ REPORTER (HTTP server + API)

... run in a standalone binary.

ETH User =
- One ETH account
- Etherman (user's private key)
- Call Approve() in redeem...
- Call Request() in redeem...
- Call BalanceOf() in TWBTC...

... run in an interactive shell.
