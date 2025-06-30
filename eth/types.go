package eth

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

// High-level Ethereum types and structures, for application-specific use
type Account struct {
	Address    common.Address    // Ethereum adress
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
