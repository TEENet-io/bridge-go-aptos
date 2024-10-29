### `btc` module

This module contains btc related operations.

Package definition see respective `README.md` files in each folder.

# Notes

## Test
Use a MODIFIED `bitcoin-testnet-box` as local testing.

## Regtest bitcoin core node Configuration

localhost
19001
admin1
123

### Embedded default wallets
wallet 1
P2WPKH  bcrt1qa3ma47jt8mdqq699vv2f0f0ahpp66f2tj0pa0f
PRIV    cRXkPMW52JPErML1Bgg5dTMFx1G28HFRAdSBsFc7pRZPiRaXY7J7

wallet 2
P2WPKH  bcrt1q67j4rp78g9dnsdd0hv47vffycan0x8prlhg4yt
PRIV    cT7QVDDEmtNRhSSAhSuUm8PYud18eQbDKnt68839CcVXswy5HfRX

Tricks:

- Start bitcoin core with `-txindex` otherwise query Tx via TxID is a problem.
- Start bitcoin core with `-fallbackfee` a minimum relay fee.
- Use `bitcoin-cli importprivkey [priv_in_WIF_format] [label_name] [rescan:true]` to import a private key. This will keep track of the corresponding address.
- Use `bitcoin-cli importaddress "address" "label" [rescan:true]` to keep track on an address (but cannot spend the funds)

Query for balances/utxo/txs:

```bash
listbalance
listunspent
listtransactions
```

### Example - Create legacy address and dump its private key:

(Required after 0.17.0 version)

`bitcoin-cli -datadir=1 getnewaddress "label" "legacy"`
moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn

`bitcoin-cli -datadir=1 dumpprivkey "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn"`
cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY

mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT
cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH

### Example - Create Native SegWit address (Bech32) and dump its private key:

`bitcoin-cli -datadir=1 getnewaddress "" "bech32"`
bcrt1q8eqm6dwmt23k246f4fmkruwd5pjupqs7l0l3dl

`bitcoin-cli -datadir=1 dumpprivkey bcrt1q8eqm6dwmt23k246f4fmkruwd5pjupqs7l0l3dl`
cPKT92sLkVEVmrZ9ojjWxmtLvmhbuCnUmbmo93bVDRqdMpcqCAGZ

### Format of Bitcoin address and private key:
[link](https://github.com/citizen010/bitcoin-prefixes-address-list)

### Bitcoin WIF (Wallet Import Format) of private key

str = `Base58(<0x80><32-byte-private-key><0x01><4-byte-checksum>)`

About 0x80:

if mainnet, 0x80; if testnet3 or regtest, 0xef;

About 0x01:

if priv key is compressed, 0x01; if not 0x00;

on mainnet:
if final str begins with 5, uncompressed;
if final str begins with K or L, compressed;

on testnet:
if final str begins with 9, uncompressed;
if final str begins with c, compressed;

Bitcoin private key is always the same (random 256bit/32byte), regardless what type of address you derive from it.