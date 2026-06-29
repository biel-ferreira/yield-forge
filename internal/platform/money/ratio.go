package money

import "math/big"

// ShareBps returns part/whole expressed in basis points (×10000), rounded half-away-from-zero
// — the project's documented rule, consistent with DecimalToMinor (SPEC-103 BR-1032). A whole
// of 0 (or negative) yields 0, so a divide-by-zero is impossible. part may be negative (e.g. a
// loss); whole is expected non-negative (a total). For realistic centavos magnitudes the
// intermediate product stays well within int64.
//
//	ShareBps(70, 100) -> 7000   // 70.00%
//	ShareBps(1, 3)    -> 3333    // 33.33%, half-up
func ShareBps(part, whole int64) int {
	if whole <= 0 {
		return 0
	}
	if part < 0 {
		return -ShareBps(-part, whole)
	}
	// floor(part*10000/whole + 1/2) = (2*part*10000 + whole) / (2*whole)
	return int((2*part*10000 + whole) / (2 * whole))
}

// AccrueSimpleInterest returns the simple interest accrued on principalCentavos at rateBps per
// year over elapsedDays, in centavos, rounded half-up: principal × rateBps × days /
// (10000 × 365). It computes the product with big.Int so the intermediate never overflows
// int64, then rounds (SPEC-103 D2). Non-positive inputs yield 0. The 365-day year is a
// deliberate MVP approximation (PRD A4).
func AccrueSimpleInterest(principalCentavos int64, rateBps, elapsedDays int) int64 {
	if principalCentavos <= 0 || rateBps <= 0 || elapsedDays <= 0 {
		return 0
	}
	num := new(big.Int).Mul(big.NewInt(principalCentavos), big.NewInt(int64(rateBps)))
	num.Mul(num, big.NewInt(int64(elapsedDays)))
	den := big.NewInt(10000 * 365)
	// half-up for non-negative values: (2*num + den) / (2*den)
	twoNum := new(big.Int).Lsh(num, 1)
	twoNum.Add(twoNum, den)
	q := new(big.Int).Quo(twoNum, new(big.Int).Lsh(den, 1))
	return q.Int64()
}
