package agentdoor

// Question is a question as it crosses the agent door.
type Question struct {
	Project string   `json:"project"`
	Text    string   `json:"text"`
	Details string   `json:"details"`
	Options []string `json:"options"`
}

// Answer is an answer as it crosses the agent door.
type Answer struct {
	Option string `json:"option"`
	Text   string `json:"text"`
}

// Start is a request to start a session, as it crosses the agent door.
type Start struct {
	Project string `json:"project"`
}

// Started is the reply to a start: the new session's ID.
type Started struct {
	ID string `json:"id"`
}

// Report is an agent's note for its session, as it crosses the agent door.
type Report struct {
	Note string `json:"note"`
}
