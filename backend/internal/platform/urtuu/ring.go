/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * ring.dgov.mn — where the request codes actually come from.
 *
 * The register of the state's service processes is not this platform's to
 * author (§2.5, §9). Every process already has a definition there: what the
 * service is, what it needs, and how long it is allowed to take. Өртөө imports
 * that and turns it into codes, schemas and SLAs.
 *
 * The one thing this file deliberately does NOT do is guess the register's wire
 * format. What Ring answers with — the shape of a process, how a step becomes a
 * schema field, how a norm is expressed — is an agreement to be made with its
 * operators, and a parser written from imagination would look finished, pass
 * its own tests, and be wrong in the field. So the boundary is drawn here, the
 * mock behind it is honest about being a mock, and the real client is a skeleton
 * with the one unknown marked.
 *
 * Nothing here is on any request path. Codes live in the database once they are
 * imported, so an unreachable register costs an out-of-date vocabulary and
 * never an outage — the same fallback rule the catalogue follows.
 */

package urtuu

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
	contract "github.com/gerege-systems/open-gerege-nexus/backend/pkg/urtuu"
)

const (
	ringBaseURLEnv = "RING_BASE_URL"
	// #nosec G101 -- the name of an environment variable, not a credential.
	ringAPIKeyEnv = "RING_API_KEY"

	// ringMockURL is what an operator sets RING_BASE_URL to in order to develop
	// against the shape of the feature without credentials. Spelled out rather
	// than inferred from an empty value, because "unconfigured" and "pretend"
	// must not be the same state: the first is off, the second serves invented
	// codes and has to be a choice somebody made.
	ringMockURL = "mock"

	// ringTimeout bounds one import.
	ringTimeout = 30 * time.Second

	// maxRingBytes bounds the register's answer.
	maxRingBytes = 8 << 20
)

// RingImporter is the register, as this package needs it.
//
// One method, because one question is asked: what processes are there. It is an
// interface so the unknown — the wire format — sits behind a boundary that can
// be filled in without touching the import, the storage or the screen.
type RingImporter interface {
	Processes(ctx context.Context) ([]contract.RequestCode, error)
}

// newRingImporter builds whichever importer the environment names, or nil.
func newRingImporter(client *http.Client) RingImporter {
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv(ringBaseURLEnv)), "/")
	switch base {
	case "":
		return nil
	case ringMockURL:
		slog.Warn("urtuu: " + ringBaseURLEnv + " is set to \"" + ringMockURL +
			"\", so request codes are invented for development and are not the state register's")
		return ringMock{}
	default:
		key := strings.TrimSpace(os.Getenv(ringAPIKeyEnv))
		if key == "" {
			slog.Error("urtuu: " + ringBaseURLEnv + " is set but " + ringAPIKeyEnv +
				" is not, so the request-code import is off")
			return nil
		}
		slog.Info("urtuu: request codes will be imported from the state register", "base_url", base)
		return &ringHTTP{base: base, key: key, client: client}
	}
}

// ringMock is a handful of plausible codes for developing against.
//
// They are labelled as invented in their own names, in every language, because
// the one failure this must not have is a mock code reaching a real deployment
// and being read as a national one.
type ringMock struct{}

func (ringMock) Processes(_ context.Context) ([]contract.RequestCode, error) {
	return []contract.RequestCode{
		{
			Code: "D-101",
			Names: map[string]string{
				"mn": "Тооллого явуулах (жишиг)", "en": "Carry out a count (sample)",
				"ar": "إجراء إحصاء (عينة)", "zh": "开展清点（示例）",
				"fr": "Réaliser un recensement (exemple)", "ru": "Провести перепись (образец)",
				"es": "Realizar un recuento (muestra)",
			},
			Schema: []byte(`{"type":"object","required":["period"],"properties":` +
				`{"period":{"type":"string","title":"Хамрах хугацаа"},` +
				`"scope":{"type":"string","title":"Хамрах хүрээ"}}}`),
			DefaultSLA: 14 * 24 * time.Hour,
			// The register is the state's list of services, so what comes from
			// it is service-line work by definition.
			Line:           contract.LineService,
			Source:         contract.SourceRing,
			RingProcessRef: "mock/D-101",
			Version:        1,
			Active:         true,
		},
		{
			Code: "D-204",
			Names: map[string]string{
				"mn": "Мэдээлэл гаргуулах (жишиг)", "en": "Provide information (sample)",
				"ar": "تقديم معلومات (عينة)", "zh": "提供信息（示例）",
				"fr": "Fournir des informations (exemple)", "ru": "Предоставить сведения (образец)",
				"es": "Proporcionar información (muestra)",
			},
			Schema: []byte(`{"type":"object","required":["subject"],"properties":` +
				`{"subject":{"type":"string","title":"Асуулгын сэдэв"}}}`),
			DefaultSLA:     3 * 24 * time.Hour,
			Line:           contract.LineService,
			Source:         contract.SourceRing,
			RingProcessRef: "mock/D-204",
			Version:        1,
			Active:         true,
		},
	}, nil
}

// ringHTTP is the real client.
//
// The request half is settled — a bearer key against a base URL, bounded and
// with a timeout, like every other outbound call on this platform. The answer
// half is not, and is marked rather than invented.
type ringHTTP struct {
	base   string
	key    string
	client *http.Client
}

func (r *ringHTTP) Processes(ctx context.Context) ([]contract.RequestCode, error) {
	ctx, cancel := context.WithTimeout(ctx, ringTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.base+"/processes", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.key)

	res, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ring.dgov.mn answered %s", res.Status)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxRingBytes))
	if err != nil {
		return nil, err
	}

	// TODO(urtuu/ring): decode the register's answer into []contract.RequestCode.
	//
	// Blocked on an agreement with the operators of ring.dgov.mn, and left
	// blocked on purpose (§9): the mapping from a process definition to a code,
	// a JSON Schema and an SLA is theirs to describe, and a parser guessed from
	// the field names one would expect is a parser that passes its own tests
	// and is wrong against the real endpoint. What is needed before this line
	// can be written:
	//
	//	* the shape of a process record, and which field is the service code;
	//	* how the seven names are carried, if they are;
	//	* how a process's steps and required documents map to schema fields;
	//	* how a time norm is expressed — working days, calendar days, hours;
	//	* how a change is signalled, so an import can be conditional rather
	//	  than a full re-read (an ETag would settle it, as with the catalogue).
	//
	// Until then this refuses loudly. A deployment that has configured a real
	// base URL gets an error naming the gap; one that has not is simply off,
	// and one developing against the shape sets RING_BASE_URL=mock.
	return nil, fmt.Errorf("ring.dgov.mn answered %d bytes, and this build cannot yet read them: "+
		"the register's format has not been agreed (see the TODO in internal/platform/urtuu/ring.go)", len(body))
}

// importRing reads the register and writes what it says into one organisation's
// vocabulary.
//
// Imported codes are not switched on for anybody by themselves: `active` is
// left alone on an update, and a newly imported code still has to be opened on
// a link before a child ever hears of it.
func (s *Service) importRing(ctx context.Context, tenantID string) (int, error) {
	if s.ring == nil {
		return 0, ErrRingUnconfigured
	}
	codes, err := s.ring.Processes(ctx)
	if err != nil {
		return 0, err
	}

	ctx = nexus.WithTenantID(ctx, tenantID)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, code := range codes {
		if strings.HasPrefix(code.Code, contract.LocalPrefix) {
			// The register owns the unprefixed namespace and nothing else. A
			// record that claimed a local code would overwrite something this
			// organisation authored.
			continue
		}
		if err := upsertCode(ctx, tx, tenantID, contract.SourceRing, "", code); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(codes), nil
}

func (s *Service) handleRingSync(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := nexus.RequireTenant(w, r)
	if !ok {
		return
	}

	imported, err := s.importRing(r.Context(), tenantID)
	if err != nil {
		// 503 rather than 500: the register is somebody else's service, and the
		// answer to "it is not configured" or "it did not answer" is to try
		// again or to configure it, not to report a fault in this platform.
		nexus.Error(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	audit.Record(r.Context(), tenantID, actorOf(r), "urtuu.ring_imported", "urtuu_request_code",
		map[string]any{"count": imported})
	nexus.JSON(w, http.StatusOK, map[string]any{"imported": imported})
}
