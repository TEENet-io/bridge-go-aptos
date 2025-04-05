# A BTC-Aptos token bridge built on TEENet
This repository hosts a BTC-Aptos token bridge, developed for the Aptos EVERMORE Hackerhouse 2025.

[DEMO VIDOE](https://drive.google.com/file/d/11CrX4p3qFoZ-3wmhb-0xQbMv0rZzUvGm/view?usp=sharing)

## About TEENet
[TEENet](https://teenet.io) is a next-generation infrastructure for decentralized applications (dApps), built on the latest Trusted Execution Environment (TEE) technology. It is designed to eliminate security risks introduced by human involvement in the operation of dApps. TEENet provides:
A built-in wallet service for secure private key management.
A TEE-based environment for deploying and running dApps securely.

## About the bridge
This token bridge was developed as a demo for the hackathon event and runs on a local Bitcoin Regtest node and the Aptos Devnet. The bridge is designed to be stateless, relying solely on on-chain data for decision-making rather than any off-chain or locally stored information. 

### Aptos smart contracts
Two smart contracts have been developed and deployed on the Aptos Devnet:

* [Wrapped BTC Token Contract](https://github.com/laalaguer/bridge-go-aptos/blob/main/aptos_contract/contract/sources/btc_token.move)
  This contract represents the wrapped BTC token on Aptos. It includes `mint` and `burn` functions, both of which can only be called by the bridge admin account hosted by the bridge backend.
* [Bridge Contract](https://github.com/laalaguer/bridge-go-aptos/blob/main/aptos_contract/contract/sources/btc_bridge.move)
  This contract handles the bridging logic on Aptos and includes the following key methods:
  * `mint`: Called by the bridge backend to issue wrapped BTC tokens to the user after detecting a BTC deposit.
  * `redeemRequest`: Called by users to initiate a BTC withdrawal from Aptos.
  * `redeemPrepare`: Called by the bridge backend to lock UTXOs and ensure the withdrawal process is secure and double-spend resistant.

## Installation and Configuration

### Prerequisites
- Aptos CLI Tool (Installation guide available at: https://aptos.dev/en/build/cli)
- Go environment (for testing)

### Account Configuration
1. Initialize an Aptos account using the CLI:
   ```bash
   aptos init
   ```

2. Retrieve your account credentials:
   ```bash
   cat .aptos/config.yaml
   ```

### Smart Contract Deployment
1. Configure the contract deployment settings:
   - Navigate to `aptoscontract/Move.toml`
   - Update the public address parameter with your account address

2. Compile and deploy the smart contracts:
   ```bash
   aptos move compile --named-addresses my_address=<your_address>
   aptos move publish --named-addresses my_address=<your_address>
   ```

### Integration Testing
1. Update the test configuration:
   - Locate `bridge-go-aptos/cmd/demo_test_cmd/integration_test.go`
   - Replace the placeholder private and public keys with your account credentials

2. Execute the integration tests:
   ```bash
   go test -v
   ```

## How to use
### First
You need to apply an Aptos account to deploy smart contrast. Try to install Aptos CLI Tool:https://aptos.dev/en/build/cli
After that, you can use this command to create an aptos account
`aptos init`

Then, you can see your private key and public key
`cat .aptos/config.yaml`

Try to replace public address in `aptoscontract/Move.toml` to your account

Then, 

```
aptos move compile --named-addresses my_address=your_address
aptos move publish --named-addresses my_address=
```

Try to replace privatekey and publickey in `bridge-go-aptos/cmd/demo_test_cmd/integration_test.go`

Then, you can run this command to test
`go test -v`

## Disclaimer
This project is provided "as is" without warranty of any kind, express or implied. The authors and contributors are not liable for any damages or losses arising from the use of this software. Use at your own risk. This repository is open-source and maintained by volunteers. There is no guarantee of support, updates, or continued maintenance. Any use of this code is subject to the terms of the LICENSE file included in this repository.
