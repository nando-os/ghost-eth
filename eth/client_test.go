package eth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	internalmocks "github.com/nando-os/gostheth/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testAccountAndConfig() (*Account, *config) {
	accs := []*Account{
		{
			Address:    common.HexToAddress("0x0000000000000000000000000000000000000001"),
			ChainId:    1,
			Label:      "main",
			PrivateKey: &ecdsa.PrivateKey{}, // dummy, not used for real signing
		},
	}
	cfg := &config{chainId: 1, acounts: accs, rpcURL: "http://localhost:8545"}
	return accs[0], cfg
}

func TestGhostClient_GetBalance(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	wantBalance := big.NewInt(42)
	mockClient.On("BalanceAt", mock.Anything, acc.Address, (*big.Int)(nil)).Return(wantBalance, nil)
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	bal, err := gc.GetBalance(acc.Address)
	assert.NoError(t, err)
	assert.Equal(t, wantBalance, bal)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_GetBalance_Error(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	mockClient.On("BalanceAt", mock.Anything, acc.Address, (*big.Int)(nil)).Return(nil, errors.New("fail"))
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	_, err := gc.GetBalance(acc.Address)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_Close(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	mockClient.On("Close").Return()
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	gc.Close()
	mockClient.AssertExpectations(t)
}

func TestGhostClient_EstimateGasAndSetLimit_Simple(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	// Simulate EstimateGas returns 21000, block gas limit is 30000000
	mockClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(21000), nil)
	header := &types.Header{GasLimit: 30000000}
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(header, nil)
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
		Data: []byte{}, // simple
	}
	err := gc.estimateGasAndSetLimit(tx)
	assert.NoError(t, err)
	// Default buffer for simple is 1.1, so expect 21000*1.1 = 23100
	assert.Equal(t, uint64(23100), tx.GasLimit)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_EstimateGasAndSetLimit_Complex(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	mockClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(50000), nil)
	header := &types.Header{GasLimit: 30000000}
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(header, nil)
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
		Data: []byte{1, 2, 3}, // complex
	}
	err := gc.estimateGasAndSetLimit(tx)
	assert.NoError(t, err)
	// Default buffer for complex is 1.2, so expect 50000*1.2 = 60000
	assert.Equal(t, uint64(60000), tx.GasLimit)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_EstimateGasAndSetLimit_Errors(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	// Simulate EstimateGas error
	mockClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(0), errors.New("fail estimate"))
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
	}
	err := gc.estimateGasAndSetLimit(tx)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)

	// Simulate gas limit too high
	mockClient = &internalmocks.EthClient{}
	mockClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(10000000), nil)
	header := &types.Header{GasLimit: 12000000}
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(header, nil)
	gc.client = mockClient
	tx = &Transaction{
		From: acc.Address,
		To:   acc.Address,
	}
	err = gc.estimateGasAndSetLimit(tx)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_CalculateOptimalFees_EIP1559(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	// Simulate EIP-1559 header
	header := &types.Header{BaseFee: big.NewInt(100)}
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(header, nil)
	// Priority fee for mainnet is 2 gwei
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
	}
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	err := gc.calculateOptimalFees(tx)
	assert.NoError(t, err)
	assert.Equal(t, cfg.PriorityFeeMainnet(), tx.MaxPriorityFeePerGas)
	// MaxFeePerGas should be 2*baseFee + priorityFee
	expectedMaxFee := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
	expectedMaxFee.Add(expectedMaxFee, cfg.PriorityFeeMainnet())
	assert.Equal(t, expectedMaxFee, tx.MaxFeePerGas)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_CalculateOptimalFees_Legacy(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	header := &types.Header{BaseFee: nil}
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(header, nil)
	mockClient.On("SuggestGasPrice", mock.Anything).Return(big.NewInt(12345), nil)
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
	}
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	err := gc.calculateOptimalFees(tx)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(12345), tx.GasPrice)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_CalculateOptimalFees_HeaderError(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(nil, errors.New("fail header"))
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
	}
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	err := gc.calculateOptimalFees(tx)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_CalculateOptimalFees_GasPriceError(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	header := &types.Header{BaseFee: nil}
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(header, nil)
	mockClient.On("SuggestGasPrice", mock.Anything).Return(nil, errors.New("fail gas price"))
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
	}
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	err := gc.calculateOptimalFees(tx)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_CalculateOptimalFees_MaxFeeTooHigh(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	header := &types.Header{BaseFee: big.NewInt(1e18)} // very high base fee
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(header, nil)
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
	}
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	err := gc.calculateOptimalFees(tx)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_GetTransactionReceipt_Success(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	hash := common.HexToHash("0xabc")
	receipt := &types.Receipt{
		TxHash:      hash,
		Status:      1,
		BlockNumber: big.NewInt(123),
		GasUsed:     21000,
		Logs:        []*types.Log{},
	}
	tx := &types.Transaction{}
	to := common.HexToAddress("0x0000000000000000000000000000000000000002")
	mockClient.On("TransactionReceipt", mock.Anything, hash).Return(receipt, nil)
	mockClient.On("TransactionByHash", mock.Anything, hash).Return(tx, true, nil)
	tx.To = func() *common.Address { return &to }
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	r, err := gc.GetTransactionReceipt(hash)
	assert.NoError(t, err)
	assert.Equal(t, hash, r.TxHash)
	assert.Equal(t, uint64(123), r.BlockNumber)
	assert.Equal(t, acc.Address, r.From)
	assert.Equal(t, to, r.To)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_GetTransactionReceipt_Error(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	hash := common.HexToHash("0xabc")
	mockClient.On("TransactionReceipt", mock.Anything, hash).Return(nil, errors.New("not found"))
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	_, err := gc.GetTransactionReceipt(hash)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_SignTransaction_EIP1559_Success(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	// Nonce
	mockClient.On("PendingNonceAt", mock.Anything, acc.Address).Return(uint64(7), nil)
	// Gas estimation
	mockClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(21000), nil)
	header := &types.Header{GasLimit: 30000000, BaseFee: big.NewInt(100)}
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(header, nil)
	// Fee calculation
	// No need to mock SuggestGasPrice for EIP-1559
	tx := &Transaction{
		From:  acc.Address,
		To:    acc.Address,
		Value: big.NewInt(1e18),
		Data:  []byte{},
	}
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	// Patch types.SignTx to avoid real signing (not needed for this test)
	// We'll just check that no error is returned and fields are set
	result, err := gc.SignTransaction(tx)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(7), tx.Nonce)
	assert.NotZero(t, tx.GasLimit)
	assert.NotNil(t, tx.MaxFeePerGas)
	assert.NotNil(t, tx.MaxPriorityFeePerGas)
	mockClient.AssertExpectations(t)
}

func TestGhostClient_SignTransaction_Errors(t *testing.T) {
	acc, cfg := testAccountAndConfig()
	mockClient := &internalmocks.EthClient{}
	gc := &ghostClient{
		client:  mockClient,
		ctx:     context.Background(),
		chainId: 1,
		account: acc,
		config:  cfg,
	}
	tx := &Transaction{
		From: acc.Address,
		To:   acc.Address,
	}
	// Nonce error
	mockClient.On("PendingNonceAt", mock.Anything, acc.Address).Return(uint64(0), errors.New("fail nonce")).Once()
	_, err := gc.SignTransaction(tx)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)

	// Gas estimation error
	mockClient = &internalmocks.EthClient{}
	mockClient.On("PendingNonceAt", mock.Anything, acc.Address).Return(uint64(1), nil)
	mockClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(0), errors.New("fail gas")).Once()
	gc.client = mockClient
	tx = &Transaction{From: acc.Address, To: acc.Address}
	_, err = gc.SignTransaction(tx)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)

	// Fee error (simulate header error)
	mockClient = &internalmocks.EthClient{}
	mockClient.On("PendingNonceAt", mock.Anything, acc.Address).Return(uint64(2), nil)
	mockClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(21000), nil)
	mockClient.On("HeaderByNumber", mock.Anything, (*big.Int)(nil)).Return(nil, errors.New("fail header")).Once()
	gc.client = mockClient
	tx = &Transaction{From: acc.Address, To: acc.Address}
	_, err = gc.SignTransaction(tx)
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}
