package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

type BalancePrometheusMetrics struct {
	balanceOperations *prometheus.CounterVec
	balanceMovements  *prometheus.CounterVec
}

func newBalancePrometheusMetrics(reg prometheus.Registerer) *BalancePrometheusMetrics {
	mtc := &BalancePrometheusMetrics{
		balanceOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "acuan_balance_operations_total",
				Help: "Number of balance transactions by type",
			},
			[]string{"transaction_type"},
		),
		balanceMovements: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "acuan_balance_movements_total",
				Help: "Number of balance movements by type",
			},
			[]string{"transaction_type"},
		),
	}

	reg.MustRegister(mtc.balanceOperations)
	reg.MustRegister(mtc.balanceMovements)

	return mtc
}

func (m *BalancePrometheusMetrics) Record(transactions []models.Transaction) {
	if m == nil {
		return
	}

	for _, transaction := range transactions {
		amount, _ := transaction.Amount.Decimal.Float64()

		m.balanceOperations.WithLabelValues(transaction.TypeTransaction).Inc()
		m.balanceMovements.WithLabelValues(transaction.TypeTransaction).Add(amount)
	}
}
