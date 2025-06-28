package pkg

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

const (
	envRpcURL  = "ETH_RPC_URL"
	envChainID = "ETH_CHAIN_ID"

	// -- accounts and private keys
	envAccountsList         = "ETH_ACCOUNTS"
	envAccountPrivateKeyFmt = "ETH_ACCOUNT_%s_PRIVATE_KEY"
	envAccountPublicKeyFmt  = "ETH_ACCOUNT_%s_PUBLIC_KEY"

	// -- gas configuration
	// Recommended settings:
	// Development/Testing:
	//   ETH_GAS_LIMIT_BUFFER_SIMPLE=1.2    # Higher buffers for testing
	//   ETH_GAS_LIMIT_BUFFER_COMPLEX=1.4
	// Production - Base:
	//   ETH_GAS_LIMIT_BUFFER_SIMPLE=1.05   # Lower costs, faster blocks
	//   ETH_GAS_LIMIT_BUFFER_COMPLEX=1.15
	// Production - Ethereum Mainnet:
	//   ETH_GAS_LIMIT_BUFFER_SIMPLE=1.1    # Higher costs, more conservative
	//   ETH_GAS_LIMIT_BUFFER_COMPLEX=1.25
	envGasLimitBufferSimple  = "1.2" // Buffer for simple ETH transfers
	envGasLimitBufferComplex = "1.4" // Buffer for complex transactions

	// -- fee configuration
	// Max fee per gas in wei (default: 500 gwei)
	envMaxFeePerGas = "ETH_MAX_FEE_PER_GAS"
	// Priority fee per gas in wei (network-specific, defaults: 2 gwei for mainnet, 1 gwei for Base, 1.5 gwei for others)
	envPriorityFeeMainnet = "ETH_PRIORITY_FEE_MAINNET"
	envPriorityFeeBase    = "ETH_PRIORITY_FEE_BASE"
	envPriorityFeeDefault = "ETH_PRIORITY_FEE_DEFAULT"

	// --- Units and defaults ---
	GWEI = 1000000000 // 1 gwei in wei

	DEFAULT_PRIORITY_FEE_MAINNET = 2 * GWEI       // 2 gwei
	DEFAULT_PRIORITY_FEE_BASE    = 1 * GWEI       // 1 gwei
	DEFAULT_PRIORITY_FEE_OTHER   = 15 * GWEI / 10 // 1.5 gwei
	DEFAULT_MAX_FEE_PER_GAS      = 500 * GWEI     // 500 gwei

	// --- Transaction monitoring defaults ---
	DEFAULT_TRANSACTION_TIMEOUT_SECONDS = 300 // 5 minutes
	DEFAULT_TRANSACTION_TICKER_SECONDS  = 3   // 3 seconds
)

type config struct {
	chainId int64
	acounts []*Account
	rpcURL  string
}

func NewConfiguration() (*config, error) {

	chainIDStr := os.Getenv(envChainID)
	if chainIDStr == "" {
		return nil, fmt.Errorf(envChainID + " environment variable is not set")
	}

	chainId, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ETH_CHAIN_ID: %w", err)
	}

	accounts, err := loadAccountsFromEnv(chainId)
	if err != nil {
		return nil, fmt.Errorf("failed to load accounts: %w", err)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts found in %s environment variable", envAccountsList)
	}

	rpcURL := os.Getenv(envRpcURL)

	return &config{
		rpcURL:  rpcURL,
		chainId: chainId,
		acounts: accounts,
	}, nil
}

func (c *config) ChainID() int64 {
	return c.chainId
}
func (c *config) Accounts() []*Account {
	return c.acounts
}

func (c *config) RPCURL() string {
	return c.rpcURL
}

// GasLimitBufferSimple returns the buffer multiplier for simple ETH transfers
func (c *config) GasLimitBufferSimple() float64 {
	bufferStr := os.Getenv(envGasLimitBufferSimple)
	if bufferStr == "" {
		return 1.1 // Default 10% buffer for simple transfers
	}

	buffer, err := strconv.ParseFloat(bufferStr, 64)
	if err != nil {
		return 1.1 // Fallback to default on parse error
	}

	// Validate reasonable bounds (0.5 to 3.0)
	if buffer < 0.5 || buffer > 3.0 {
		return 1.1
	}

	return buffer
}

// GasLimitBufferComplex returns the buffer multiplier for complex transactions
func (c *config) GasLimitBufferComplex() float64 {
	bufferStr := os.Getenv(envGasLimitBufferComplex)
	if bufferStr == "" {
		return 1.2 // Default 20% buffer for complex transactions
	}

	buffer, err := strconv.ParseFloat(bufferStr, 64)
	if err != nil {
		return 1.2 // Fallback to default on parse error
	}

	// Ennsure reasonable bounds (0.5 to 3.0)
	if buffer < 0.5 || buffer > 3.0 {
		return 1.2
	}

	return buffer
}

// MaxFeePerGas returns the max fee per gas in wei (default: 500 gwei)
func (c *config) MaxFeePerGas() *big.Int {
	maxFeeStr := os.Getenv(envMaxFeePerGas)
	if maxFeeStr == "" {
		return big.NewInt(DEFAULT_MAX_FEE_PER_GAS)
	}
	maxFee, ok := new(big.Int).SetString(maxFeeStr, 10)
	if !ok {
		return big.NewInt(DEFAULT_MAX_FEE_PER_GAS)
	}
	return maxFee
}

// PriorityFeeMainnet returns the fixed priority fee for Ethereum mainnet (default: 2 gwei)
func (c *config) PriorityFeeMainnet() *big.Int {
	feeStr := os.Getenv(envPriorityFeeMainnet)
	if feeStr == "" {
		return big.NewInt(DEFAULT_PRIORITY_FEE_MAINNET)
	}
	fee, ok := new(big.Int).SetString(feeStr, 10)
	if !ok {
		return big.NewInt(DEFAULT_PRIORITY_FEE_MAINNET)
	}
	return fee
}

// PriorityFeeBase returns the fixed priority fee for Base (default: 1 gwei)
func (c *config) PriorityFeeBase() *big.Int {
	feeStr := os.Getenv(envPriorityFeeBase)
	if feeStr == "" {
		return big.NewInt(DEFAULT_PRIORITY_FEE_BASE)
	}
	fee, ok := new(big.Int).SetString(feeStr, 10)
	if !ok {
		return big.NewInt(DEFAULT_PRIORITY_FEE_BASE)
	}
	return fee
}

// PriorityFeeDefault returns the fixed priority fee for other networks (default: 1.5 gwei)
func (c *config) PriorityFeeDefault() *big.Int {
	feeStr := os.Getenv(envPriorityFeeDefault)
	if feeStr == "" {
		return big.NewInt(DEFAULT_PRIORITY_FEE_OTHER)
	}
	fee, ok := new(big.Int).SetString(feeStr, 10)
	if !ok {
		return big.NewInt(DEFAULT_PRIORITY_FEE_OTHER)
	}
	return fee
}

func loadAccountsFromEnv(chainID int64) ([]*Account, error) {
	var accounts []*Account
	accountLabels := os.Getenv(envAccountsList)
	if accountLabels == "" {
		return nil, fmt.Errorf("ETH_ACCOUNTS env variable not set")
	}
	labels := strings.Split(accountLabels, ",")
	for _, label := range labels {
		label = strings.TrimSpace(label)

		keyEnv := fmt.Sprintf(envAccountPrivateKeyFmt, strings.ToUpper(label))
		privHex := os.Getenv(keyEnv)

		pubkeyEnv := fmt.Sprintf(envAccountPublicKeyFmt, strings.ToUpper(label))
		pubHex := os.Getenv(pubkeyEnv)

		// -- validate
		// if both private and public keys are provided, they must match
		if privHex == "" && pubHex == "" {
			return nil, fmt.Errorf("no private or public key found for account[%s] in environment variables", label)
		}
		var account *Account
		if privHex != "" {
			// create account based on private key
			privKey, err := crypto.HexToECDSA(privHex)
			if err != nil {
				return nil, fmt.Errorf("invalid private key for %s: %w", label, err)
			}
			pubKey := privKey.Public().(*ecdsa.PublicKey)
			address := crypto.PubkeyToAddress(*pubKey)
			account = &Account{
				Address:    address,
				PublicKey:  pubKey,
				ChainId:    chainID,
				Label:      label,
				PrivateKey: privKey,
			}
			// continue to next account if account has been created
			accounts = append(accounts, account)
			continue
		}

		if pubHex != "" {
			// create account based on public key
			// -- this typically happens when the private key is not available
			// -- but the public key is known (e.g., for read-only accounts)
			// -- this type of account can be used for receiving funds or verifying signatures
			pubKey, err := crypto.UnmarshalPubkey([]byte(pubHex))
			if err != nil {
				return nil, fmt.Errorf("invalid public key for %s: %w", label, err)
			}
			address := crypto.PubkeyToAddress(*pubKey)
			account = &Account{
				Address:   address,
				PublicKey: pubKey,
				ChainId:   chainID,
				Label:     label,
			}
			// continue to next account if account has been created
			accounts = append(accounts, account)
			continue
		}
		return nil, fmt.Errorf("no private or public key found for account[%s] in environment variables", label)
	}
	return accounts, nil
}

// Account represents an Ethereum account with its address, public key, chain ID, and an optional label.

// TransactionTimeoutSeconds returns the transaction timeout in seconds (default: 300)
func (c *config) TransactionTimeoutSeconds() int {
	timeoutStr := os.Getenv("ETH_TRANSACTION_TIMEOUT_SECONDS")
	if timeoutStr == "" {
		return DEFAULT_TRANSACTION_TIMEOUT_SECONDS
	}
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil || timeout <= 0 {
		return DEFAULT_TRANSACTION_TIMEOUT_SECONDS
	}
	return timeout
}

// TransactionTickerSeconds returns the transaction ticker interval in seconds (default: 3)
func (c *config) TransactionTickerSeconds() int {
	tickerStr := os.Getenv("ETH_TRANSACTION_TICKER_SECONDS")
	if tickerStr == "" {
		return DEFAULT_TRANSACTION_TICKER_SECONDS
	}
	ticker, err := strconv.Atoi(tickerStr)
	if err != nil || ticker <= 0 {
		return DEFAULT_TRANSACTION_TICKER_SECONDS
	}
	return ticker
}
