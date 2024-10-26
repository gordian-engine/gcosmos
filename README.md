# gcosmos

This repository integrates [Gordian](https://github.com/gordian-engine/gordian)
with [the Cosmos SDK](https://github.com/cosmos/cosmos-sdk).

## Quickstart

Start the example Gordian-backed Cosmos SDK chain with one validator

```bash
CHAIN_ID=gchain-1 make start
```

### Interact

In a second terminal, interact with the running chain

Show validator address

```bash
VAL_ADDR=$(gcosmos keys show val --address)
echo $VAL_ADDR
```

Query bank balance of validator, it has `9000000stake`
```bash
gcosmos q bank balance $VAL_ADDR stake
```

### Transaction Testing

Send `100stake` from the validator to a new account.

```bash
gcosmos --chain-id gchain-1 tx bank send val cosmos10r39fueph9fq7a6lgswu4zdsg8t3gxlqvvvyvn 100stake
```

Confirm the balance in the new account, it now has `100stake`

```bash
gcosmos q bank balance cosmos10r39fueph9fq7a6lgswu4zdsg8t3gxlqvvvyvn stake
```

## Multiple validator example

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
