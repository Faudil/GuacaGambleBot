package loan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testStore(t *testing.T) *store.Store {
	d := testutil.NewDB(t)
	return store.New(d, &config.Config{StartingBalance: 100, DailyAmount: 50})
}

func TestBorrow(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	require.NoError(t, svc.Borrow(1, 500))

	var loans []model.Loan
	require.NoError(t, s.DB.Where("borrower_id = ?", 1).Find(&loans).Error)
	require.Len(t, loans, 1)
	assert.Equal(t, 500, loans[0].AmountDue)
	assert.Equal(t, int64(0), loans[0].LenderID)

	bal, err := s.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 600, bal)
}

func TestBorrowErrors(t *testing.T) {
	s := testStore(t)
	svc := New(s)
	assert.ErrorIs(t, svc.Borrow(1, 0), ErrAmount)
	assert.ErrorIs(t, svc.Borrow(1, -5), ErrAmount)
	assert.ErrorIs(t, svc.Borrow(1, 1001), ErrMaxLoan)
}

func TestRepay(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	require.NoError(t, svc.Borrow(1, 500))

	debt, err := s.GetTotalDebt(1)
	require.NoError(t, err)
	assert.Equal(t, 500, debt)

	paid, err := svc.Repay(1, 200)
	require.NoError(t, err)
	assert.Equal(t, 200, paid)

	bal, err := s.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 400, bal)

	debt, err = s.GetTotalDebt(1)
	require.NoError(t, err)
	assert.Equal(t, 300, debt)
}

func TestRepayErrors(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	_, err := svc.Repay(1, 100)
	assert.ErrorIs(t, err, ErrNoDebt)

	require.NoError(t, svc.Borrow(1, 300))

	_, err = svc.Repay(1, 0)
	assert.ErrorIs(t, err, ErrAmount)

	_, err = svc.Repay(1, 500)
	assert.ErrorIs(t, err, ErrExceedsDebt)
}
