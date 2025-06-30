package main

import (
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/nando-os/gostheth/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

func setup() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("Warning: error loading .env file: %+v\n", err)
		return
	}

	// To test TOR proxy functionality, set these environment variables:
	// HTTP_PROXY=socks5://127.0.0.1:9050
	// HTTPS_PROXY=socks5://127.0.0.1:9050
	// The client will automatically use the proxy when these are set

	// Log environment variables for debugging
	fmt.Printf("Environment check:\n")
	fmt.Printf("  ETH_RPC_URL: %s\n", os.Getenv("ETH_RPC_URL"))
	fmt.Printf("  ETH_CHAIN_ID: %s\n", os.Getenv("ETH_CHAIN_ID"))
	fmt.Printf("  HTTP_PROXY: %s\n", os.Getenv("HTTP_PROXY"))
	fmt.Printf("  HTTPS_PROXY: %s\n", os.Getenv("HTTPS_PROXY"))
	fmt.Printf("  ETH_ACCOUNTS: %s\n", os.Getenv("ETH_ACCOUNTS"))
}

func main() {
	// --- Setup ---
	setup()

	// --- Load Configuration ---
	fmt.Println("Loading configuration...")
	config, err := eth.NewConfiguration()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}
	fmt.Printf("Configuration loaded successfully. Chain ID: %d\n", config.ChainID())

	// --- Get Account ---
	accounts := config.Accounts()
	if len(accounts) == 0 {
		log.Fatal("No accounts found in configuration")
	}
	fmt.Printf("Found %d accounts\n", len(accounts))

	sender := accounts[0]
	receiver := accounts[1]

	fmt.Printf("Sender: %s\n", sender.Address.Hex())
	fmt.Printf("Receiver: %s\n", receiver.Address.Hex())

	// --- Create client ---
	fmt.Println("Creating Ethereum client...")
	client, err := eth.NewGhostClient(sender, config)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.(interface{ Close() }).Close()
	fmt.Println("Ethereum client created successfully")

	// --- Create Transaction ---
	// Send 0.001 ETH
	value := big.NewInt(1e15) // 0.001 ETH in wei
	recipient := common.HexToAddress(receiver.Address.String())

	fmt.Printf("Creating transaction: %s ETH from %s to %s\n",
		new(big.Float).Quo(new(big.Float).SetInt(value), big.NewFloat(1e18)).Text('f', 6),
		sender.Address.Hex(),
		recipient.Hex())

	tx := &eth.Transaction{
		From:  sender.Address,
		To:    recipient,
		Value: value,
		Data:  []byte{}, // Simple ETH transfer
	}

	// --- Sign Transaction ---
	fmt.Println("Signing transaction...")
	signedTx, err := client.SignTransaction(tx)
	if err != nil {
		log.Fatal("Failed to sign transaction:", err)
	}
	fmt.Printf("Transaction signed successfully. Hash: %s\n", signedTx.Hash().Hex())

	// --- Send Transaction (Non-bloking) ---
	fmt.Println("Sending transaction...")
	receipt, err := client.SendTransaction(signedTx)
	if err != nil {
		log.Fatal("Failed to send transaction:", err)
	}

	fmt.Printf("Transaction sent! Hash: %s\n", receipt.TxHash.Hex())
	fmt.Printf("Status: %d (0 = pending)\n", receipt.Status)
	fmt.Printf("From: %s\n", receipt.From.Hex())
	fmt.Printf("To: %s\n", receipt.To.Hex())

	// --- Wait for Confirmation (Optional) ---
	fmt.Println("Waiting for transaction confirmation...")
	confirmedReceipt, err := client.WaitForTransaction(receipt.TxHash)
	if err != nil {
		log.Fatal("Transaction failed:", err)
	}

	fmt.Printf("Transaction confirmed!\n")
	fmt.Printf("Block Number: %d\n", confirmedReceipt.BlockNumber)
	fmt.Printf("Gas Used: %d\n", confirmedReceipt.GasUsed)
	etherumGasPrice := big.NewInt(0).Mul(big.NewInt(int64(confirmedReceipt.GasUsed)), signedTx.GasPrice()).Uint64()
	fmt.Printf("Gas Price: %d wei\n", etherumGasPrice)
	fmt.Printf("Status: %d (1 = success, 0 = failed)\n", confirmedReceipt.Status)

	if confirmedReceipt.Status == 1 {
		fmt.Println("✅ Transaction successful!")
	} else {
		fmt.Println("❌ Transaction failed!")
	}

	// --- Check Balance ---
	fmt.Println("Checking balance...")
	balance, err := client.GetBalance(sender.Address)
	if err != nil {
		fmt.Printf("Failed to get balance: %v\n", err)
	} else {
		fmt.Printf("Current balance: %s wei\n", balance.String())
	}
}
