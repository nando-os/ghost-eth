package main

import (
	"fmt"
	"log"
	"math/big"

	"github.com/nando-os/gostheth/pkg"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

func setup() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Errorf("error loading ..env file:%+v", err)
		return
	}

	// To test TOR proxy functionality, set these environment variables:
	// HTTP_PROXY=socks5://127.0.0.1:9050
	// HTTPS_PROXY=socks5://127.0.0.1:9050
	// The service will automatically use the proxy when these are set
}

func main() {
	// --- Setup ---
	setup()

	// --- Load Configuration ---
	config, err := pkg.NewConfiguration()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// --- Get Account ---
	accounts := config.Accounts()
	if len(accounts) == 0 {
		log.Fatal("No accounts found in configuration")
	}
	sender := accounts[0]
	receiver := accounts[1]

	// --- Create Service ---
	service, err := pkg.NewEthereumClient(sender, config)
	if err != nil {
		log.Fatal("Failed to create service:", err)
	}
	defer service.(interface{ Close() }).Close()

	// --- Create Transaction ---
	// Send 0.001 ETH
	value := big.NewInt(1e15) // 0.001 ETH in wei
	recipient := common.HexToAddress(receiver.Address.String())

	tx := &pkg.Transaction{
		From:  sender.Address,
		To:    recipient,
		Value: value,
		Data:  []byte{}, // Simple ETH transfer
	}

	// --- Sign Transaction ---
	log.Println("Signing transaction...")
	signedTx, err := service.SignTransaction(tx)
	if err != nil {
		log.Fatal("Failed to sign transaction:", err)
	}

	// --- Send Transaction (Non-bloking) ---
	log.Println("Sending transaction...")
	receipt, err := service.SendTransaction(signedTx)
	if err != nil {
		log.Fatal("Failed to send transaction:", err)
	}

	log.Printf("Transaction sent! Hash: %s", receipt.TxHash.Hex())
	log.Printf("Status: %d (0 = pending)", receipt.Status)
	log.Printf("From: %s", receipt.From.Hex())
	log.Printf("To: %s", receipt.To.Hex())

	// --- Wait for Confirmation (Optional) ---
	log.Println("Waiting for transaction confirmation...")
	confirmedReceipt, err := service.WaitForTransaction(receipt.TxHash)
	if err != nil {
		log.Fatal("Transaction failed:", err)
	}

	log.Printf("Transaction confirmed!")
	log.Printf("Block Number: %d", confirmedReceipt.BlockNumber)
	log.Printf("Gas Used: %d", confirmedReceipt.GasUsed)
	etherumGasPrice := big.NewInt(0).Mul(big.NewInt(int64(confirmedReceipt.GasUsed)), signedTx.GasPrice()).Uint64()
	log.Printf("Gas Price: %d wei", etherumGasPrice)
	log.Printf("Status: %d (1 = success, 0 = failed)", confirmedReceipt.Status)

	if confirmedReceipt.Status == 1 {
		log.Println("✅ Transaction successful!")
	} else {
		log.Println("❌ Transaction failed!")
	}

	// --- Check Balance ---
	log.Println("Checking balance...")
	balance, err := service.GetBalance(sender.Address)
	if err != nil {
		log.Printf("Failed to get balance: %v", err)
	} else {
		log.Printf("Current balance: %s wei", balance.String())
	}
}
