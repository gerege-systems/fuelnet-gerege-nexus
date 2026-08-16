/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * The register number — what a person quotes down a telephone.
 *
 * A UUID is perfect for a machine and useless to anybody else. What an officer
 * at a sum actually says is "our fourteenth assignment, the one from the aimag's
 * eighty-seventh", and until now there was nothing on the row that could be
 * said out loud.
 *
 * See migration 00066 for the format and for why there is no platform prefix on
 * it: the link already carries the other installation's name, given by the
 * administrator who established it, and a second human name for the same thing
 * is one that can disagree with the first.
 */

package urtuu

import (
	"context"
	"fmt"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
	contract "github.com/gerege-systems/open-gerege-nexus/backend/pkg/urtuu"
	"github.com/jackc/pgx/v5"
)

// lineMark is the first character of a number: the promise, in one letter.
//
// Cyrillic, because the number is printed on Mongolian paperwork and read by
// people who call the two lines Үйлчилгээ and Даалгавар. It is never an API
// identifier — routes take a UUID — so it does not have to survive a URL.
func lineMark(line string) string {
	if line == contract.LineService {
		return "Ү"
	}
	return "Д"
}

// nextNumber allocates one register number inside the caller's transaction.
//
// In the caller's transaction on purpose. The counter is a single row and the
// UPDATE takes a row lock, so two people raising work at the same moment
// serialise on it rather than racing — and a transaction that rolls back gives
// its number back rather than leaving a gap in a register somebody audits.
//
// The year comes from the moment of registration rather than from an envelope's
// stamp: a request that arrives on the 31st of December and is registered on
// the 1st of January belongs to the new year's register, which is the year its
// number has to name.
func nextNumber(ctx context.Context, tx pgx.Tx, tenantID, line string, when time.Time) (string, error) {
	if !contract.KnownLine(line) {
		line = contract.LineAssignment
	}
	// The platform's clock. A request registered at half past midnight in
	// Ulaanbaatar on the first of January belongs to the new year's register,
	// and on a UTC container `when.Year()` would have filed it under the old
	// one — for eight hours every New Year, on the numbers people audit.
	year := when.In(config.Location()).Year()

	var sequence int
	if err := tx.QueryRow(ctx, `
		INSERT INTO urtuu_numbers (tenant_id, line, year, next)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (tenant_id, line, year)
		DO UPDATE SET next = urtuu_numbers.next + 1
		RETURNING next`, tenantID, line, year).Scan(&sequence); err != nil {
		return "", err
	}

	// Five digits, and it grows rather than wrapping: an organisation that
	// passes a hundred thousand requests in a year gets a longer number, not a
	// number somebody else already has.
	return fmt.Sprintf("%s%d-%05d", lineMark(line), year, sequence), nil
}
