package asset

import (
	"encoding/json"
	"time"

	"encrypted-db/internal/domain/shared"
)

// Asset represents a tradable asset (fiat, crypto, token, stablecoin)
type Asset struct {
	ID              shared.UUID       `json:"id" db:"id"`
	Symbol          string            `json:"symbol" db:"symbol"`                           // e.g., "BTC", "USDT", "USD"
	Name            string            `json:"name" db:"name"`                               // e.g., "Bitcoin", "Tether USD"
	Type            shared.AssetType  `json:"type" db:"type"`                               // FIAT, COIN, TOKEN, STABLE, WRAPPED
	Status          shared.AssetStatus `json:"status" db:"status"`                          // ACTIVE, INACTIVE, etc.
	Decimals        int               `json:"decimals" db:"decimals"`                       // Precision (8 for BTC, 6 for USDT, 2 for USD)
	
	// Blockchain info (for crypto assets)
	Network         string            `json:"network,omitempty" db:"network"`               // e.g., "bitcoin", "ethereum", "tron", "solana"
	ContractAddress string            `json:"contract_address,omitempty" db:"contract_address"` // For tokens
	NativeSymbol    string            `json:"native_symbol,omitempty" db:"native_symbol"`   // Native chain symbol (e.g., "TRX" for Tron)
	
	// Display & metadata
	Description     string            `json:"description,omitempty" db:"description"`
	LogoURL         string            `json:"logo_url,omitempty" db:"logo_url"`
	Website         string            `json:"website,omitempty" db:"website"`
	ExplorerURL     string            `json:"explorer_url,omitempty" db:"explorer_url"`
	
	// Trading config
	MinWithdrawal   shared.Decimal    `json:"min_withdrawal" db:"min_withdrawal"`
	MaxWithdrawal   shared.Decimal    `json:"max_withdrawal" db:"max_withdrawal"`
	WithdrawalFee   shared.Decimal    `json:"withdrawal_fee" db:"withdrawal_fee"`
	DepositEnabled  bool              `json:"deposit_enabled" db:"deposit_enabled"`
	WithdrawalEnabled bool            `json:"withdrawal_enabled" db:"withdrawal_enabled"`
	
	// Custody config
	CustodyMode     shared.SettlementMode `json:"custody_mode" db:"custody_mode"` // OFF_CHAIN, ON_CHAIN
	RequiresTag     bool              `json:"requires_tag" db:"requires_tag"`     // For XRP, XLM, etc.
	
	// Compliance
	SanctionsScreened bool            `json:"sanctions_screened" db:"sanctions_screened"`
	LastScreenedAt    *time.Time      `json:"last_screened_at,omitempty" db:"last_screened_at"`
	
	// Timestamps
	CreatedAt       shared.Timestamp  `json:"created_at" db:"created_at"`
	UpdatedAt       shared.Timestamp  `json:"updated_at" db:"updated_at"`
	DeletedAt       *shared.Timestamp `json:"deleted_at,omitempty" db:"deleted_at"`
}

// AssetConfig holds chain-specific configuration for custody providers
type AssetConfig struct {
	AssetID           shared.UUID       `json:"asset_id"`
	ChainID           string            `json:"chain_id"`             // e.g., "1" for Ethereum, "tron" for Tron
	RPCEndpoints      []string          `json:"rpc_endpoints"`        // Multiple for failover
	ExplorerAPI       string            `json:"explorer_api"`         // For tx indexing
	ConfirmationsRequired int          `json:"confirmations_required"` // Blocks to wait
	
	// Gas/fee config
	GasLimit          uint64            `json:"gas_limit,omitempty"`  // For EVM
	GasPriceStrategy  string            `json:"gas_price_strategy"`   // "fixed", "eip1559", "oracle"
	FeeAssetID        *shared.UUID      `json:"fee_asset_id,omitempty"` // For paying fees in different asset
	
	// Deposit/withdrawal
	DepositContractAddress string       `json:"deposit_contract_address,omitempty"` // For token deposits
	WithdrawalMethod    string            `json:"withdrawal_method"`    // "native", "contract", "batch"
	MinConfirmations    int               `json:"min_confirmations"`
	
	// Tags/memo
	RequiresTag         bool              `json:"requires_tag"`
	TagType             string            `json:"tag_type,omitempty"`   // "memo", "tag", "payment_id"
	
	Metadata            json.RawMessage   `json:"metadata,omitempty"`   // Chain-specific extras
	
	CreatedAt           shared.Timestamp  `json:"created_at"`
	UpdatedAt           shared.Timestamp  `json:"updated_at"`
}

// AssetPair represents a trading pair (base/quote)
type AssetPair struct {
	BaseAssetID  shared.UUID `json:"base_asset_id" db:"base_asset_id"`
	QuoteAssetID shared.UUID `json:"quote_asset_id" db:"quote_asset_id"`
	
	// Computed fields (not stored)
	BaseAsset  *Asset `json:"base_asset,omitempty"`
	QuoteAsset *Asset `json:"quote_asset,omitempty"`
}

// Symbol returns the standard trading symbol (e.g., "BTC/USDT")
func (p *AssetPair) Symbol() string {
	base := "UNKNOWN"
	quote := "UNKNOWN"
	if p.BaseAsset != nil {
		base = p.BaseAsset.Symbol
	}
	if p.QuoteAsset != nil {
		quote = p.QuoteAsset.Symbol
	}
	return base + "/" + quote
}

// IsValid checks if the pair is valid for trading
func (p *AssetPair) IsValid() bool {
	return p.BaseAssetID != shared.UUID{} && p.QuoteAssetID != shared.UUID{} && p.BaseAssetID != p.QuoteAssetID
}