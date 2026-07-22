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

// ClassSlice is one asset class's current value and its share of the total (bps). Invested/
// Growth (SPEC-110 FR-1104) are 0 for Stocks/ETFs (always-zero classes in the MVP).
type ClassSlice struct {
	Class            AssetClass
	ValueCentavos    int64
	ShareBps         int
	InvestedCentavos int64 // new (SPEC-110): this class's own cost basis
	GrowthCentavos   int64 // new (SPEC-110): ValueCentavos - InvestedCentavos
	GrowthBps        int   // new (SPEC-110): growth relative to InvestedCentavos
}

// SectorSlice is one FII sector's current value and its share of the FII total (bps).
type SectorSlice struct {
	Sector        marketdata.Sector
	ValueCentavos int64
	ShareBps      int
}

// FIIHoldingSlice is one held FII ticker's current value and its share of the FII total (bps) —
// new in SPEC-110 FR-1109, mirrors SectorSlice's value+share shape at holding granularity.
type FIIHoldingSlice struct {
	Ticker        marketdata.Ticker
	ValueCentavos int64
	ShareBps      int
}

// FixedIncomeHoldingSlice is one fixed-income holding's current value, growth, and share of the
// FI total (bps) — new in SPEC-110 FR-1109. ID+Name (not Name alone) because two holdings can
// share a display name (e.g. two CDBs both named "CDB Nubank" opened months apart).
type FixedIncomeHoldingSlice struct {
	ID             string
	Name           string
	ValueCentavos  int64
	GrowthCentavos int64
	ShareBps       int
}

// Dashboard is the full computed view returned for one user (SPEC-103 §6). StaleTickers lists
// the held FIIs that had no current quote and were valued at cost basis instead (FR-1036).
// FixedIncomeReconciliationDue/NeedsAttention/FIIHoldings/FixedIncomeHoldings are new in SPEC-110.
type Dashboard struct {
	Summary                      Summary
	Allocation                   []ClassSlice
	FIISectors                   []SectorSlice
	StaleTickers                 []string
	FixedIncomeReconciliationDue []string                  // FI holdings due for reconciliation (FR-1105)
	NeedsAttention               bool                      // StaleTickers or FixedIncomeReconciliationDue non-empty
	FIIHoldings                  []FIIHoldingSlice         // per-ticker breakdown (FR-1109)
	FixedIncomeHoldings          []FixedIncomeHoldingSlice // per-holding breakdown (FR-1109)
}
