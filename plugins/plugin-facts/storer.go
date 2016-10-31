package pluginfacts

import "strings"

// factStorer interface
type factStorer interface {
	AddFact(*fact)
	DelFact(name string)
	ListFacts() []string
	NumberOfFacts() int
	FindFact(message string) string
}

// Simple store in an array
type inMemStore struct {
	facts []*fact
}

func (s *inMemStore) AddFact(f *fact) {
	s.facts = append(s.facts, f)
}

func (s *inMemStore) NumberOfFacts() int {
	return len(s.facts)
}

func (s *inMemStore) ListFacts() (factlist []string) {
	for _, f := range s.facts {
		factlist = append(factlist, f.Name)
	}
	return factlist
}

func (s *inMemStore) DelFact(name string) {
	for i, f := range s.facts {
		if f.Name == name {
			s.facts = append(s.facts[:i], s.facts[i+1:]...)
			return
		}
	}
}

func (s *inMemStore) FindFact(message string) string {
	for _, f := range s.facts {
		for _, p := range f.Patterns {
			if strings.Contains(message, p) {
				return f.Content
			}
		}
	}
	return ""
}
