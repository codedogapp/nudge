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
