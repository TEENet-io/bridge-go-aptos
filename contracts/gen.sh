#!/bin/sh

set -e

gen() {
    local package=$1

    if [ ! -d "$package" ]; then
        # Directory does not exist, so create it
        mkdir -p "$package"
        echo "Directory $DIR_PATH created."
    fi

    abigen --bin bin/${package}.bin --abi abi/${package}.abi --pkg=${package} --out=${package}/${package}.go
}

genNoBin() {
    local package=$1

    if [ ! -d "$package" ]; then
        # Directory does not exist, so create it
        mkdir -p "$package"
        echo "Directory $DIR_PATH created."
    fi

    abigen --abi abi/${package}.abi --pkg=${package} --out=${package}/${package}.go
}

gen TEENetBtcBridge
gen TWBTC
