package uniswap

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Config struct {
	GasLimitBuffer   uint64
	SwapGasLimit     uint64
	DeadlineDuration time.Duration
}

type Client struct {
	cfg              Config
	ethClient        *ethclient.Client
	routerAddress    common.Address
	signerAddress    common.Address
	signerPrivateKey *ecdsa.PrivateKey
}

func NewClient(rpcUrl string, routerAddress common.Address, hexPrivateKey string, cfg Config) (*Client, error) {
	ethClient, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}

	signerPrivateKey, signerAddress, err := privateKeyAndAddress(hexPrivateKey)
	if err != nil {
		return nil, err
	}

	return &Client{
		cfg,
		ethClient,
		routerAddress,
		signerAddress,
		signerPrivateKey,
	}, nil
}

func (uc *Client) MintWETH(amount *big.Int, tokenAddress common.Address) error {
	wethABI := `[{"name":"deposit","type":"function","payable":true}]`
	parsedABI, err := abi.JSON(strings.NewReader(wethABI))
	if err != nil {
		return err
	}

	data, err := parsedABI.Pack("deposit")
	if err != nil {
		return err
	}
	gasPrice, err := uc.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	nonce, err := uc.ethClient.PendingNonceAt(context.Background(), uc.signerAddress)
	if err != nil {
		return err
	}
	gasLimit, err := uc.ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		To:   &tokenAddress,
		Data: data,
	})
	if err != nil {
		return err
	}
	gasLimit += uc.cfg.GasLimitBuffer
	tx := types.NewTransaction(nonce, tokenAddress, amount, gasLimit, gasPrice, data)
	return uc.sendTransaction(tx)
}

func (uc *Client) ApproveERC20Token(tokenAddress, spenderAddress common.Address, amount *big.Int) error {
	tokenABI := `[
		{
			"name": "approve",
			"type": "function",
			"inputs": [
				{
					"name": "spender",
					"type": "address"
				},
				{
					"name": "value",
					"type": "uint256"
				}
			],
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			]
		}
	]`
	parsedABI, err := abi.JSON(strings.NewReader(tokenABI))
	if err != nil {
		return err
	}

	approveData, err := parsedABI.Pack("approve", spenderAddress, amount)
	if err != nil {
		return err
	}
	nonce, err := uc.ethClient.PendingNonceAt(context.Background(), uc.signerAddress)
	if err != nil {
		return err
	}
	gasPrice, err := uc.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	gasLimit, err := uc.ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		To:   &tokenAddress,
		Data: approveData,
	})
	if err != nil {
		return err
	}
	gasLimit += uc.cfg.GasLimitBuffer
	tx := types.NewTransaction(nonce, tokenAddress, big.NewInt(0), gasLimit, gasPrice, approveData)
	return uc.sendTransaction(tx)
}

func (uc *Client) SwapTokens(amountIn, amountOutMin *big.Int, path []common.Address) error {
	log.Println("Swapping tokens...")
	routerABI := `[
		{
			"name": "swapExactTokensForTokens",
			"type": "function",
			"inputs": [
				{
					"name": "amountIn",
					"type": "uint256"
				},
				{
					"name": "amountOutMin",
					"type": "uint256"
				},
				{
					"name": "path",
					"type": "address[]"
				},
				{
					"name": "to",
					"type": "address"
				},
				{
					"name": "deadline",
					"type": "uint256"
				}
			]
		}
	]`
	parsedRouterABI, err := abi.JSON(strings.NewReader(routerABI))
	if err != nil {
		return err
	}

	deadline := big.NewInt(time.Now().Add(uc.cfg.DeadlineDuration).Unix())

	swapData, err := parsedRouterABI.Pack("swapExactTokensForTokens", amountIn, amountOutMin, path, uc.signerAddress, deadline)
	if err != nil {
		return err
	}
	nonce, err := uc.ethClient.PendingNonceAt(context.Background(), uc.signerAddress)
	if err != nil {
		return err
	}
	gasPrice, err := uc.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	tx := types.NewTransaction(nonce, uc.routerAddress, big.NewInt(0), uc.cfg.SwapGasLimit, gasPrice, swapData)
	return uc.sendTransaction(tx)
}

func (uc *Client) GetTokenBalance(tokenAddress common.Address) (*big.Int, error) {
	tokenABI := `[
		{
			"name": "balanceOf",
			"type": "function",
			"inputs": [
				{
					"name": "account",
					"type": "address"
				}
			],
			"outputs": [
				{
					"name": "",
					"type": "uint256"
				}
			]
		}
	]`
	parsedABI, err := abi.JSON(strings.NewReader(tokenABI))
	if err != nil {
		return nil, err
	}
	callData, err := parsedABI.Pack("balanceOf", uc.signerAddress)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: callData,
	}

	result, err := uc.ethClient.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}

	balance := new(big.Int)
	balance.SetBytes(result)
	return balance, nil
}

func (uc *Client) GetExpectedAmountOut(amountIn *big.Int, path []common.Address) (*big.Int, error) {
	routerABI := `[
		{
			"name": "getAmountsOut",
			"type": "function",
			"inputs": [
				{
					"name": "amountIn",
					"type": "uint256"
				},
				{
					"name": "path",
					"type": "address[]"
				}
			],
			"outputs": [
				{
					"name": "",
					"type": "uint256[]"
				}
			]
		}
	]`
	parsedABI, err := abi.JSON(strings.NewReader(routerABI))
	if err != nil {
		return nil, err
	}

	callData, err := parsedABI.Pack("getAmountsOut", amountIn, path)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &uc.routerAddress,
		Data: callData,
	}

	result, err := uc.ethClient.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}

	var amountsOut []*big.Int
	err = parsedABI.UnpackIntoInterface(&amountsOut, "getAmountsOut", result)
	if err != nil {
		return nil, err
	}

	if len(amountsOut) < 2 {
		return nil, fmt.Errorf("unexpected result length")
	}

	return amountsOut[len(amountsOut)-1], nil
}

func (uc *Client) sendTransaction(tx *types.Transaction) error {
	chainID, err := uc.ethClient.NetworkID(context.Background())
	if err != nil {
		return err
	}
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), uc.signerPrivateKey)
	if err != nil {
		return err
	}
	err = uc.ethClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}
	log.Printf("Transaction sent: %s", signedTx.Hash().Hex())

	receipt, err := bind.WaitMined(context.Background(), uc.ethClient, signedTx)
	if err != nil {
		return err
	}
	log.Printf("Transaction receipt status: %v", receipt.Status)
	return nil
}

func (uc *Client) CalculateAmountOutMin(expectedAmountOut *big.Int, slippagePercentage float64) *big.Int {
	slippageFactor := big.NewFloat(1 - slippagePercentage/100)
	expectedAmountOutFloat := new(big.Float).SetInt(expectedAmountOut)
	amountOutMinFloat := new(big.Float).Mul(expectedAmountOutFloat, slippageFactor)
	amountOutMin, _ := amountOutMinFloat.Int(nil)
	return amountOutMin
}
