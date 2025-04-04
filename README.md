# A BTC-Aptos token bridge built on TEENet
This repository hosts a BTC-Aptos token bridge, developed for the Aptos EVERMORE Hackerhouse 2025.

## About TEENet
[TEENet](https://teenet.io) is a next-generation infrastructure for decentralized applications (dApps), built on the latest Trusted Execution Environment (TEE) technology. It is designed to eliminate security risks introduced by human involvement in the operation of dApps. TEENet provides:
A built-in wallet service for secure private key management.
A TEE-based environment for deploying and running dApps securely.

## About the bridge
This token bridge was developed as a demo for the hackathon event and runs on a local Bitcoin Regtest node and the Aptos Devnet. The bridge is designed to be stateless, relying solely on on-chain data for decision-making rather than any off-chain or locally stored information. 

### Aptos smart contracts
Two smart contracts have been developed and deployed on the Aptos Devnet:

* Wrapped BTC Token Contract
  This contract represents the wrapped BTC token on Aptos. It includes `mint` and `burn` functions, both of which can only be called by the bridge admin account hosted by the bridge backend.
* Bridge Contract
  This contract handles the bridging logic on Aptos and includes the following key methods:
  * `mint`: Called by the bridge backend to issue wrapped BTC tokens to the user after detecting a BTC deposit.
  * `redeemRequest`: Called by users to initiate a BTC withdrawal from Aptos.
  * `redeemPrepare`: Called by the bridge backend to lock UTXOs and ensure the withdrawal process is secure and double-spend resistant.

## Installation

## How to use

## Disclaimer
This project is provided "as is" without warranty of any kind, express or implied. The authors and contributors are not liable for any damages or losses arising from the use of this software. Use at your own risk. This repository is open-source and maintained by volunteers. There is no guarantee of support, updates, or continued maintenance. Any use of this code is subject to the terms of the LICENSE file included in this repository.
