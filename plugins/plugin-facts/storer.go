package pluginfacts

import (
	"strings"

	"github.com/asdine/storm"
)

// fact struct
type fact struct {
	Name                 string `storm:"id"`
	Patterns             []string
	Content              string
	RestrictToChannelsID []string
}

// factStorer interface
type factStorer interface {
	Connect(string) error
	AddFact(*fact) error
	DelFact(name string) error
	ListFacts() ([]fact, error)
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

func (s *stormDB) ListFacts() (factlist []fact, err error) {
	if err := s.db.All(&factlist); err != nil {
		return nil, err
	}
	return factlist, nil
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
				if strings.Contains(strings.ToLower(message), strings.ToLower(p)) {
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
