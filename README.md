# gcosmos

This repository integrates [Gordian](https://github.com/gordian-engine/gordian)
with [the Cosmos SDK](https://github.com/cosmos/cosmos-sdk).

## Quickstart

Start the example Gordian-backed Cosmos SDK chain with four validators

```bash
CHAIN_ID=gchain-1 make testnet-start
```

### Interact

In a second terminal, interact with the running chain

Show validator address

```bash
VAL_ADDR=$(./gcosmos keys show val --address)
echo $VAL_ADDR
```

Query bank balance of validator, it has `9000000stake`
```bash
./gcosmos q bank balance $VAL_ADDR stake
```

### Shutdown

Shutdown the testnet by pressing Ctrl+C in the first terminal.

### Transaction Testing

Send `100stake` from the validator to a new account.

```bash
./gcosmos --chain-id gchain-1 tx bank send val cosmos10r39fueph9fq7a6lgswu4zdsg8t3gxlqvvvyvn 100stake
```

Confirm the balance in the new account, it now has `100stake`

```bash
./gcosmos q bank balance cosmos10r39fueph9fq7a6lgswu4zdsg8t3gxlqvvvyvn stake
```

## Additional Information

Refer to the `http*.go` files in [the gserver/internal/gsi directory](gserver/internal/gsi/) for more details on available HTTP paths.
