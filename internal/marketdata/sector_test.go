package marketdata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSector(t *testing.T) {
	cases := map[string]Sector{
		"Logística":           SectorLogistics,
		"logistica":           SectorLogistics,
		"Lajes Corporativas":  SectorOffices,
		"Escritórios":         SectorOffices,
		"Shoppings":           SectorShopping,
		"Híbrido":             SectorHybrid,
		"Papel":               SectorPaper,
		"Recebíveis":          SectorPaper,
		"Títulos e Val. Mob.": SectorPaper,
		"CRI":                 SectorPaper,
		"Algo Desconhecido":   SectorOther, // unknown -> Other (never breaks ingestion)
		"":                    SectorOther,
	}
	for raw, want := range cases {
		require.Equal(t, want, ParseSector(raw), "segment %q", raw)
	}
}
