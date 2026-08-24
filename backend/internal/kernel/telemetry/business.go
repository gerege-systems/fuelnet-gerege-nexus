/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * What the platform is being used for, as opposed to how it is holding up.
 */

package telemetry

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

	// invoices_created_total is not here. It counts a billing event, and
	// billing is a distribution module — a platform that ships a counter named
	// after somebody else's domain has to be edited every time that domain
	// moves. The metric keeps its name and is registered by the module that
	// knows when an invoice happens; a deployment without billing simply does
	// not export it, which is the truth rather than a permanent zero.

	// DocumentsSignedTotal counts completed signature ceremonies.
	// rail: EID|DAN|HSM. result: success|failure.
	DocumentsSignedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "documents_signed_total",
			Help: "Signature ceremonies by rail and outcome",
		},
		[]string{"rail", "result"},
	)

	// CPLoginAttemptsTotal counts attempts to sign in to the operator console,
	// by how they ended: success, unknown, bad_password, bad_code, locked,
	// disabled, no_second_factor, step_up, bad_step_up.
	//
	// Kept apart from LoginsTotal rather than given a method label, because the
	// two answer different questions and one of them needs an alert. Sign-ins
	// to the platform fail all day — people mistype passwords. Sign-ins to the
	// control plane are a handful of people a week, so a dozen failures in an
	// hour is not noise, it is somebody trying.
	//
	// The result is a closed set and there is no operator identity in the
	// label. Both matter: an address as a label value would be an unbounded
	// series driven by whoever is guessing.
	CPLoginAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cp_login_attempts_total",
			Help: "Operator console sign-in attempts by outcome",
		},
		[]string{"result"},
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
	prometheus.MustRegister(LoginsTotal, CPLoginAttemptsTotal,
		DocumentsSignedTotal, AIRequestsTotal)
}

// RecordLogin counts one sign-in attempt.
func RecordLogin(method string, ok bool) {
	LoginsTotal.WithLabelValues(method, resultLabel(ok)).Inc()
}

// RecordControlPlaneLogin counts one attempt to reach the operator console.
func RecordControlPlaneLogin(result string) {
	CPLoginAttemptsTotal.WithLabelValues(result).Inc()
}

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
