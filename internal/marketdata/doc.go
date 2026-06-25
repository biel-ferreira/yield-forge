// Package marketdata is the Market Data feature (SPEC-006): the MarketDataProvider port
// and the domain it ingests — FII quotes (price, dividend yield, P/VP, sector, last
// dividend; FR-006) and macro indicators (SELIC, IPCA, CDI, IFIX; FR-007).
//
// The package core is pure: it depends on no HTTP, SQL, or vendor SDK (BR-601). Provider
// adapters (Fundamentus/Yahoo for FIIs, BCB-SGS for macro) and the Postgres repositories
// live in subpackages at the edge and implement the ports defined here. Market data is
// global reference data — there is no per-user scoping anywhere in this package (BR-603).
package marketdata
