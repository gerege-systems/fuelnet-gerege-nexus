package billing

import "github.com/prometheus/client_golang/prometheus"

// invoicesCreated counts invoices issued, under the name the dashboards already
// use.
//
// It used to live in the platform's observability package, which meant the
// platform carried a counter named after this module's domain — and would have
// had to be edited the day this module moved to its own repository. The metric
// belongs where the event is known.
//
// Registered on the default registerer, which is what `/metrics` serves. A
// deployment that does not install billing does not export this series at all;
// an absent metric is a truer answer than a zero that never moves.
var invoicesCreated = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "invoices_created_total",
	Help: "Invoices issued across all tenants",
})

func init() { prometheus.MustRegister(invoicesCreated) }
