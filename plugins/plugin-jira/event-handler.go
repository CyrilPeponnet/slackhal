package jiraplugin

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/andygrunwald/go-jira"
)

type eventHandler struct {
	IssueEvents chan *jiraEvent
}

// New eventHandler
func newEventHandler() *eventHandler {
	return &eventHandler{
		IssueEvents: make(chan *jiraEvent),
	}
}

type jiraEvent struct {
	Issue        jira.Issue `json:"issue"`
	WebhookEvent string     `json:"webhookEvent"`
}

func (j eventHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Handle events
	if req.Method != "POST" {
		http.Error(w, req.Method, http.StatusMethodNotAllowed)
		return
	}

	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	var event jiraEvent

	_ = json.Unmarshal(body, &event)

	j.IssueEvents <- &event
}
