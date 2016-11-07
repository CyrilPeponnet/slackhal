package pluginfacts

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/nlopes/slack"
)

const (
	// Pattern const
	Pattern = 1
	// Content const
	Content = 2
	// Scope const
	Scope = 3
)

// fact struct
type fact struct {
	Name                 string `storm:"id"`
	Patterns             []string
	Content              string
	RestrictToChannelsID []string
}

// Chatentries struct
type learningFact struct {
	Channel string
	User    string
	Fact    fact
	State   int
}

// learn struct
type learn struct {
	entries []*learningFact
	lock    sync.Mutex
}

// New will start a new fact learning session.
func (f *learn) New(message slack.Msg) string {
	f.lock.Lock()
	defer f.lock.Unlock()
	for _, f := range f.entries {
		if f.Channel == message.Channel && f.User == message.User {
			return fmt.Sprintf("Sorry <@%v> be we are still learning _%v_. If you want to cancel send me _stop-learning_", message.User, f.Fact.Name)
		}
	}
	currentFact := strings.TrimSpace(message.Text[strings.Index(message.Text, cmdnew)+len(cmdnew) : len(message.Text)])
	f.entries = append(f.entries, &learningFact{Channel: message.Channel, User: message.User, Fact: fact{Name: currentFact, RestrictToChannelsID: []string{}}, State: Content})
	return fmt.Sprintf("Ok <@%v> let's do that! Can you define _%v_? \n(type stop-learning to stop this learning session)", message.User, currentFact)
}

// Cancel a pending learning session.
func (f *learn) Cancel(message slack.Msg) string {
	f.lock.Lock()
	defer f.lock.Unlock()
	for i, e := range f.entries {
		if e.Channel == message.Channel && e.User == message.User {
			f.entries = append(f.entries[:i], f.entries[i+1:]...)
			return fmt.Sprintf("Ok <@%v>, let's do that later then.", message.User)
		}
	}
	return fmt.Sprintf("Sorry <@%v>, no learning session are pending.", message.User)
}

// Learn will contine a learning session and return the fact once done.
func (f *learn) Learn(message slack.Msg) (fact fact, response string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	for i, e := range f.entries {
		if e.Channel == message.Channel && e.User == message.User {
			switch e.State {
			case Content:
				// Store the content and update the state
				e.Fact.Content = strings.TrimSpace(message.Text)
				e.State = Pattern
				return fact, fmt.Sprintf("Got it <@%v>. And now can you tell me list of pattern I should match for this fact (Use || as separator).", message.User)
			case Pattern:
				// Store the patterns and remove the learning session
				patterns := strings.TrimSpace(message.Text)
				for _, pattern := range strings.Split(patterns, "||") {
					e.Fact.Patterns = append(e.Fact.Patterns, strings.TrimSpace(pattern))
				}
				e.State = Scope
				return fact, fmt.Sprintf("One last things <@%v>, in which channel(s) should I check those patterns? (all or #chan1 #chan2...)", message.User)
			case Scope:
				if strings.ToLower(message.Text) != "all" {
					re := regexp.MustCompile(`<#(\S+)+\|\S+>`)
					for _, m := range re.FindAllStringSubmatch(message.Text, -1) {
						e.Fact.RestrictToChannelsID = append(e.Fact.RestrictToChannelsID, m[1])
					}
				}
				fact = e.Fact
				f.entries = append(f.entries[:i], f.entries[i+1:]...)
				return fact, fmt.Sprintf("All good! I'll keep that in mind.")
			}
		}
	}
	return
}
