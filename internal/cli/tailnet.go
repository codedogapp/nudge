package cli

import "net/http"

// agentPrefix is where the phone door serves the agent door's endpoints, for agents on other
// machines (docs/plan.md, decision 35).
const agentPrefix = "/agent"

// tailnetDoor returns the phone door's handler with the agent door's under agentPrefix. The caller
// puts the lock in front of both.
func tailnetDoor(phone, agent http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", phone)
	mux.Handle(agentPrefix+"/", http.StripPrefix(agentPrefix, agent))

	return mux
}
