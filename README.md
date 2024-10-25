# gcosmos

This is a temporary workspace for integrating [Gordian](https://github.com/gordian-engine/gordian)
with [the Cosmos SDK](https://github.com/cosmos/cosmos-sdk).

## Setup

We are currently using a local, patched clone of the SDK in tandem with Go workspaces.
It is a bit unconventional to commit a Go workspace file, but while both Gordian and the Cosmos SDK
are being actively changed, a fixed Go workspace fits well for now.

From the root directory, run `./_cosmosvendor/sync_sdk.bash` to clone or fetch and checkout
a "known working" version of the Cosmos SDK compatible with the current gcosmos tree,
and then apply any currently necessary patches to the SDK.
You may need to run `go work sync` from the `gcosmos` directory again.

### New patches

New patches to the SDK should build upon the existing patches,
so long as the existing patches are necessary.

To continue adding patches on top of the existing ones,
the simplest workflow is:

1. Ensure you are already synced via the `sync_sdk.bash` script.
2. Ensure you have the latest SDK commit, typically via `git fetch` inside the `_cosmosvendor/cosmos-sdk` directory.
3. Rebase the existing patch set onto the latest commit, typically with `git rebase origin/main`. Address conflicts as needed.
4. Optionally commit new code to your SDK checkout.
4. From your new rebased set of patches, within the `_cosmosvendor/cosmos-sdk` directory,
   run `git format-patch -o ../patches origin/main` to overwrite the existing set of patches with a new set that no longer has conflicts.
5. Be sure to update `_cosmosvendor/COSMOS_SDK.commit`.
6. Double check that running `_cosmosvendor/sync_sdk.bash` still results in passing tests.

The set of SDK patches is minimal for now, and our goal is to depend on a tagged release of the SDK
instead of having to maintain a patched version.

## Running

Begin running the updated simapp commands from the `gcosmos` directory.

```bash
sh ./scripts/run_gcosmos.sh
```

### Interact
```bash
# Install the grpcurl binary in your relative directory to interact with the GRPC server.
# GOBIN="$PWD" go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1

./grpcurl -plaintext localhost:9092 list
./grpcurl -plaintext localhost:9092 gordian.server.v1.GordianGRPC/GetBlocksWatermark
./grpcurl -plaintext localhost:9092 gordian.server.v1.GordianGRPC/GetValidators

./grpcurl -plaintext -d '{"address":"cosmos1r5v5srda7xfth3hn2s26txvrcrntldjumt8mhl","denom":"stake"}' localhost:9092 gordian.server.v1.GordianGRPC/QueryAccountBalance
```

### Transaction Testing
```bash
./gcosmos tx bank send val cosmos10r39fueph9fq7a6lgswu4zdsg8t3gxlqvvvyvn 1stake --chain-id=localchain-1 --generate-only > example-tx.json

# TODO: get account number
./gcosmos tx sign ./example-tx.json --offline --from=val --sequence=1 --account-number=0 --chain-id=localchain-1 --keyring-backend=test > example-tx-signed.json

./grpcurl -plaintext -emit-defaults -d '{"tx":"'$(cat example-tx-signed.json | base64 | tr -d '\n')'"}' localhost:9092 gordian.server.v1.GordianGRPC/SimulateTransaction

./grpcurl -plaintext -emit-defaults -d '{"tx":"'$(cat example-tx-signed.json | base64 | tr -d '\n')'"}' localhost:9092 gordian.server.v1.GordianGRPC/SubmitTransaction

./grpcurl -plaintext -d '{"address":"cosmos10r39fueph9fq7a6lgswu4zdsg8t3gxlqvvvyvn","denom":"stake"}' localhost:9092 gordian.server.v1.GordianGRPC/QueryAccountBalance
```

## Hacking on a demo with four validators

We have a current hack that uses a test setup with four validators,
and then keeps them alive so that you can temporarily interact with the network.

From the root of this repository, run:
`HACK_TEST_DEMO=1 go test . -v -run=TestTx_multiple_simpleSend -timeout=5m`

Which will run the test and then print out details like:
```
main_test.go:1195: >>>>>>>>>>>>> Test will run until 2024-10-25T13:36:32-04:00 (38.95884875s)
main_test.go:1196: >>>>>>>>>>>>> (pass flag -timeout=0 to run forever)
main_test.go:1201: VVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV
main_test.go:1202: >            CONNECTION INFO           <
main_test.go:1204: Validator 0:
main_test.go:1205: 	HTTP address: http://127.0.0.1:51758
main_test.go:1206: 	Home directory: /tmp/TestTx_multiple_simpleSend2865145828/001
main_test.go:1204: Validator 1:
main_test.go:1205: 	HTTP address: http://127.0.0.1:51760
main_test.go:1206: 	Home directory: /tmp/TestTx_multiple_simpleSend2865145828/002
main_test.go:1204: Validator 2:
main_test.go:1205: 	HTTP address: http://127.0.0.1:51759
main_test.go:1206: 	Home directory: /tmp/TestTx_multiple_simpleSend2865145828/003
main_test.go:1204: Validator 3:
main_test.go:1205: 	HTTP address: http://127.0.0.1:51761
main_test.go:1206: 	Home directory: /tmp/TestTx_multiple_simpleSend2865145828/004
main_test.go:1208: ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
```

From there, you can interact with the HTTP API of each validator, or inspect their home directories as desired.

Useful HTTP interactions include:

Check the validator's view of the voting and committing heights:
```shell
$ curl "$ADDR/blocks/watermark"
{"VotingHeight":4,"VotingRound":4,"CommittingHeight":3,"CommittingRound":0}
```

See the current validator set:
```shell
$ curl -s http://127.0.0.1:53779/validators | jq
{
  "FinalizationHeight": 3,
  "Validators": [
    {
      "PubKey": "ZWQyNTUxOQACP0CD6CdN4xH9mUgwIR5ntLO0DgUkT5iASs8vHTpA8A==",
      "Power": 100000003
    },
    {
      "PubKey": "ZWQyNTUxOQCQjWqgssuCmzDdkK15aPb3xQ2iMtTNvC3u1F9pu3Wctw==",
      "Power": 100000002
    },
    ...
```

Refer to the `http*.go` files in [the gserver/internal/gsi directory](gserver/internal/gsi/) for more details on available HTTP paths.
