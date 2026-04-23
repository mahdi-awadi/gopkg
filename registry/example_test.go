package registry_test

import (
	"fmt"
	"sort"

	"github.com/mahdi-awadi/gopkg/registry"
)

// Plugin is the sort of thing you'd register: an interface value.
type Plugin interface{ Describe() string }

type plainPlugin struct{ name string }

func (p *plainPlugin) Describe() string { return "plugin-" + p.name }

func ExampleRegistry() {
	r := registry.New[string, Plugin]()
	_ = r.Register("alpha", &plainPlugin{name: "alpha"})
	_ = r.Register("beta", &plainPlugin{name: "beta"})

	keys := r.Keys()
	sort.Strings(keys)
	for _, k := range keys {
		p, _ := r.Get(k)
		fmt.Println(p.Describe())
	}
	// Output:
	// plugin-alpha
	// plugin-beta
}

// A "pending" queue is useful when each plugin wants to self-register
// from its own package init() without forcing the main package to know
// about it. Late init() calls enqueue registrations; the main package
// builds the registry and flushes the queue in one place.
var globalQueue registry.PendingQueue[string, Plugin]

func init() {
	// Pretend this ran in an imported plugin's own init().
	globalQueue.Add(func(r *registry.Registry[string, Plugin]) error {
		return r.Register("deferred", &plainPlugin{name: "deferred"})
	})
}

func ExamplePendingQueue() {
	r := registry.New[string, Plugin]()
	_ = globalQueue.Flush(r)
	p, _ := r.Get("deferred")
	fmt.Println(p.Describe())
	// Output:
	// plugin-deferred
}
