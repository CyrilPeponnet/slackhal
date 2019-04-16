package pluginfacts

import (
	"strings"

	"go.uber.org/zap"

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
	FindFactByName(name string) *fact
}

//Storm orm db
type stormDB struct {
	db *storm.DB
}

func (s *stormDB) Connect(dbPath string) (err error) {
	if s.db == nil {
		s.db, err = storm.Open(dbPath)
	}
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
	if err := s.db.All(&factlist); err != nil {
		zap.L().Error("Error while getting facts", zap.Error(err))
	}
	return factlist
}

func (s *stormDB) DelFact(name string) (err error) {
	var f fact
	err = s.db.One("Name", name, &f)
	if err == nil {
		err = s.db.DeleteStruct(&f)
	}
	return err
}

func (s *stormDB) FindFact(message string) *fact {
	var factList []fact
	err := s.db.All(&factList)
	if err == nil {
		for _, f := range factList {
			for _, p := range f.Patterns {
				//TODO: Not really optmized.
				m := strings.ToLower(message)
				if p == m ||
					strings.HasPrefix(m, p) ||
					strings.HasSuffix(m, p) ||
					strings.Contains(m, " "+p+" ") {
					return &f
				}
			}
		}
	}
	return nil
}

func (s *stormDB) FindFactByName(name string) *fact {
	var f fact
	err := s.db.One("Name", name, &f)
	if err != nil {
		return nil
	}
	return &f
}
