/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 */

package nexus

import (
	"fmt"
	"sync"
)

// The compile-time module registry.
//
// A module joins the binary by calling Register from its constructor, and the
// platform asks this registry what is in the binary when it mounts routes,
// builds menus and checks that an installation has code behind it. It is
// package-level state on purpose: "which modules are compiled into this
// program" is a property of the program, and threading a registry value
// through every constructor would only let two of them disagree.
//
// This is what a distribution's main.go touches:
//
//	func main() {
//	    mymodule.New(db)   // calls nexus.Register itself
//	    platform.Run()
//	}
type registry struct {
	mu      sync.RWMutex
	modules map[string]Module
}

var global = &registry{modules: make(map[string]Module)}

// Register adds a compile-time module to the global registry.
//
// Last writer wins, by id. Two modules claiming one id is a build mistake
// rather than a runtime condition — both are compiled in by name — so this
// does not refuse it; the version check the platform runs against the
// catalogue is what surfaces the disagreement, and it names the id.
func Register(m Module) {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.modules[m.ID()] = m
}

// Get returns a module by ID from the global registry.
func Get(id string) (Module, bool) {
	global.mu.RLock()
	defer global.mu.RUnlock()
	m, ok := global.modules[id]
	return m, ok
}

// List returns every registered module.
//
// The order is the map's and therefore arbitrary. Callers that render a list to
// a person sort it themselves — see platform/menu, which sorts by app and by
// the order each module asked for.
func List() []Module {
	global.mu.RLock()
	defer global.mu.RUnlock()
	list := make([]Module, 0, len(global.modules))
	for _, m := range global.modules {
		list = append(list, m)
	}
	return list
}

// VerifyModuleExists reports whether an id names code in this binary.
//
// The store can offer an app that this build does not carry — the catalogue
// comes from a registry that serves many deployments — and installing one would
// otherwise write an installation row for routes that will never be mounted.
func VerifyModuleExists(id string) error {
	if _, ok := Get(id); !ok {
		return fmt.Errorf("module %s is not present in binary registry", id)
	}
	return nil
}
