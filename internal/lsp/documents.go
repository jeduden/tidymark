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

func (s *documentStore) get(uri string) (*document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.m[uri]
	if !ok {
		return nil, false
	}
	// Return a shallow copy so callers can't race the stored slice.
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
