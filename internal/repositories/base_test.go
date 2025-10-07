package repositories

import (
	"os"
	"testing"

	xlog "bitbucket.org/Amartha/go-x/log"
	"github.com/google/go-cmp/cmp"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}

func balanceComparer() cmp.Option {
	return cmp.Comparer(func(x, y models.Balance) bool {
		return x.Actual().Equal(y.Actual()) &&
			x.Pending().Equal(y.Pending()) &&
			x.Available().Equal(y.Available())
	})
}
