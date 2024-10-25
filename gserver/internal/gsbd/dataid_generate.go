//go:build ignore

// Generate a constant for the zero hash.
// We could have copied these values as literal declarations,
// but I would argue this is a bit more provable and typo-proof.
// Furthermore, doing this in a go:generate step gives us the provability
// without incurring a runtime cost of executing the hash during
// an init function, for instance.

package main

import (
	"encoding/hex"
	"os"

	"golang.org/x/crypto/blake2b"
)

// The txsHashSize constant in gsi is unexported,
// so we are just redeclaring it here.
const txsHashSize = 32

func main() {
	hasher, err := blake2b.New(txsHashSize, nil)
	if err != nil {
		panic(err)
	}

	zeroHash := make([]byte, 0, txsHashSize)
	zeroHash = hasher.Sum(zeroHash)

	program := `package gsbd

// Code generated by 'go run dataid_generate.go'; DO NOT EDIT.

// The hash of no input, indicating that no transactions were included.
const zeroHash = "` + hex.EncodeToString(zeroHash) + `"

// The suffix of the app data ID including nTxs=0,
// data_len=0, and the corresponding hash.
const zeroHashSuffix = ":0:0:" + zeroHash
`

	if err := os.WriteFile("dataid_zero.go", []byte(program), 0o644); err != nil {
		panic(err)
	}
}
