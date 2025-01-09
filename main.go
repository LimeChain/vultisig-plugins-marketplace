package main

import (
	"log"
	"math/big"
	"time"

	"os"

	"github.com/LimeChain/vulti-mono/uniswap"
	"github.com/ethereum/go-ethereum/common"
)

const (
	rpcURL = "http://127.0.0.1:8545"
)

var (
	uniswapV2RouterAddress = common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")

	tokenInAddress  = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2") // WETH
	tokenOutAddress = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48") // UCDC

	swapAmountIn = big.NewInt(1e18)
	amountOutMin = big.NewInt(1)
)

var (
	signerPrivateKey = os.Getenv("PRIVATE_KEY")
)

func logTokenBalances(client *uniswap.Client) {
	tokenInBalance, err := client.GetTokenBalance(tokenInAddress)
	if err != nil {
		log.Printf("Error getting input token balance: %v", err)
		return
	}
	log.Printf("input token balance: %s", tokenInBalance.String())

	tokenOutBalance, err := client.GetTokenBalance(tokenOutAddress)
	if err != nil {
		log.Printf("Error getting output token balance: %v", err)
		return
	}
	log.Printf("output token balance: %s", tokenOutBalance.String())
}

func main() {
	cfg := uniswap.Config{
		GasLimitBuffer:   50000,
		SwapGasLimit:     1000000,
		DeadlineDuration: 15 * time.Minute,
	}

	uniswapClient, err := uniswap.NewClient(rpcURL, uniswapV2RouterAddress, signerPrivateKey, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Uniswap client: %v", err)
	}

	tokensPair := []common.Address{tokenInAddress, tokenOutAddress}

	// fetch token pair amount out
	expectedAmountOut, err := uniswapClient.GetExpectedAmountOut(swapAmountIn, tokensPair)
	if err != nil {
		log.Fatalf("Failed to get expected amount out: %v", err)
	}
	log.Println("Expected amount out:", expectedAmountOut.String())

	// calculate output amount with slippage
	slippagePercentage := 1.0
	amountOutMin := uniswapClient.CalculateAmountOutMin(expectedAmountOut, slippagePercentage)

	// mint WETH
	log.Println("Minting WETH...")
	logTokenBalances(uniswapClient)
	if err := uniswapClient.MintWETH(swapAmountIn, tokenInAddress); err != nil {
		log.Fatalf("Failed to mint WETH: %v", err)
	}
	logTokenBalances(uniswapClient)

	// approve Router to spend input token
	log.Printf("Approving Uniswap Router to spend %s...", tokenInAddress.Hex())
	if err := uniswapClient.ApproveERC20Token(tokenInAddress, uniswapV2RouterAddress, swapAmountIn); err != nil {
		log.Fatalf("Failed to approve token: %v", err)
	}

	// swap tokens
	if err := uniswapClient.SwapTokens(swapAmountIn, amountOutMin, tokensPair); err != nil {
		log.Fatalf("Failed to swap tokens: %v", err)
	}
	logTokenBalances(uniswapClient)
}
