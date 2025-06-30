package eth

import (
	"math/big"
	"os"
	"testing"
)

func TestNewConfiguration_Success(t *testing.T) {
	os.Setenv("ETH_CHAIN_ID", "1234")
	os.Setenv("ETH_ACCOUNTS", "main")
	os.Setenv("ETH_ACCOUNT_MAIN_PRIVATE_KEY", "4f3edf983ac636a65a842ce7c78d9aa706d3b113b37e5a4d5e1e4e6a1f7a1e08") // test key
	os.Setenv("ETH_RPC_URL", "http://localhost:8545")
	defer os.Clearenv()

	cfg, err := NewConfiguration()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.ChainID() != 1234 {
		t.Errorf("expected chain ID 1234, got %d", cfg.ChainID())
	}
	if len(cfg.Accounts()) != 1 {
		t.Errorf("expected 1 account, got %d", len(cfg.Accounts()))
	}
	if cfg.RPCURL() != "http://localhost:8545" {
		t.Errorf("expected RPC URL http://localhost:8545, got %s", cfg.RPCURL())
	}
}

func TestNewConfiguration_MissingEnv(t *testing.T) {
	os.Clearenv()
	_, err := NewConfiguration()
	if err == nil {
		t.Fatal("expected error for missing env vars, got nil")
	}
}

func TestGasLimitBufferDefaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("ETH_CHAIN_ID", "1")
	os.Setenv("ETH_ACCOUNTS", "main")
	os.Setenv("ETH_ACCOUNT_MAIN_PRIVATE_KEY", "4f3edf983ac636a65a842ce7c78d9aa706d3b113b37e5a4d5e1e4e6a1f7a1e08")
	cfg, err := NewConfiguration()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.GasLimitBufferSimple() != 1.1 {
		t.Errorf("expected default simple buffer 1.1, got %f", cfg.GasLimitBufferSimple())
	}
	if cfg.GasLimitBufferComplex() != 1.2 {
		t.Errorf("expected default complex buffer 1.2, got %f", cfg.GasLimitBufferComplex())
	}
}

func TestFeeConfigDefaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("ETH_CHAIN_ID", "1")
	os.Setenv("ETH_ACCOUNTS", "main")
	os.Setenv("ETH_ACCOUNT_MAIN_PRIVATE_KEY", "4f3edf983ac636a65a842ce7c78d9aa706d3b113b37e5a4d5e1e4e6a1f7a1e08")
	cfg, err := NewConfiguration()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.MaxFeePerGas().Cmp(big.NewInt(500000000000)) != 0 {
		t.Errorf("expected default max fee per gas 500000000000, got %s", cfg.MaxFeePerGas().String())
	}
	if cfg.PriorityFeeMainnet().Cmp(big.NewInt(2000000000)) != 0 {
		t.Errorf("expected default mainnet priority fee 2000000000, got %s", cfg.PriorityFeeMainnet().String())
	}
	if cfg.PriorityFeeBase().Cmp(big.NewInt(1000000000)) != 0 {
		t.Errorf("expected default base priority fee 1000000000, got %s", cfg.PriorityFeeBase().String())
	}
	if cfg.PriorityFeeDefault().Cmp(big.NewInt(1500000000)) != 0 {
		t.Errorf("expected default default priority fee 1500000000, got %s", cfg.PriorityFeeDefault().String())
	}
}

func TestTransactionTimeoutDefaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("ETH_CHAIN_ID", "1")
	os.Setenv("ETH_ACCOUNTS", "main")
	os.Setenv("ETH_ACCOUNT_MAIN_PRIVATE_KEY", "4f3edf983ac636a65a842ce7c78d9aa706d3b113b37e5a4d5e1e4e6a1f7a1e08")
	cfg, err := NewConfiguration()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.TransactionTimeoutSeconds() != 300 {
		t.Errorf("expected default timeout 300, got %d", cfg.TransactionTimeoutSeconds())
	}
	if cfg.TransactionTickerSeconds() != 3 {
		t.Errorf("expected default ticker 3, got %d", cfg.TransactionTickerSeconds())
	}
}
