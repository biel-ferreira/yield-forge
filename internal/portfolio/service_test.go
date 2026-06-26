package portfolio

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeRepo captures created holdings and lets a test force the scoped-mutation not-found path.
type fakeRepo struct {
	createdFII []FIIHolding
	createdFI  []FixedIncomeHolding
	fiiList    []FIIHolding
	fiList     []FixedIncomeHolding
	mutateErr  error // returned by Update*/Delete* (e.g. ErrHoldingNotFound)
}

func (f *fakeRepo) CreateFIIHolding(_ context.Context, h FIIHolding) (FIIHolding, error) {
	h.ID = "fii-id"
	f.createdFII = append(f.createdFII, h)
	return h, nil
}
func (f *fakeRepo) ListFIIHoldingsByUserID(context.Context, string) ([]FIIHolding, error) {
	return f.fiiList, nil
}
func (f *fakeRepo) UpdateFIIHolding(_ context.Context, h FIIHolding) (FIIHolding, error) {
	if f.mutateErr != nil {
		return FIIHolding{}, f.mutateErr
	}
	return h, nil
}
func (f *fakeRepo) DeleteFIIHolding(context.Context, string, string) error { return f.mutateErr }

func (f *fakeRepo) CreateFixedIncomeHolding(_ context.Context, h FixedIncomeHolding) (FixedIncomeHolding, error) {
	h.ID = "fi-id"
	f.createdFI = append(f.createdFI, h)
	return h, nil
}
func (f *fakeRepo) ListFixedIncomeHoldingsByUserID(context.Context, string) ([]FixedIncomeHolding, error) {
	return f.fiList, nil
}
func (f *fakeRepo) UpdateFixedIncomeHolding(_ context.Context, h FixedIncomeHolding) (FixedIncomeHolding, error) {
	if f.mutateErr != nil {
		return FixedIncomeHolding{}, f.mutateErr
	}
	return h, nil
}
func (f *fakeRepo) DeleteFixedIncomeHolding(context.Context, string, string) error {
	return f.mutateErr
}

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

func newService(repo Repository) *Service {
	return NewService(repo, fakeClock{t: time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)})
}

func TestService_CreateFIIHolding(t *testing.T) {
	repo := &fakeRepo{}
	svc := newService(repo)

	got, err := svc.CreateFIIHolding(context.Background(), "u1", FIIInput{Ticker: "hglg11", Quantity: 100, AveragePriceCentavos: 15_750})
	require.NoError(t, err)
	require.Equal(t, "u1", got.UserID, "identity from the userID argument")
	require.Equal(t, "HGLG11", got.Ticker.String())
	require.Equal(t, 100, got.Quantity.Value())
	require.Equal(t, int64(15_750), got.AveragePriceCentavos)
}

func TestService_CreateFIIHolding_InvalidWritesNothing(t *testing.T) {
	cases := map[string]FIIInput{
		"bad ticker":     {Ticker: "BAD", Quantity: 1, AveragePriceCentavos: 100},
		"zero quantity":  {Ticker: "HGLG11", Quantity: 0, AveragePriceCentavos: 100},
		"negative price": {Ticker: "HGLG11", Quantity: 1, AveragePriceCentavos: -1},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepo{}
			_, err := newService(repo).CreateFIIHolding(context.Background(), "u1", in)
			require.Error(t, err)
			require.Empty(t, repo.createdFII, "nothing written on a validation failure")
		})
	}
}

func TestService_CreateFixedIncome_MaturityRules(t *testing.T) {
	future := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("at_maturity with future date is stored", func(t *testing.T) {
		repo := &fakeRepo{}
		got, err := newService(repo).CreateFixedIncomeHolding(context.Background(), "u1", FixedIncomeInput{
			Name: "CDB", Institution: "Banco", InvestedAmountCentavos: 1000, AnnualRateBps: 1200,
			MaturityDate: &future, LiquidityType: "at_maturity",
		})
		require.NoError(t, err)
		require.NotNil(t, got.MaturityDate)
	})

	t.Run("at_maturity in the past is rejected", func(t *testing.T) {
		repo := &fakeRepo{}
		_, err := newService(repo).CreateFixedIncomeHolding(context.Background(), "u1", FixedIncomeInput{
			Name: "CDB", Institution: "Banco", InvestedAmountCentavos: 1000, AnnualRateBps: 1200,
			MaturityDate: &past, LiquidityType: "at_maturity",
		})
		require.ErrorIs(t, err, ErrPastMaturity)
		require.Empty(t, repo.createdFI)
	})

	t.Run("at_maturity without a date is rejected", func(t *testing.T) {
		_, err := newService(&fakeRepo{}).CreateFixedIncomeHolding(context.Background(), "u1", FixedIncomeInput{
			Name: "CDB", Institution: "Banco", InvestedAmountCentavos: 1000, AnnualRateBps: 1200,
			LiquidityType: "at_maturity",
		})
		require.ErrorIs(t, err, ErrMaturityRequired)
	})

	t.Run("daily normalizes the maturity to nil", func(t *testing.T) {
		repo := &fakeRepo{}
		got, err := newService(repo).CreateFixedIncomeHolding(context.Background(), "u1", FixedIncomeInput{
			Name: "Caixinha", Institution: "Banco", InvestedAmountCentavos: 1000, AnnualRateBps: 1100,
			MaturityDate: &future, LiquidityType: "daily", // date provided but irrelevant
		})
		require.NoError(t, err)
		require.Nil(t, got.MaturityDate, "a daily-liquidity holding never carries a maturity date")
	})
}

func TestService_CreateFixedIncome_FieldValidation(t *testing.T) {
	base := FixedIncomeInput{Name: "CDB", Institution: "Banco", InvestedAmountCentavos: 1000, AnnualRateBps: 1200, LiquidityType: "daily"}
	cases := map[string]func(FixedIncomeInput) FixedIncomeInput{
		"empty name":        func(in FixedIncomeInput) FixedIncomeInput { in.Name = "  "; return in },
		"empty institution": func(in FixedIncomeInput) FixedIncomeInput { in.Institution = ""; return in },
		"zero amount":       func(in FixedIncomeInput) FixedIncomeInput { in.InvestedAmountCentavos = 0; return in },
		"negative rate":     func(in FixedIncomeInput) FixedIncomeInput { in.AnnualRateBps = -1; return in },
		"bad liquidity":     func(in FixedIncomeInput) FixedIncomeInput { in.LiquidityType = "weekly"; return in },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newService(&fakeRepo{}).CreateFixedIncomeHolding(context.Background(), "u1", mutate(base))
			require.Error(t, err)
		})
	}
}

func TestService_UpdatePropagatesNotFound(t *testing.T) {
	repo := &fakeRepo{mutateErr: ErrHoldingNotFound}
	_, err := newService(repo).UpdateFIIHolding(context.Background(), "u1", "some-id", FIIInput{Ticker: "HGLG11", Quantity: 1, AveragePriceCentavos: 100})
	require.ErrorIs(t, err, ErrHoldingNotFound)
}

func TestService_ListHoldings_Aggregates(t *testing.T) {
	repo := &fakeRepo{
		fiiList: []FIIHolding{{ID: "a"}},
		fiList:  []FixedIncomeHolding{{ID: "b"}, {ID: "c"}},
	}
	got, err := newService(repo).ListHoldings(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, got.FII, 1)
	require.Len(t, got.FixedIncome, 2)
}
