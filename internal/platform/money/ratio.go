package money

import (
	"math/big"
	"sort"
)

// ShareBps returns part/whole expressed in basis points (×10000), rounded half-away-from-zero
// — the project's documented rule, consistent with DecimalToMinor (SPEC-103 BR-1032). A whole
// of 0 (or negative) yields 0, so a divide-by-zero is impossible. part may be negative (e.g. a
// loss); whole is expected non-negative (a total). The intermediate `2*part*10000` overflows
// int64 only for |part| > ~4.6e14 centavos (≈ R$4.6 trillion) — far beyond any retail
// portfolio, so int64 arithmetic is safe here (unlike the accrual, which needs big.Int).
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

// AllocateBps distributes 10000 basis points across the given weights in proportion to each
// weight, using the largest-remainder method so the result sums to EXACTLY 10000 — no rounding
// drift (SPEC-105 BR-1056: the rebalancing split reconciles to 100%). Negative weights are
// treated as 0; a zero or empty total yields all-zero shares. Ties in the leftover remainder go
// to the lower index (deterministic). Like ShareBps, weight*10000 is int64-safe at retail scale.
func AllocateBps(weights []int64) []int {
	shares := make([]int, len(weights))
	var total int64
	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return shares // all zero
	}

	// Floor each share; remember the remainders so the leftover bps go to the largest of them.
	type rem struct {
		idx int
		r   int64
	}
	rems := make([]rem, len(weights))
	distributed := 0
	for i, w := range weights {
		rems[i].idx = i
		if w <= 0 {
			continue
		}
		num := w * 10000
		base := int(num / total)
		shares[i] = base
		distributed += base
		rems[i].r = num % total
	}

	leftover := 10000 - distributed
	// Stable sort by descending remainder keeps the original (ascending-index) order on ties.
	sort.SliceStable(rems, func(a, b int) bool { return rems[a].r > rems[b].r })
	for k := 0; k < leftover && k < len(rems); k++ {
		shares[rems[k].idx]++
	}
	return shares
}

// ApplyBps returns amount × bps / 10000, rounded half-up — the inverse of ShareBps, used to turn
// a basis-point share into a centavos amount. Non-positive inputs yield 0; big.Int guards the
// intermediate product against int64 overflow (SPEC-105).
func ApplyBps(amount int64, bps int) int64 {
	if amount <= 0 || bps <= 0 {
		return 0
	}
	num := new(big.Int).Mul(big.NewInt(amount), big.NewInt(int64(bps)))
	den := big.NewInt(10000)
	// half-up for non-negative values: (2*num + den) / (2*den)
	twoNum := new(big.Int).Lsh(num, 1)
	twoNum.Add(twoNum, den)
	q := new(big.Int).Quo(twoNum, new(big.Int).Lsh(den, 1))
	return q.Int64()
}
