// example code to demo the usage of btc_monitor
package btcsync

// 1. create a monitor
// 2. register observers into monitor
// 3. start observers' "get notified" goroutines
// 4. start monitor

// deposit, we need at least two obsevers:
// 1. deposit action storage
// 2. utxo storage in btc vault
