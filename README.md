# gcosmos

This repository integrates [Gordian](https://github.com/gordian-engine/gordian)
with [the Cosmos SDK](https://github.com/cosmos/cosmos-sdk).

Refer to the [`_docs` directory](/docs)
for more technical details about how gcosmos works.

## Quickstart

Start the example Gordian-backed Cosmos SDK chain with four validators

```bash
make testnet-start
```

### Interact

In a second terminal, interact with the running chain

Show validator address

```bash
VAL_ADDR=$(./gcosmos --home ./test/val1 keys show val --address)
echo $VAL_ADDR
```

Query bank balance of validator, it has `9000000stake`.
```bash
./gcosmos q bank balance $VAL_ADDR stake
```

### Transaction Testing

Send `100stake` from the first validator to a new account.

```bash
./gcosmos --home ./test/val1 --chain-id gchain-1 tx bank send val cosmos10r39fueph9fq7a6lgswu4zdsg8t3gxlqvvvyvn 100stake
```

Wait a few seconds for the block to be produced, then confirm the balance in the new account. It now has `100stake`.

```bash
./gcosmos q bank balance cosmos10r39fueph9fq7a6lgswu4zdsg8t3gxlqvvvyvn stake
```

### Shutdown

Shutdown the testnet by pressing Ctrl+C in the first terminal.

## Additional Information

Refer to the `http*.go` files in [the gserver/internal/gsi directory](gserver/internal/gsi/) for more details on available HTTP paths.
