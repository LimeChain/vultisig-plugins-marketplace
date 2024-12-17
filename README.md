
## Dev Setup

### Prerequisites

- Git
- Go
- Docker
- [Foundry (Anvil)](https://book.getfoundry.sh/anvil/)

### Fork Ethereum mainnet for local development

- `make start-network`
- `make restart-network`
- `make reset-network`

Send tx and get account balance

`cast send 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --value 1ether --rpc-url http://127.0.0.1:8545 --private-key $(PRIVATE_KEY)`

`cast balance 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --rpc-url http://127.0.0.1:8545`