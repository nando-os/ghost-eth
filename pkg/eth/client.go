package pkg

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthereumClient interface {
	// SendTransaction sends a signed transaction to the network
	SendTransaction(signedTx *types.Transaction) (*TransactionReceipt, error)

	// SignTransaction signs a transaction with the service's private key
	SignTransaction(tx *Transaction) (*types.Transaction, error)

	// GetBalance returns the ETH balance of an address
	GetBalance(address common.Address) (*big.Int, error)

	// WaitForTransaction waits for a transaction to be mined and returns the receipt
	WaitForTransaction(hash common.Hash) (*TransactionReceipt, error)

	// GetTransactionReceipt returns the receipt for a transaction if it exists
	GetTransactionReceipt(hash common.Hash) (*TransactionReceipt, error)

	// Close closes the Ethereum client connection
	Close()
}

type ethereumClient struct {
	client  *ethclient.Client
	ctx     context.Context
	chainId int64
	account *Account
	config  *config
}

func NewEthereumClient(account *Account, cfg *config) (EthereumClient, error) {
	ctx := context.Background()
	chainId := account.ChainId

	// -- validate account
	if account.PrivateKey == nil {
		return nil, fmt.Errorf("account private key is nil")
	}

	if account.Address == (common.Address{}) {
		return nil, fmt.Errorf("account address is not set")
	}

	if account.ChainId == 0 {
		return nil, fmt.Errorf("account chain ID is not set")
	}

	if account.PublicKey == nil {
		return nil, fmt.Errorf("account public key is not set")
	}

	// -- Connect to Ethereum client
	// HTTP_PROXY and HTTPS_PROXY environment variables are automatically used by ethclient.DialContext
	client, err := ethclient.DialContext(ctx, cfg.RPCURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum network: %w", err)
	}

	// Log proxy usage if configured
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" {
		log.Info("Connected to Ethereum network via proxy",
			"http_proxy", os.Getenv("HTTP_PROXY"),
			"https_proxy", os.Getenv("HTTPS_PROXY"))
	} else {
		log.Info("Connected to Ethereum network directly")
	}

	// -- Verify conection and get chain ID
	clientChainId, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// -- Check if chain ID matches config
	if clientChainId.Int64() != chainId {
		return nil, fmt.Errorf("expected chain ID %d, got %d", chainId, clientChainId.Int64())
	}

	return &ethereumClient{
		client:  client,
		ctx:     ctx,
		chainId: clientChainId.Int64(),
		account: account,
		config:  cfg,
	}, nil
}

// SendTransaction sends a signed transaction to the network
func (es *ethereumClient) SendTransaction(signedTx *types.Transaction) (*TransactionReceipt, error) {
	// Send the transaction
	err := es.client.SendTransaction(es.ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Return immediately with transaction hash
	return &TransactionReceipt{
		TxHash: signedTx.Hash(),
		Status: 0,                  // Pending
		From:   es.account.Address, // Use known address
		To:     *signedTx.To(),
	}, nil
}

// WaitForTransaction waits for a transaction to be mined and returns the receipt
func (es *ethereumClient) WaitForTransaction(hash common.Hash) (*TransactionReceipt, error) {
	receipt, err := es.waitForTransaction(hash)
	if err != nil {
		return nil, err
	}

	// Get the transaction to find the To address
	tx, _, err := es.client.TransactionByHash(es.ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &TransactionReceipt{
		TxHash:      receipt.TxHash,
		Status:      receipt.Status,
		BlockNumber: receipt.BlockNumber,
		GasUsed:     receipt.GasUsed,
		From:        es.account.Address, // Use known address
		To:          *tx.To(),           // Get To address from transaction
		Logs:        receipt.Logs,
	}, nil
}

// SignTransaction signs a transaction with the service's private key
func (es *ethereumClient) SignTransaction(tx *Transaction) (*types.Transaction, error) {
	// Get nonce if not provided
	if tx.Nonce == 0 {
		nonce, err := es.client.PendingNonceAt(es.ctx, tx.From)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce: %w", err)
		}
		tx.Nonce = nonce
	}

	// Estimate gas if not provided
	if tx.GasLimit == 0 {
		msg := ethereum.CallMsg{
			From:  tx.From,
			To:    &tx.To,
			Value: tx.Value,
			Data:  tx.Data,
		}

		gasLimit, err := es.client.EstimateGas(es.ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas: %w", err)
		}

		// Add dynamic buffer based on transaction complexity
		var buffer float64
		if len(tx.Data) == 0 {
			buffer = es.config.GasLimitBufferSimple() // Configurable buffer for simple ETH transfers
		} else {
			buffer = es.config.GasLimitBufferComplex() // Configurable buffer for complex transactions
		}
		tx.GasLimit = uint64(float64(gasLimit) * buffer)

		// Validate against network gas limit, transaction will get blocked if goes above it
		header, err := es.client.HeaderByNumber(es.ctx, nil)
		if err == nil && header.GasLimit > 0 {
			maxGas := header.GasLimit * 2 / 3 // Use 2/3 of block gas limit
			if tx.GasLimit > maxGas {
				return nil, fmt.Errorf("gas limit %d exceeds maximum allowed %d", tx.GasLimit, maxGas)
			}
		}
	}

	// Calulate fees based on network conditions
	err := es.calculateOptimalFees(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fees: %w", err)
	}

	var ethereumTx *types.Transaction

	if tx.MaxFeePerGas != nil && tx.MaxPriorityFeePerGas != nil {
		// EIP-1559 transaction
		ethereumTx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(es.chainId),
			Nonce:     tx.Nonce,
			GasTipCap: tx.MaxPriorityFeePerGas,
			GasFeeCap: tx.MaxFeePerGas,
			Gas:       tx.GasLimit,
			To:        &tx.To,
			Value:     tx.Value,
			Data:      tx.Data,
		})
	} else if tx.GasPrice != nil {
		// Legacy transaction
		ethereumTx = types.NewTransaction(
			tx.Nonce,
			tx.To,
			tx.Value,
			tx.GasLimit,
			tx.GasPrice,
			tx.Data,
		)
	} else {
		return nil, fmt.Errorf("transaction must specify either EIP-1559 fields (MaxFeePerGas, MaxPriorityFeePerGas) or legacy GasPrice")
	}

	// Sign the transaction
	signedTx, err := types.SignTx(ethereumTx, types.LatestSignerForChainID(big.NewInt(es.chainId)), es.account.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return signedTx, nil
}

// calculateOptimalFees calculates optimal gas fees based on network conditions
func (es *ethereumClient) calculateOptimalFees(tx *Transaction) error {
	// Get latest header for base fee
	header, err := es.client.HeaderByNumber(es.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest header: %w", err)
	}

	// Check if network supports EIP-1559
	if header.BaseFee != nil && tx.MaxFeePerGas == nil || tx.MaxPriorityFeePerGas == nil {
		log.Info("Using EIP-1559 fee calculation")
		// EIP-1559 network - calculate optimal fees
		// Use fixed priority fee based on network
		tx.MaxPriorityFeePerGas = es.getFixedPriorityFee()

		// Calculate max fee with room for base fee increases
		maxFee := new(big.Int).Mul(header.BaseFee, big.NewInt(2)) // 2x base fee
		maxFee.Add(maxFee, tx.MaxPriorityFeePerGas)
		tx.MaxFeePerGas = maxFee
	} else {
		log.Info("Using legacy fee calculation")
		// Legacy network - use gas price
		if tx.GasPrice == nil {
			gasPrice, err := es.client.SuggestGasPrice(es.ctx)
			if err != nil {
				return fmt.Errorf("failed to get gas price: %w", err)
			}
			tx.GasPrice = gasPrice
		}
	}

	// Basic validation
	return es.validateFees(tx)
}

// getFixedPriorityFee returns a fixed priority fee based on the network
func (es *ethereumClient) getFixedPriorityFee() *big.Int {
	switch es.chainId {
	case 1: // Ethereum mainnet
		return es.config.PriorityFeeMainnet()
	case 8453: // Base
		return es.config.PriorityFeeBase()
	default:
		return es.config.PriorityFeeDefault()
	}
}

// validateFees does basic fee validation
func (es *ethereumClient) validateFees(tx *Transaction) error {
	if tx.MaxFeePerGas == nil {
		return nil // Legacy transaction
	}

	// Check if max fee is reasonable (prevent overpayment)
	maxAllowed := es.config.MaxFeePerGas()
	if tx.MaxFeePerGas.Cmp(maxAllowed) > 0 {
		return fmt.Errorf("max fee too high: %s wei", tx.MaxFeePerGas.String())
	}

	return nil
}

// GetBalance returns the ETH balance of an address
func (es *ethereumClient) GetBalance(address common.Address) (*big.Int, error) {
	balance, err := es.client.BalanceAt(es.ctx, address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// waitForTransaction waits for a transaction to be mined
func (es *ethereumClient) waitForTransaction(hash common.Hash) (*TransactionReceipt, error) {
	timeout := time.Duration(es.config.TransactionTimeoutSeconds()) * time.Second
	tickerInterval := time.Duration(es.config.TransactionTickerSeconds()) * time.Second

	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			return nil, fmt.Errorf("transaction timeout: %s", hash.Hex())
		case <-ticker.C:
			receipt, err := es.GetTransactionReceipt(hash)
			if err == nil {
				return receipt, nil
			}
		}
	}
}

// GetTransactionReceipt returns the receipt for a transaction if it exists
func (es *ethereumClient) GetTransactionReceipt(hash common.Hash) (*TransactionReceipt, error) {
	receipt, err := es.client.TransactionReceipt(es.ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("transaction not found or pending: %w", err)
	}

	// Get the transaction to find the To address
	tx, _, err := es.client.TransactionByHash(es.ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &TransactionReceipt{
		TxHash:      receipt.TxHash,
		Status:      receipt.Status,
		BlockNumber: receipt.BlockNumber.Uint64(),
		GasUsed:     receipt.GasUsed,
		From:        es.account.Address, // Use known address
		To:          *tx.To(),           // Get To address from transaction
		Logs:        receipt.Logs,
	}, nil
}

// Close closes the Ethereum client connection
func (es *ethereumClient) Close() {
	if es.ctx != nil {
		es.ctx.Done() // Signal context cancellation
		es.ctx = nil  // Prevent further use
	}
	if es.client != nil {
		es.client.Close()
	}
}
