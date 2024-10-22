package utils

import "strings"

// Remove0xPrefix removes the "0x" prefix from a string if it exists
// It is helpful because BTC address/txid don't have the "0x" prefix.
func Remove0xPrefix(input string) string {
	if strings.HasPrefix(input, "0x") {
		return input[2:]
	}
	return input
}

// Add0xPrefix adds the "0x" prefix to a string if it doesn't already have it
func Add0xPrefix(input string) string {
	if strings.HasPrefix(input, "0x") {
		return input
	}
	return "0x" + input
}
