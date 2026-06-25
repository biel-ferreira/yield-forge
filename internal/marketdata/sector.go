package marketdata

import "strings"

// Sector is the FII segment classification the dashboard reports (PRD FR-005 / Epic 3).
// It is a closed enum; an unrecognized provider label normalizes to SectorOther so a new
// or oddly-spelled segment never breaks ingestion (SPEC-006 §6, edge cases).
type Sector string

const (
	SectorLogistics Sector = "logistics"
	SectorOffices   Sector = "offices"
	SectorShopping  Sector = "shopping"
	SectorHybrid    Sector = "hybrid"
	SectorPaper     Sector = "paper"
	SectorOther     Sector = "other"
)

// sectorKeywords maps substrings found in provider segment labels (e.g. Fundamentus
// "Segmento", in PT-BR) to a canonical Sector. Order matters only when a label could
// contain two keywords; the lists are disjoint in practice.
var sectorKeywords = []struct {
	needle string
	sector Sector
}{
	{"logist", SectorLogistics},   // Logística
	{"laje", SectorOffices},       // Lajes Corporativas
	{"corporativ", SectorOffices}, // Lajes Corporativas
	{"escrit", SectorOffices},     // Escritórios
	{"office", SectorOffices},     //
	{"shopping", SectorShopping},  // Shoppings
	{"varejo", SectorShopping},    // Varejo
	{"hibrid", SectorHybrid},      // Híbrido (accent-folded)
	{"hybrid", SectorHybrid},      //
	{"papel", SectorPaper},        // Papel
	{"receb", SectorPaper},        // Recebíveis
	{"titulo", SectorPaper},       // Títulos e Val. Mob. (accent-folded)
	{"cri", SectorPaper},          // CRI
	{"paper", SectorPaper},        //
}

// ParseSector maps a raw provider segment label to a canonical Sector, falling back to
// SectorOther for anything unrecognized. Matching is accent-insensitive.
func ParseSector(raw string) Sector {
	s := foldAccents(strings.ToLower(strings.TrimSpace(raw)))
	for _, k := range sectorKeywords {
		if strings.Contains(s, k.needle) {
			return k.sector
		}
	}
	return SectorOther
}

// foldAccents strips the Portuguese accents that appear in segment labels so keyword
// matching is robust without a Unicode-normalization dependency (stdlib-first, ADR-0003).
func foldAccents(s string) string {
	return accentReplacer.Replace(s)
}

var accentReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a",
	"é", "e", "ê", "e",
	"í", "i",
	"ó", "o", "ô", "o", "õ", "o",
	"ú", "u",
	"ç", "c",
)
