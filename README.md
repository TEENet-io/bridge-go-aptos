# Mono repo, workspace

Welcome to the workspace of project `teenet.io/bridge-go`

## Project Structure
Go modules are hosted inside their folders, eg:

`btc/` folder = 
`teenet.io/bridge-go/btc` module.

Packages are hosted inside each sub-folder:

```
btc/
├── data
│   ├── // package, common data structure and definitions
├── rpc
│   ├── // package, rpc: send/read tx.
└── wallet
    ├── // package, wallet: assemble tx, wallet management.
```

## Useful commands

### `module` level
```bash
cd ./btc && go mod tidy # cleanup dependency

go test -v ./btc/data # Run tests of a package

cd ./new-folder && go mod init teenet.io/bridge-go/new-module-name # Create a new module
```

### `workspace` level
```bash
go work init # Create a workspace

go work sync # cleanup dependency

go work use ./btc # declare to us a newly-written module as dependency
```