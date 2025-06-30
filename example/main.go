package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/nando-os/gostheth/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

func setup(log *logrus.Logger) {
	err := godotenv.Load(".env")
	if err != nil {
		log.WithError(err).Warn("Warning: error loading .env file")
		return
	}

	// To test TOR proxy functionality, set these environment variables:
	// HTTP_PROXY=socks5://127.0.0.1:9050
	// HTTPS_PROXY=socks5://127.0.0.1:9050
	// The client will automatically use the proxy when these are set

	// Log environment variables for debugging
	log.WithFields(logrus.Fields{
		"ETH_RPC_URL":  os.Getenv("ETH_RPC_URL"),
		"ETH_CHAIN_ID": os.Getenv("ETH_CHAIN_ID"),
		"HTTP_PROXY":   os.Getenv("HTTP_PROXY"),
		"HTTPS_PROXY":  os.Getenv("HTTPS_PROXY"),
		"ETH_ACCOUNTS": os.Getenv("ETH_ACCOUNTS"),
	}).Info("Environment check")
}

func main() {
	// --
	// Initialize logrus logger
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{}) // Use JSON format for logs
	log.SetOutput(os.Stdout)                  // Output logs to stdout
	log.SetLevel(logrus.InfoLevel)            // Set log level to Info

	// --- Setup ---
	setup(log)

	// --- Load Configuration ---
	fmt.Println("Loading configuration...")
	config, err := eth.NewConfiguration()
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}
	log.WithField("chain_id", config.ChainID()).Info("Configuration loaded successfully")

	// --- Get Account ---
	accounts := config.Accounts()
	if len(accounts) == 0 {
		log.Fatal("No accounts found in configuration")
	}
	log.WithField("num_accounts", len(accounts)).Info("Found accounts")

	sender := accounts[0]
	receiver := accounts[1]

	log.WithFields(logrus.Fields{
		"sender":   sender.Address.Hex(),
		"receiver": receiver.Address.Hex(),
	}).Info("Account addresses")

	// --- Create client ---
	fmt.Println("Creating Ethereum client...")
	client, err := eth.NewGhostClient(sender, config, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create client")
	}
	defer client.(interface{ Close() }).Close()
	fmt.Println("Ethereum client created successfully")

	// --- Create Transaction ---
	// Send 0.001 ETH
	value := big.NewInt(1e15) // 0.001 ETH in wei
	recipient := common.HexToAddress(receiver.Address.String())

	log.WithFields(logrus.Fields{
		"amount_eth": new(big.Float).Quo(new(big.Float).SetInt(value), big.NewFloat(1e18)).Text('f', 6),
		"from":       sender.Address.Hex(),
		"to":         recipient.Hex(),
	}).Info("Creating transaction")

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
		log.WithError(err).Fatal("Failed to sign transaction")
	}
	log.WithField("tx_hash", signedTx.Hash().Hex()).Info("Transaction signed successfully")

	// --- Send Transaction (Non-bloking) ---
	fmt.Println("Sending transaction...")
	receipt, err := client.SendTransaction(signedTx)
	if err != nil {
		log.WithError(err).Fatal("Failed to send transaction")
	}

	log.WithFields(logrus.Fields{
		"tx_hash": receipt.TxHash.Hex(),
		"status":  receipt.Status,
		"from":    receipt.From.Hex(),
		"to":      receipt.To.Hex(),
	}).Info("Transaction sent")

	// --- Wait for Confirmation (Optional) ---
	fmt.Println("Waiting for transaction confirmation...")
	confirmedReceipt, err := client.WaitForTransaction(receipt.TxHash)
	if err != nil {
		log.WithError(err).Fatal("Transaction failed")
	}

	log.WithFields(logrus.Fields{
		"block_number": confirmedReceipt.BlockNumber,
		"gas_used":     confirmedReceipt.GasUsed,
		"gas_price":    big.NewInt(0).Mul(big.NewInt(int64(confirmedReceipt.GasUsed)), signedTx.GasPrice()).Uint64(),
		"status":       confirmedReceipt.Status,
	}).Info("Transaction confirmed")

	if confirmedReceipt.Status == 1 {
		fmt.Println("✅ Transaction successful!")
	} else {
		fmt.Println("❌ Transaction failed!")
	}

	// --- Check Balance ---
	fmt.Println("Checking balance...")
	balance, err := client.GetBalance(sender.Address)
	if err != nil {
		log.WithError(err).Warn("Failed to get balance")
	} else {
		log.WithField("balance_wei", balance.String()).Info("Current balance")
	}
}
