package controlplane

// Role is what an operator is allowed to be. The four come from §2.2 of the
// plan and are stored as text, checked by the database as well (migration
// 00049), because an unrecognised role must not be a role that quietly means
// "superadmin" to a `switch` with no default.
type Role string

const (
	// RoleSuperadmin may do everything, including the two-person actions —
	// it is also the only role that can approve another superadmin's.
	RoleSuperadmin Role = "superadmin"
	// RoleOperator does the daily work: tenants, settings, deployments. Not
	// deletion, and nothing that reveals a person's data.
	RoleOperator Role = "operator"
	// RoleSupport answers for people: find an account, reset it, look inside
	// an organisation with consent and a reason.
	RoleSupport Role = "support"
	// RoleAuditor reads. Every screen, no button.
	RoleAuditor Role = "auditor"
)

// Capability is one thing a role may do. Roles are compared against these
// rather than against each other, so that adding a role later is a row in one
// table instead of an inequality every handler got slightly differently.
type Capability string

const (
	// CapTenantRead is the organisation list and its detail pages.
	CapTenantRead Capability = "tenant.read"
	// CapAuditRead is the operator audit trail. Every role has it: a console
	// where the people using it cannot see what they did to each other is not
	// one anybody should trust.
	CapAuditRead Capability = "audit.read"
	// CapOperatorRead is the roster of operators. Who can reach this platform
	// is itself a thing to be able to check.
	CapOperatorRead Capability = "operator.read"
)

// capabilities is the whole authorization model, in one readable place.
//
// CP-1 grants little because CP-1 does little: everything here is read-only.
// The phases after it add their own rows — tenant.suspend, tenant.delete,
// user.impersonate, settings.write, deploy.trigger — and the reason to write
// the table now, with three entries, is that they will be added by editing
// this map rather than by scattering `if role == "superadmin"` through
// handlers, which is where privilege bugs live.
var capabilities = map[Role]map[Capability]bool{
	RoleSuperadmin: {CapTenantRead: true, CapAuditRead: true, CapOperatorRead: true},
	RoleOperator:   {CapTenantRead: true, CapAuditRead: true, CapOperatorRead: true},
	RoleSupport:    {CapTenantRead: true, CapAuditRead: true, CapOperatorRead: false},
	RoleAuditor:    {CapTenantRead: true, CapAuditRead: true, CapOperatorRead: true},
}

// Can reports whether this role holds a capability. An unknown role holds
// none — a row whose `role` column was written by something that did not know
// the list ends up able to sign in and do nothing, rather than able to do
// everything.
func (r Role) Can(c Capability) bool { return capabilities[r][c] }

// Valid reports whether r is one of the four.
func (r Role) Valid() bool { _, known := capabilities[r]; return known }

// Roles lists the four in the order they appear in the plan, most privileged
// first. Used by the bootstrap command's help text and by the console's UI.
func Roles() []Role { return []Role{RoleSuperadmin, RoleOperator, RoleSupport, RoleAuditor} }

// Operator is an account, as the console needs to know it.
type Operator struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  Role   `json:"role"`
}
