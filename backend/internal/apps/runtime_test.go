/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package apps

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Bootstrap takes the platform and nothing else.
//
// This signature is the reason pkg/nexus/capability.go exists. Between
// 2026-08-09 and 2026-08-20 — eleven days — it went from four parameters to
// nine, in seven separate changes:
//
//	0ce4229  2026-08-09   4
//	f292b64  2026-08-12   5
//	8c40fac  2026-08-13   7
//	29b4549  2026-08-14   7   (types changed, arity did not)
//	2c02555  2026-08-15   8
//	5e2b48d  2026-08-16   8   (types changed, arity did not)
//	bb6f952  2026-08-16   8
//	bbb8240  2026-08-20   9
//
// None of the seven was noticed, and the reason they were not is worth being
// precise about: this package is under internal/, so no distribution compiles
// against it and no build outside this repository could report the change. The
// growth was not a series of decisions to widen an interface. It was what
// happened each time the platform had something new to lend, because a
// parameter was the only place to put it.
//
// There is now somewhere else to put it, so widening this is a decision again,
// and this test is where the decision has to be made out loud.
func TestBootstrapTakesOnlyThePlatform(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "runtime.go", nil, 0)
	if err != nil {
		t.Fatalf("parse runtime.go: %v", err)
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Bootstrap" || fn.Recv != nil {
			continue
		}

		count := 0
		for _, field := range fn.Type.Params.List {
			// One field can name several parameters: `a, b nexus.Platform`.
			count += max(len(field.Names), 1)
		}
		if count != 1 {
			t.Errorf(`Bootstrap takes %d parameters, not 1.

It takes the platform and nothing else. Anything more the platform lends its
modules is published with nexus.Provide in server.go and asked for here with
nexus.Capability, so that adding one is a line rather than a signature.

If a dependency genuinely cannot go through the registry, say why in a comment
beside it and change the number this test expects — deliberately, which is the
whole point.`, count)
		}
		return
	}
	t.Fatal("runtime.go declares no Bootstrap")
}
