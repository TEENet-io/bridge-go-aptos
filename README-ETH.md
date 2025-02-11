### priv keys (import to) -> sim_ether_man

has: a full simulated eth chain instance as backend.
has: eth accounts + ETH coins.
has: a schnorr signer, the pub_key is used in smart contract creation, signer itself is used in signing txs.
did: deployed smart contracts (once) to the internal simulated eth chain.
has: an "Etherman" instance. this operates on the eth chain.

do: use "Etherman" to Mint(), Approve(TWBTC), Trigger RedeemRequest(), Trigger RedeemPrepare().

Technically, this sim_ether_man is a wrapper.
In the end-to-end test, it actively do:

- Approve(TWBTC) which approve the usage of TWBTC token (user action, first step of redeem, allow bridge to use twbtc from user), and call Etherman.Approve() underhood.
- GetAuth() to get EOA entity (an eth user that can sign tx).
- Request2() to call Etherman.RedeemRequest() underhood (user action, issue the tx as user, to really kickstart the redeem process).

Under the the hood, it also prepares params for the above functions.

- [server action] the sim exposed Mint() function uses GenMintParams(), which uses schnorr signer to sign [btctxid, amount, receiver], the signature is provided to the mint() smart contract call and is VERIFIED by smart contract. The smart contract call is Etherman.Mint(). This function is however NOT called in the end-to-end test. It is implicitly handled in a similar manner by the ethtxmanager in a monitor loop in a real life function (see below).
- [server action] In real life, the ethtxmanager.mint() (see mint.go) is using a async way to request for signature. Then call the Etherman.Mint() smart contract method on chain and provide such signature as a part of the params.
- [server action] the sim exposed Prepare() function uses GenPrepareParams(), which uses schnorr signer to sign [requestTxID, requester, Amount, Receiver, outputTxids, outputIds], of course the outputs are pure random. the signature is also provided to the smart contract call and is VERIFIED by smart contract method. However, it is NOT used in the end-to-end test. It is implicitly handled in a similar manner by the ethtxmanager in a monitor loop in a real life function (see below).
- [server action] In real life, the ethtxmanager.handleRedeemPrepareTx() also calls smart contract etherman.RedeemPrepare() method. See prepare.go. It is triggered automatically by capturing the event log of RedeemRequest, in a public function prepareRedeem(). Similarily, it asks for a btc wallet to yield some usable UTXOs, and call the schnorr signing service to generate a signature as part of the parameters that is provided to the smart contract call.

## ETH side: core components

### statedb (pure db)

-> sqldb -> file

### state

has: statedb
has: evm_chain_id

do: operates on statedb.

### eth_tx_manager_db (pure db)

-> sqldb -> file

### eth_synchronizer (read info from evm chain, change state)

has: an instance of "Etherman"
has: state
has: evm_chain_id

do: use "Etherman" to scan for 1) finalized blocks, 2) minted, requested, prepared event logs.
do: operates on "state", when each type of event log is captured.

### eth_tx_manager (operates on evm chain, send tx)

has: an instance of "Etherman"
has: state_db
has: eth_tx_manager_db
has: Schnorr wallet (for sign mint and prepare)
has: BTC wallet (for query UTXOs for prepare)

do: use "Etherman" to find out "isMinted", "IsPrepared". Do dirty jobs like "Mint", "RedeemPrepare".
do: use "schnorrwallet" to sign Mint Tx & Prepare Tx.
do: use "btcwallet" to query and lock some BTC UTXO (redeem process: prepare stage).
do: use "eth_tx_manager_db" to track monitored tx.
do: use "state_db" to find "not minted yet" db records, "user requested but not prepared" db records.


So techniqually:
1) the deployment of smart contracts is BEFORE the eth side core components are running.
2) the ETH and accounts are already there.
3) Etherman (regardless of simulated or not), is the component to interact with the evm chain.
4) Etherman is used by synchronizer and tx manager. (also by sim_ether_man wrapper).


# Rewrite: simEtherman -> realEtherman

Prepare:
- (external) Geth chain.
- (external) ETH accounts + money.
- ETH accounts' priv + address
- (external) Schnorr signer.
- Schnorr signer => fetch pub_key.
- (once) Deploy smart contracts (with pub_key).
- Bridge contract address + TWBTC address.

Launch:

- Create an Etherman instance (Ethereum RPC client + one Ethereum account controlled by bridge + bridge smart contract address + TWBTC contract address).
- state_db (file)
- state (controls state_db)
- eth_tx_manager_db (file)
- eth_synchronizer (put Etherman instance in, put state in)
- eth_tx_manager (put Etherman instance in, put state_db in, put eth_tx_manager_db in, put Schnorr signer in, put BTC wallet in)

Roles:

Bridge = 
- Etherman
- eth_synchrorizer
- eth_tx_manager
- db + state

+ BTC_SIDE
+ REPORTER (HTTP SERVER + API)

... run in a standalone binary.

ETH User =
- ETH account
- Etherman (different private key)
- Call Request()
- Call Approve()
- Call BalanceOf()

... run in an interactive shell.
