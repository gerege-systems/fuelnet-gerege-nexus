/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 *
 * The organisation in PostgreSQL. Every query here was lifted out of an HTTP
 * handler unchanged — same columns, same joins, same ordering — because a
 * refactor that also rewrites the SQL cannot say which of the two broke
 * anything.
 *
 * What this layer decides is one thing: what a database error means. A unique
 * violation is a code somebody has already used, a foreign key violation is a
 * unit from another organisation, and no rows is a membership that is not this
 * tenant's. Above here those are domain refusals, and nothing has to know that
 * PostgreSQL numbers them 23505 and 23503.
 */
package postgres

import (
	"context"
	"errors"

	"github.com/gerege-systems/open-gerege-nexus/backend/domain/organisation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB is the handle this store needs. It is pgx's own shape and deliberately so:
// the platform hands modules a *pgxpool.Pool behind nexus.DB, and that value
// satisfies this as it is. Naming nexus.DB here would be the one import that
// puts the SDK back into the domain.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Store struct{ db DB }

func New(db DB) *Store { return &Store{db: db} }

func (s *Store) ListPeople(ctx context.Context, tenantIDs []string) ([]organisation.Person, error) {
	rows, err := s.db.Query(ctx,
		`SELECT ms.id::text, u.id::text, u.name, u.email, u.phone, ms.job_title,
		        COALESCE(ms.department_id::text, ''), COALESCE(d.name, ''),
		        ms.active, u.is_admin, ms.created_at::text,
		        COALESCE(ARRAY_AGG(r.code) FILTER (WHERE r.code IS NOT NULL), '{}'),
		        ms.tenant_id::text, tn.name
		   FROM memberships ms
		   JOIN users u ON u.id = ms.user_id
		   JOIN tenants tn ON tn.id = ms.tenant_id
		   LEFT JOIN departments d ON d.id = ms.department_id
		   LEFT JOIN membership_roles mr ON mr.membership_id = ms.id
		   LEFT JOIN roles r ON r.id = mr.role_id
		  WHERE ms.tenant_id = ANY($1::uuid[])
		  GROUP BY ms.id, u.id, d.name, tn.name
		  ORDER BY tn.name, ms.active DESC, u.name`, tenantIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	people := make([]organisation.Person, 0)
	for rows.Next() {
		var p organisation.Person
		if err := rows.Scan(&p.MembershipID, &p.UserID, &p.Name, &p.Email, &p.Phone,
			&p.JobTitle, &p.DepartmentID, &p.DepartmentName, &p.Active, &p.IsAdmin,
			&p.JoinedAt, &p.Roles, &p.TenantID, &p.TenantName); err != nil {
			return nil, organisation.Failed("could not read the people", err)
		}
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		return nil, organisation.Failed("could not read the people", err)
	}
	return people, nil
}

func (s *Store) Membership(ctx context.Context, tenantID, membershipID string) (organisation.Membership, error) {
	var m organisation.Membership
	err := s.db.QueryRow(ctx,
		`SELECT ms.user_id::text, u.is_admin
		   FROM memberships ms JOIN users u ON u.id = ms.user_id
		  WHERE ms.id = $1 AND ms.tenant_id = $2`,
		membershipID, tenantID).Scan(&m.UserID, &m.IsAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return organisation.Membership{}, organisation.ErrCrossTenant
	}
	return m, err
}

func (s *Store) UpdatePerson(ctx context.Context, tenantID, membershipID string, edit organisation.PersonEdit) (bool, error) {
	department, set := edit.Department()
	tag, err := s.db.Exec(ctx,
		`UPDATE memberships SET
		     job_title     = COALESCE($3, job_title),
		     department_id = CASE WHEN $4 THEN $5::uuid ELSE department_id END
		 WHERE id = $1 AND tenant_id = $2`,
		membershipID, tenantID, edit.JobTitle, set, department)
	if isForeignKeyViolation(err) {
		// Same composite key as everywhere else: a department belonging to
		// another organisation is refused by the schema, not by a check here.
		return false, organisation.ErrForeignDepartment
	}
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) CountAdmins(ctx context.Context, tenantID, exceptMembershipID string) (int, error) {
	var remaining int
	err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM memberships ms
		   JOIN users u ON u.id = ms.user_id
		  WHERE ms.tenant_id = $1 AND ms.active AND u.is_admin AND ms.id <> $2`,
		tenantID, exceptMembershipID).Scan(&remaining)
	return remaining, err
}

func (s *Store) SetPersonActive(ctx context.Context, tenantID, membershipID string, active bool) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE memberships SET active = $3,
		        deactivated_at = CASE WHEN $3 THEN NULL ELSE NOW() END
		 WHERE id = $1 AND tenant_id = $2`, membershipID, tenantID, active)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) ListDepartments(ctx context.Context, tenantIDs []string) ([]organisation.Department, error) {
	rows, err := s.db.Query(ctx,
		`SELECT d.id::text, d.code, d.name, COALESCE(d.parent_id::text, ''),
		        COALESCE(d.manager_membership_id::text, ''), COALESCE(u.name, ''), d.active,
		        (SELECT count(*) FROM memberships ms WHERE ms.department_id = d.id AND ms.active),
		        d.tenant_id::text, tn.name
		   FROM departments d
		   JOIN tenants tn ON tn.id = d.tenant_id
		   LEFT JOIN memberships mgr ON mgr.id = d.manager_membership_id
		   LEFT JOIN users u ON u.id = mgr.user_id
		  WHERE d.tenant_id = ANY($1::uuid[])
		  ORDER BY tn.name, d.active DESC, d.name`, tenantIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]organisation.Department, 0)
	for rows.Next() {
		var d organisation.Department
		if err := rows.Scan(&d.ID, &d.Code, &d.Name, &d.ParentID, &d.ManagerID,
			&d.ManagerName, &d.Active, &d.PeopleCount, &d.TenantID, &d.TenantName); err != nil {
			return nil, organisation.Failed("could not read the departments", err)
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		return nil, organisation.Failed("could not read the departments", err)
	}
	return list, nil
}

func (s *Store) CreateDepartment(ctx context.Context, tenantID string, edit organisation.DepartmentEdit) (string, error) {
	var id string
	err := s.db.QueryRow(ctx,
		`INSERT INTO departments (tenant_id, code, name, parent_id, manager_membership_id)
		 VALUES ($1, $2, $3, $4::uuid, $5::uuid) RETURNING id::text`,
		tenantID, edit.Code, edit.Name, edit.Parent(), edit.Manager()).Scan(&id)
	if isUniqueViolation(err) {
		return "", organisation.ErrDuplicateCode
	}
	if isForeignKeyViolation(err) {
		// The composite foreign keys make this the schema's answer to a parent
		// or a manager from another tenant, rather than a check somebody has to
		// remember to write.
		return "", organisation.ErrForeignUnit
	}
	return id, err
}

func (s *Store) UpdateDepartment(ctx context.Context, tenantID, id string, edit organisation.DepartmentEdit) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE departments SET name = $3, parent_id = $4::uuid,
		        manager_membership_id = $5::uuid, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID, edit.Name, edit.Parent(), edit.Manager())
	if isForeignKeyViolation(err) {
		return false, organisation.ErrForeignUnit
	}
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) IsDescendant(ctx context.Context, tenantID, ancestorID, candidateID string) (bool, error) {
	var descendant bool
	err := s.db.QueryRow(ctx,
		`WITH RECURSIVE below AS (
		    SELECT id FROM departments WHERE parent_id = $1 AND tenant_id = $3
		    UNION ALL
		    SELECT d.id FROM departments d JOIN below b ON d.parent_id = b.id
		     WHERE d.tenant_id = $3
		 )
		 SELECT EXISTS (SELECT 1 FROM below WHERE id = $2::uuid)`,
		ancestorID, candidateID, tenantID).Scan(&descendant)
	return descendant, err
}

func (s *Store) Parent(ctx context.Context, tenantID, id string) (string, bool, error) {
	var name string
	var archived bool
	err := s.db.QueryRow(ctx,
		`SELECT p.name, NOT p.active
		   FROM departments d JOIN departments p ON p.id = d.parent_id
		  WHERE d.id = $1 AND d.tenant_id = $2`, id, tenantID).Scan(&name, &archived)
	// No row is a root — or a unit that is not there at all, which the update
	// after this will answer for.
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return name, archived, err
}

func (s *Store) SetDepartmentArchived(ctx context.Context, tenantID, id string, archived bool) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE departments SET active = $3, updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`, id, tenantID, !archived)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) CountChildren(ctx context.Context, tenantID, id string) (int, int, error) {
	var people, units int
	err := s.db.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM memberships WHERE department_id = $1 AND tenant_id = $2),
		        (SELECT count(*) FROM departments  WHERE parent_id     = $1 AND tenant_id = $2)`,
		id, tenantID).Scan(&people, &units)
	return people, units, err
}

func (s *Store) DeleteDepartment(ctx context.Context, tenantID, id string) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM departments WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func isUniqueViolation(err error) bool { return sqlState(err) == "23505" }

func isForeignKeyViolation(err error) bool { return sqlState(err) == "23503" }

func sqlState(err error) string {
	if err == nil {
		return ""
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState()
	}
	return ""
}

var _ organisation.Repository = (*Store)(nil)
