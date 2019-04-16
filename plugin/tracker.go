package plugin

import (
	"strconv"
	"sync"
	"time"

	"github.com/nlopes/slack"
)

/*
The main purpose of this tracket is to be able to track sent message and edit them later
*/

// TrackerManager is keeping list of Trackers
type TrackerManager struct {
	// Contains the list of current trackers
	Trackers map[int]*Tracker
	// Lock to avoid concurent access to array
	lock sync.Mutex
}

// Tracker define a tracker
type Tracker struct {
	// The tracker id set by the plugin
	TrackerID int
	// The TimeStamp of the message to track
	TimeStamp string
	// The TTL of the Tracker before it's gargabe collected (in minutes)
	TTL int
}

// Init the TrackerManager and the garbageCollector
func (t *TrackerManager) Init() {
	t.Trackers = map[int]*Tracker{}
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			t.garbageCollector()
		}
	}()
}

// GetTimeStampFor will return the timestamp of the tracked conversation
func (t *TrackerManager) GetTimeStampFor(id int) string {
	if v, found := t.Trackers[id]; found {
		return v.TimeStamp
	}
	return ""
}

// Track add a new tracker to the TrackerManager
func (t *TrackerManager) Track(tracker Tracker) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.Trackers[tracker.TrackerID] = &tracker
}

// garbageCollector clean Tracker that exceed their TTL
func (t *TrackerManager) garbageCollector() {
	// Lock access to the array as we are doing some cleaning
	t.lock.Lock()
	defer t.lock.Unlock()

	var keys []int
	for _, t := range t.Trackers {
		ts, err := msToTime(t.TimeStamp)
		if err != nil {
			keys = append(keys, t.TrackerID)
			continue
		}
		durration := time.Since(ts)
		if int(durration.Minutes()) > t.TTL {
			keys = append(keys, t.TrackerID)
		}
	}
	// Remove items
	for _, i := range keys {
		delete(t.Trackers, i)
	}
}

// UpdateTracking will try to match a Pending Tracker with an event msg id
func (t *TrackerManager) UpdateTracking(ack *slack.AckMessage) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if tracker, found := t.Trackers[ack.ReplyTo]; found {
		tracker.TimeStamp = ack.Timestamp
	}
}

// Convert ns UNIX epoch to time.Unix
func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, msInt*int64(time.Millisecond)), nil
}
