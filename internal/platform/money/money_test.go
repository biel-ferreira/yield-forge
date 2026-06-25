package money

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecimalToMinor(t *testing.T) {
	cases := []struct {
		in    string
		scale int
		want  int64
	}{
		// Brazilian forms (comma decimal, dot thousands).
		{"15,75", 2, 1575},      // price -> centavos
		{"1.234,56", 2, 123456}, // thousands sep stripped
		{"8,50", 2, 850},        // percent -> bps
		{"8,5", 2, 850},         // short fraction padded
		{"0,95", 4, 9500},       // P/VP ratio -> ratio bps
		{"10", 2, 1000},         // integer
		{"0", 2, 0},             //
		{"-2,50", 2, -250},      // negative
		// Plain forms (dot decimal — Yahoo json.Number).
		{"0.11", 2, 11},        //
		{"1234.56", 2, 123456}, //
		// Half-up rounding.
		{"1,005", 2, 101}, // 100.5 -> 101 (round up)
		{"1,004", 2, 100}, // 100.4 -> 100 (round down)
		{"2,5", 0, 3},     // round up at scale 0
		{"2,4", 0, 2},     //
	}
	for _, c := range cases {
		got, err := DecimalToMinor(c.in, c.scale)
		require.NoError(t, err, "input %q scale %d", c.in, c.scale)
		require.Equal(t, c.want, got, "input %q scale %d", c.in, c.scale)
	}
}

func TestDecimalToMinor_Invalid(t *testing.T) {
	for _, in := range []string{
		"", "abc", "1,2,3", "R$ 10", "1.2.3,4x",
		"-", "+", ".", // a lone sign / dot is not a number (must not parse to 0)
		"92233720368547758.075", // round-up carry would overflow int64
	} {
		_, err := DecimalToMinor(in, 2)
		require.ErrorIs(t, err, ErrInvalidDecimal, "should reject %q", in)
	}
}
