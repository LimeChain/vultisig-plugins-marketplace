package uniswap

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func privateKeyAndAddress(hexPrivateKey string) (*ecdsa.PrivateKey, common.Address, error) {
	if hexPrivateKey == "" {
		return nil, common.Address{}, fmt.Errorf("private key is not set")
	}
	privateKey, err := crypto.HexToECDSA(hexPrivateKey)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to get private key: %v", err)
	}
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKey)
	return privateKey, address, nil
}
