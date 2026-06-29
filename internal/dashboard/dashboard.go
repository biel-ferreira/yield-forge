package dashboard

import "github.com/biel-ferreira/yield-forge/internal/marketdata"

// AssetClass is a managed allocation bucket (SPEC-103 FR-1032). Stocks and ETFs are present
// for forward-compatibility but are always zero in the MVP (no managed holdings).
type AssetClass string

const (
	ClassFII         AssetClass = "fii"
	ClassFixedIncome AssetClass = "fixed_income"
	ClassStocks      AssetClass = "stocks"
	ClassETFs        AssetClass = "etfs"
)

// AllAssetClasses is the fixed display order for the allocation breakdown.
var AllAssetClasses = []AssetClass{ClassFII, ClassFixedIncome, ClassStocks, ClassETFs}

// Summary is the portfolio summary (SPEC-103 FR-1031). All money is int64 centavos; growth is
// also expressed relatively in integer basis points (BR-1032).
type Summary struct {
	TotalInvestedCentavos int64 // cost basis: Σ FII (qty × avg price) + Σ FI invested amount
	CurrentValueCentavos  int64 // the full patrimony / net worth: Σ each holding's current value
	MonthlyIncomeCentavos int64 // Σ FII (last dividend × qty)
	GrowthCentavos        int64 // CurrentValue − TotalInvested
	GrowthBps             int   // growth relative to invested, basis points (0 when invested is 0)
}

// ClassSlice is one asset class's current value and its share of the total (bps).
type ClassSlice struct {
	Class         AssetClass
	ValueCentavos int64
	ShareBps      int
}

// SectorSlice is one FII sector's current value and its share of the FII total (bps).
type SectorSlice struct {
	Sector        marketdata.Sector
	ValueCentavos int64
	ShareBps      int
}

// Dashboard is the full computed view returned for one user (SPEC-103 §6). StaleTickers lists
// the held FIIs that had no current quote and were valued at cost basis instead (FR-1036).
type Dashboard struct {
	Summary      Summary
	Allocation   []ClassSlice
	FIISectors   []SectorSlice
	StaleTickers []string
}
