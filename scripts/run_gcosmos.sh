#!/bin/sh

CHAIN_ID=${CHAIN_ID:-"localchain-1"}
G_HTTP_ADDR=${G_HTTP_ADDR:-"26657"}
G_GRPC_ADDR=${G_GRPC_ADDR:-"9092"}

# cleanup previous run data
rm -rf ./test
mkdir -p ./test/genesis/config/gentx

trap 'rm -rf ./test; echo; echo "Gordian Cosmos testnet stopped"; exit' INT

# Separate dir used for genesis.json assembly (could also use as fullnode)
./gcosmos init genesis --chain-id=${CHAIN_ID} --home ./test/genesis 

./gcosmos init val1 --chain-id=${CHAIN_ID} --home ./test/val1
./gcosmos init val2 --chain-id=${CHAIN_ID} --home ./test/val2
./gcosmos init val3 --chain-id=${CHAIN_ID} --home ./test/val3
./gcosmos init val4 --chain-id=${CHAIN_ID} --home ./test/val4

# Disable conflicting services on vals 2-4
./gcosmos config set app telemetry.enable false --home ./test/val2
./gcosmos config set app telemetry.enable false --home ./test/val3
./gcosmos config set app telemetry.enable false --home ./test/val4

./gcosmos config set app grpc.address localhost:0 --home ./test/val2
./gcosmos config set app grpc.address localhost:0 --home ./test/val3
./gcosmos config set app grpc.address localhost:0 --home ./test/val4

./gcosmos config set app rest.enable false --home ./test/val2
./gcosmos config set app rest.enable false --home ./test/val3
./gcosmos config set app rest.enable false --home ./test/val4

./gcosmos keys add val --no-backup --keyring-backend=test --home ./test/val1
./gcosmos keys add val --no-backup --keyring-backend=test --home ./test/val2
./gcosmos keys add val --no-backup --keyring-backend=test --home ./test/val3
./gcosmos keys add val --no-backup --keyring-backend=test --home ./test/val4

./gcosmos genesis add-genesis-account val 10000000stake --home ./test/val1 2>&1 > /dev/null 
./gcosmos genesis add-genesis-account val 10000000stake --home ./test/val2 2>&1 > /dev/null
./gcosmos genesis add-genesis-account val 10000000stake --home ./test/val3 2>&1 > /dev/null
./gcosmos genesis add-genesis-account val 10000000stake --home ./test/val4 2>&1 > /dev/null

VAL1ADDR=$(./gcosmos keys show val -a --keyring-backend=test --home ./test/val1)
VAL2ADDR=$(./gcosmos keys show val -a --keyring-backend=test --home ./test/val2)
VAL3ADDR=$(./gcosmos keys show val -a --keyring-backend=test --home ./test/val3)
VAL4ADDR=$(./gcosmos keys show val -a --keyring-backend=test --home ./test/val4)

./gcosmos genesis add-genesis-account ${VAL1ADDR} 10000000stake --home ./test/genesis 2>&1 > /dev/null
./gcosmos genesis add-genesis-account ${VAL2ADDR} 10000000stake --home ./test/genesis 2>&1 > /dev/null
./gcosmos genesis add-genesis-account ${VAL3ADDR} 10000000stake --home ./test/genesis 2>&1 > /dev/null
./gcosmos genesis add-genesis-account ${VAL4ADDR} 10000000stake --home ./test/genesis 2>&1 > /dev/null

./gcosmos genesis gentx val 1000000stake --chain-id=${CHAIN_ID} --home ./test/val1 --output-document ./test/genesis/config/gentx/1.json &> ./test/val1/gentx.log
./gcosmos genesis gentx val 1000000stake --chain-id=${CHAIN_ID} --home ./test/val2 --output-document ./test/genesis/config/gentx/2.json &> ./test/val2/gentx.log
./gcosmos genesis gentx val 1000000stake --chain-id=${CHAIN_ID} --home ./test/val3 --output-document ./test/genesis/config/gentx/3.json &> ./test/val3/gentx.log
./gcosmos genesis gentx val 1000000stake --chain-id=${CHAIN_ID} --home ./test/val4 --output-document ./test/genesis/config/gentx/4.json &> ./test/val4/gentx.log

# Collect all the gentxs to add the validators to the genesis file
./gcosmos genesis collect-gentxs --home ./test/genesis 2>&1 > /dev/null

# Distribute the final genesis.json to all the validators
cp ./test/genesis/config/genesis.json ./test/val1/config/genesis.json 
cp ./test/genesis/config/genesis.json ./test/val2/config/genesis.json
cp ./test/genesis/config/genesis.json ./test/val3/config/genesis.json
cp ./test/genesis/config/genesis.json ./test/val4/config/genesis.json

# Start the seed node
./gcosmos gordian seed ./test/p2p.seed.txt &

# Wait for the seed node to start
sleep 2
SEED_ADDR="$(head -n1 <./test/p2p.seed.txt)"

# Start vals 2-4 in the background
./gcosmos start --g-grpc-addr 127.0.0.1:0 --g-seed-addrs ${SEED_ADDR} --home ./test/val2 &> ./test/val2/node.log &
./gcosmos start --g-grpc-addr 127.0.0.1:0 --g-seed-addrs ${SEED_ADDR} --home ./test/val3 &> ./test/val3/node.log &
./gcosmos start --g-grpc-addr 127.0.0.1:0 --g-seed-addrs ${SEED_ADDR} --home ./test/val4 &> ./test/val4/node.log &

# Show first val logs in the terminal
./gcosmos start --g-http-addr 127.0.0.1:$G_HTTP_ADDR --g-grpc-addr 127.0.0.1:$G_GRPC_ADDR --g-seed-addrs ${SEED_ADDR} --home ./test/val1 2>&1 | tee ./test/val1/node.log