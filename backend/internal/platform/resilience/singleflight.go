package resilience

import "sync"

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Singleflight struct {
	mu sync.Mutex
	m  map[string]*call
}

func NewSingleflight() *Singleflight {
	return &Singleflight{
		m: make(map[string]*call),
	}
}

// Do executes and returns the results of the given function, making sure that only one
// execution is in-flight for a given key at a time.
func (g *Singleflight) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
