package lsp

import "sync"

// document is one open buffer in the editor.
type document struct {
	uri     string
	path    string
	text    []byte
	version int
}

// documentStore is a goroutine-safe map of open documents keyed by URI.
type documentStore struct {
	mu sync.RWMutex
	m  map[string]*document
}

func newDocumentStore() *documentStore {
	return &documentStore{m: make(map[string]*document)}
}

// get returns a shallow copy of the stored document. The copy
// prevents callers from racing the stored *document pointer (e.g.
// when set() replaces it), but does not deep-copy the `text` byte
// slice — both copies share the underlying array. Callers must treat
// `text` as read-only and never mutate it in place. Updates go
// through set() with a fresh slice instead.
func (s *documentStore) get(uri string) (*document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.m[uri]
	if !ok {
		return nil, false
	}
	cp := *d
	return &cp, true
}

func (s *documentStore) set(uri string, d *document) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *d
	s.m[uri] = &cp
}

func (s *documentStore) delete(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, uri)
}

func (s *documentStore) openURIs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.m))
	for k := range s.m {
		out = append(out, k)
	}
	return out
}
