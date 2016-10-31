package pluginfacts

import (
	"strings"

	"github.com/asdine/storm"
)

// factStorer interface
type factStorer interface {
	Connect(string) error
	AddFact(*fact) error
	DelFact(name string) error
	ListFacts() []fact
	NumberOfFacts() int
	FindFact(message string) *fact
}

//Storm orm db
type stormDB struct {
	db *storm.DB
}

func (s *stormDB) Connect(dbPath string) (err error) {
	s.db, err = storm.Open(dbPath)
	return err
}

func (s *stormDB) AddFact(f *fact) (err error) {
	return s.db.Save(f)
}

func (s *stormDB) NumberOfFacts() int {
	n, _ := s.db.Count(&fact{})
	return n
}

func (s *stormDB) ListFacts() (factlist []fact) {
	s.db.All(&factlist)
	return
}

func (s *stormDB) DelFact(name string) (err error) {
	var f fact
	err = s.db.One("Name", name, &f)
	if err != nil {
		s.db.DeleteStruct(&f)
	}
	return
}

func (s *stormDB) FindFact(message string) *fact {
	var factList []fact
	err := s.db.All(&factList)
	if err == nil {
		for _, f := range factList {
			for _, p := range f.Patterns {
				if strings.Contains(message, p) {
					return &f
				}
			}
		}
	}
	return nil
}
