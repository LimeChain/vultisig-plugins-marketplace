INFURA_ETHEREUM_MAINNET=""
BLOCK_NUMBER=21432092

start-network:
	anvil --fork-url $(INFURA_ETHEREUM_MAINNET) \
	--fork-block-number $(BLOCK_NUMBER) \
	--accounts 10 --balance 1000000 \
	--mnemonic "test test test test test test test test test test test junk" \
	--state anvil-state.json

restart-network:
	anvil --fork-url $(INFURA_ETHEREUM_MAINNET) \
	--state ./anvil-state.json

reset-network:
	rm -rf ./anvil-state.json
