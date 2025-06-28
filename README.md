# Gostheth

A Go client wrapper for Ethereum with online wallet capabilities, TOR proxy support, and smart fee calculation.

## Features

- **🔐 Multi-Account Support**: Load multiple wallet accounts from environment variables
- **⚡ Smart Gas Management**: Automatic EIP-1559 fee calculation with configurable buffers
- **🌐 TOR Proxy Support**: Optional anonymous connections via HTTP_PROXY/HTTPS_PROXY
- **🔄 Transaction Management**: Sign, send, and monitor Ethereum transactions
- **⚙️ Configurable**: Environment-based configuration with sensible defaults
- **🛡️ Production Ready**: Comprehensive error handling and validation

## Quick Start

### Installation

```bash
go get github.com/nando-os/gostheth
```

### Basic Usage

```go
package main

import (
	"log"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
	"github.com/nando-os/gostheth/pkg"
)

func main() {
	// Load configuration
	config, err := pkg.NewConfiguration()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Get accounts
	accounts := config.Accounts()
	if len(accounts) == 0 {
		log.Fatal("No accounts found")
	}

	// Create service with first account
	service, err := pkg.NewEthereumClient(accounts[0], config)
	if err != nil {
		log.Fatal("Failed to create service:", err)
	}
	defer service.Close()

	// Create transaction
	tx := &pkg.Transaction{
		From:  accounts[0].Address,
		To:    common.HexToAddress("0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6"),
		Value: big.NewInt(1000000000000000000), // 1 ETH
		Data:  []byte{}, // Simple ETH transfer
	}

	// Sign and send transaction
	signedTx, err := service.SignTransaction(tx)
	if err != nil {
		log.Fatal("Failed to sign transaction:", err)
	}

	receipt, err := service.SendTransaction(signedTx)
	if err != nil {
		log.Fatal("Failed to send transaction:", err)
	}

	log.Printf("Transaction sent! Hash: %s", receipt.TxHash.Hex())
}
```

## Configuration

### Environment Variables

#### Required
```bash
# Network settings
ETH_CHAIN_ID=1                    # 1 for Ethereum, 8453 for Base
ETH_RPC_URL=https://mainnet.infura.io/v3/YOUR_KEY

# Account configuration
ETH_ACCOUNTS=main,backup          # Account labels
ETH_ACCOUNT_MAIN_PRIVATE_KEY=0x... # Private key for 'main'
ETH_ACCOUNT_BACKUP_PRIVATE_KEY=0x... # Private key for 'backup'
```

#### Optional
```bash
# Gas configuration (environment variable names)
ETH_GAS_LIMIT_BUFFER_SIMPLE=1.1   # Buffer for simple ETH transfers
ETH_GAS_LIMIT_BUFFER_COMPLEX=1.2  # Buffer for complex transactions

# Fee configuration
ETH_MAX_FEE_PER_GAS=500000000000  # Max fee per gas in wei (500 gwei)
ETH_PRIORITY_FEE_MAINNET=2000000000  # Priority fee for mainnet (2 gwei)
ETH_PRIORITY_FEE_BASE=1000000000     # Priority fee for Base (1 gwei)
ETH_PRIORITY_FEE_DEFAULT=1500000000  # Priority fee for other networks (1.5 gwei)

# TOR proxy (optional)
HTTP_PROXY=socks5://127.0.0.1:9050
HTTPS_PROXY=socks5://127.0.0.1:9050

# Transaction monitoring
ETH_TRANSACTION_TIMEOUT_SECONDS=300  # 5 minutes
ETH_TRANSACTION_TICKER_SECONDS=3     # 3 seconds
```

## API Reference

### Service Interface

```go
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
```

### Configuration Interface

```go
type config struct {
	chainId int64
	accounts []*Account
	rpcURL  string
}

// Methods:
ChainID() int64
Accounts() []*Account
RPCURL() string
GasLimitBufferSimple() float64
GasLimitBufferComplex() float64
MaxFeePerGas() *big.Int
PriorityFeeMainnet() *big.Int
PriorityFeeBase() *big.Int
PriorityFeeDefault() *big.Int
TransactionTimeoutSeconds() int
TransactionTickerSeconds() int
```

### Core Types

```go
// Account represents an Ethereum account
type Account struct {
	Address    common.Address    // Ethereum address
	PublicKey  *ecdsa.PublicKey  // Public key (optional, can be derived)
	ChainId    int64             // Chain ID for transaction signing
	Label      string            // Optional: human-readable label
	PrivateKey *ecdsa.PrivateKey // Private key for signing transactions
}

// Transaction represents an Ethereum transaction
type Transaction struct {
	From                 common.Address `json:"from"`
	To                   common.Address `json:"to"`
	Value                *big.Int       `json:"value"`
	Data                 []byte         `json:"data"`
	GasLimit             uint64         `json:"gas_limit"`
	GasPrice             *big.Int       `json:"gas_price"`
	MaxFeePerGas         *big.Int       `json:"max_fee_per_gas"`
	MaxPriorityFeePerGas *big.Int       `json:"max_priority_fee_per_gas"`
	Nonce                uint64         `json:"nonce"`
	ChainID              *big.Int       `json:"chain_id"`
}

// TransactionReceipt represents transaction execution result
type TransactionReceipt struct {
	TxHash      common.Hash    `json:"tx_hash"`
	Status      uint64         `json:"status"`
	BlockNumber uint64         `json:"block_number"`
	GasUsed     uint64         `json:"gas_used"`
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	Logs        []*types.Log   `json:"logs"`
}
```

## Examples

### Multi-Account Usage

```go
config, err := pkg.NewConfiguration()
if err != nil {
	log.Fatal(err)
}

accounts := config.Accounts()
if len(accounts) < 2 {
	log.Fatal("Need at least 2 accounts")
}

// Create service with specific account
service, err := pkg.NewEthereumClient(accounts[0], config)
if err != nil {
	log.Fatal(err)
}
defer service.Close()

// Use account for transaction
tx := &pkg.Transaction{
	From: accounts[0].Address,
	To:   accounts[1].Address,
	// ... other fields
}
```

### TOR Proxy Usage

```go
// Set environment variables before creating service
os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:9050")
os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:9050")

config, err := pkg.NewConfiguration()
if err != nil {
	log.Fatal(err)
}

service, err := pkg.NewEthereumClient(accounts[0], config)
if err != nil {
	log.Fatal(err)
}
defer service.Close()

// All transactions will now go through TOR
```

### Custom Gas Configuration

```go
// Set custom gas buffers
os.Setenv("ETH_GAS_LIMIT_BUFFER_SIMPLE", "1.05")   // 5% buffer
os.Setenv("ETH_GAS_LIMIT_BUFFER_COMPLEX", "1.15")  // 15% buffer

config, err := pkg.NewConfiguration()
if err != nil {
	log.Fatal(err)
}

service, err := pkg.NewEthereumClient(accounts[0], config)
if err != nil {
	log.Fatal(err)
}
defer service.Close()
```

### Complete Transaction Example

```go
package main

import (
	"fmt"
	"log"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
	"github.com/nando-os/gostheth/pkg"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Load configuration
	config, err := pkg.NewConfiguration()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Get accounts
	accounts := config.Accounts()
	if len(accounts) < 2 {
		log.Fatal("Need at least 2 accounts")
	}

	// Create service
	service, err := pkg.NewEthereumClient(accounts[0], config)
	if err != nil {
		log.Fatal("Failed to create service:", err)
	}
	defer service.Close()

	// Create transaction
	tx := &pkg.Transaction{
		From:  accounts[0].Address,
		To:    accounts[1].Address,
		Value: big.NewInt(1e15), // 0.001 ETH
		Data:  []byte{},         // Simple ETH transfer
	}

	// Sign transaction
	signedTx, err := service.SignTransaction(tx)
	if err != nil {
		log.Fatal("Failed to sign transaction:", err)
	}

	// Send transaction
	receipt, err := service.SendTransaction(signedTx)
	if err != nil {
		log.Fatal("Failed to send transaction:", err)
	}

	fmt.Printf("Transaction sent! Hash: %s\n", receipt.TxHash.Hex())

	// Wait for confirmation
	confirmedReceipt, err := service.WaitForTransaction(receipt.TxHash)
	if err != nil {
		log.Fatal("Transaction failed:", err)
	}

	if confirmedReceipt.Status == 1 {
		fmt.Println("✅ Transaction successful!")
	} else {
		fmt.Println("❌ Transaction failed!")
	}
}
```

## Gas Fee Strategy

The client wrapper automatically calculates optimal gas fees:

### EIP-1559 Networks (Ethereum Mainnet, Base)
- **Priority Fee**: Network-specific defaults (2 gwei mainnet, 1 gwei Base, 1.5 gwei others)
- **Max Fee**: 2x base fee + priority fee
- **Configurable**: Override via environment variables

### Legacy Networks
- **Gas Price**: Network-suggested price
- **Configurable**: Override via environment variables

### Gas Limit Buffers
- **Simple Transfers**: Configurable buffer (default: 10%)
- **Complex Transactions**: Higher buffer (default: 20%)
- **Validation**: Prevents exceeding block gas limits

## TOR Proxy Support

For enhanced privacy, the client wrapper supports TOR proxy:

### Setup TOR
```bash
# Using Docker
docker run -d --name tor-proxy -p 9050:9050 dperson/torproxy:latest

# Or install locally
sudo apt-get install tor  # Ubuntu/Debian
brew install tor          # macOS
```

### Configure Client
```bash
# Set environment variables
export HTTP_PROXY=socks5://127.0.0.1:9050
export HTTPS_PROXY=socks5://127.0.0.1:9050

# All HTTP/HTTPS traffic will go through TOR
```

### Test TOR Connection
```bash
# Verify TOR is working
curl --socks5 127.0.0.1:9050 https://check.torproject.org/
```

## Error Handling

The client wrapper provides comprehensive error handling:

```go
config, err := pkg.NewConfiguration()
if err != nil {
	// Handle configuration errors
	log.Fatal("Configuration error:", err)
}

service, err := pkg.NewEthereumClient(accounts[0], config)
if err != nil {
	// Handle service creation errors
	log.Fatal("Service creation error:", err)
}
defer service.Close()

signedTx, err := service.SignTransaction(tx)
if err != nil {
	// Handle transaction signing errors
	switch {
	case strings.Contains(err.Error(), "insufficient funds"):
		log.Fatal("Not enough ETH for transaction")
	case strings.Contains(err.Error(), "gas limit"):
		log.Fatal("Gas limit too high")
	default:
		log.Fatal("Transaction signing failed:", err)
	}
}
```

## Best Practices

### Security
- **Environment Variables**: Store private keys securely
- **Key Rotation**: Regularly rotate private keys
- **TOR Usage**: Use TOR for sensitive transactions
- **Validation**: Always validate transaction parameters

### Performance
- **Connection Pooling**: Reuse service instances
- **Gas Optimization**: Use appropriate gas buffers
- **Error Handling**: Implement proper retry logic
- **Monitoring**: Track transaction success rates

### Configuration
- **Network-Specific**: Use appropriate settings for each network
- **Testing**: Test configurations on testnets first
- **Documentation**: Document your configuration choices
- **Backup**: Keep backup configurations

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/nando-os/gostheth/issues)
- **Examples**: [Examples Directory](examples/)
