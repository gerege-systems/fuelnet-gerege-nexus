/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * What the platform is being used for, as opposed to how it is holding up.
 */

package observability

import "github.com/prometheus/client_golang/prometheus"

// No tenant on any of these.
//
// A tenant id or slug in a label multiplies every series by the number of
// organisations on the deployment, and it never shrinks — an organisation that
// leaves keeps its series until the retention window expires. The per-tenant
// breakdown is a reporting question, and it is answered by the reports module
// against the database, where a row can be deleted.
var (
	// LoginsTotal counts sign-in attempts by how they were made and how they
	// ended. method: password|eid|dan|google|sso. result: success|failure.
	LoginsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logins_total",
			Help: "Sign-in attempts by method and outcome",
		},
		[]string{"method", "result"},
	)

	// InvoicesCreatedTotal counts invoices issued.
	InvoicesCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "invoices_created_total",
			Help: "Invoices issued across all tenants",
		},
	)

	// DocumentsSignedTotal counts completed signature ceremonies.
	// rail: EID|DAN|HSM. result: success|failure.
	DocumentsSignedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "documents_signed_total",
			Help: "Signature ceremonies by rail and outcome",
		},
		[]string{"rail", "result"},
	)

	// AIRequestsTotal counts calls into the copilot, by what was asked of it.
	// kind: copilot|chat|stt|tts|translate|forecast.
	AIRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_requests_total",
			Help: "Requests handled by the AI endpoints, by kind",
		},
		[]string{"kind"},
	)
)

// Login methods and outcomes, as label values rather than strings typed at each
// call site.
const (
	LoginPassword = "password"
	LoginEID      = "eid"
	LoginDAN      = "dan"
	LoginGoogle   = "google"
	LoginSSO      = "sso"

	ResultSuccess = "success"
	ResultFailure = "failure"
)

func init() {
	prometheus.MustRegister(LoginsTotal, InvoicesCreatedTotal, DocumentsSignedTotal, AIRequestsTotal)
}

// RecordLogin counts one sign-in attempt.
func RecordLogin(method string, ok bool) {
	LoginsTotal.WithLabelValues(method, resultLabel(ok)).Inc()
}

// RecordInvoiceCreated counts one issued invoice.
func RecordInvoiceCreated() { InvoicesCreatedTotal.Inc() }

// RecordDocumentSigned counts one signature ceremony reaching its end.
func RecordDocumentSigned(rail string, ok bool) {
	DocumentsSignedTotal.WithLabelValues(rail, resultLabel(ok)).Inc()
}

// RecordAIRequest counts one call into the AI endpoints.
func RecordAIRequest(kind string) { AIRequestsTotal.WithLabelValues(kind).Inc() }

func resultLabel(ok bool) string {
	if ok {
		return ResultSuccess
	}
	return ResultFailure
}
