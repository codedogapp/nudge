# Moving nudge into Bakhyard

Draft, 2026-10-28, for the Bakhyard session to take over. The user's choices (2026-10-28): there is one app, not two; nudge becomes a Bakhyard service plus a page in the Bakhyard app, and this repo is archived; agents reach it through the gateway on a tailnet-only route that skips Google sign-in; the client CLI (`ask`, `mcp`, `setup`) and the skill live in the Bakhyard repo, which is public.

Until the cutover step, the nudge in the cluster (`cluster/nudge`) keeps serving: other sessions ask through it every day.

## What goes where

| Today (this repo) | In Bakhyard |
| --- | --- |
| `internal/inbox`, `sessions`, `agentdoor` (questions, notes, the held request) | `services/nudge`, a service behind the gateway like `todos` and `pulls` |
| `internal/phonedoor`: the page, the answer forms, the Sessions section | A route of the Svelte app (`/ask`), over `/api/nudge/...` |
| `internal/push`, `vapid`, `subscriptions` | The planned `notify` service (decision 64): VAPID key in a Secret, subscriptions in Postgres (`push_subscriptions`), one internal "send" call |
| The Tailscale lock (`Tailscale-User-Login`) | The gateway's: Google sign-in for `/api`, tailnet login alone for `/agent/` |
| `nudge ask`, `mcp`, `setup`, the skill | A `main` package in the Bakhyard repo (say `tools/nudge`), built and released from there |
| `cluster/nudge`, the image, the Gitea mirror of this repo | Gone: the service gets Bakhyard's manifests and image pipeline |

## The requirement: no refresh when the app is open

A question, a report or an answer must show up in an open app without a refresh. Two layers, both needed:

1. **Push while the app is open.** The service worker posts a message to every open window of the app on each `push` event, and the page refetches on it. The notification is still shown: iOS revokes a subscription whose pushes show nothing, so a silent push isn't an option.
2. **A live stream.** `GET /api/nudge/events` is a server-sent events stream that says "something changed"; the page refetches `/api/nudge/questions` on each event. It is the "step that needs faster updates" that Bakhyard's decisions 43 and 49 wait for. The gateway already flushes streams at once. Fallbacks: the 60 s poll while visible, and a refetch when the page becomes visible again, because iOS suspends a Home Screen app's connections in the background.

Home's "Needs you today" feed also lists the waiting questions (one row each, linking to `/ask`), through the same attention events as the other apps.

## Agents

- Agents on any machine call `https://<bakhyard host>/agent/questions` and `/agent/sessions` (today's paths). The gateway checks the tailnet login header only, for `/agent/`, and sends nothing else past the lock.
- A question is held open for as long as its request. The gateway must not time those requests out (Bakhyard's decision 41 leaves hung upstreams to the gateway), and must not for the event stream either. Nudge's check of the operator's proxy (risk 12) showed 330 s; hours are still untested.
- `nudge setup SERVER` takes the Bakhyard host. Every machine runs it once again at the cutover.

## Steps

Each is one small PR, in the Bakhyard repo unless noted.

| Id | Step |
| --- | --- |
| `notify-service` | `notify`: the VAPID key, subscriptions in Postgres, `POST` to send; the app's "turn on notifications" button and the service worker |
| `nudge-service` | `services/nudge`: questions, held requests and sessions in memory, moved with their tests; behind the gateway at `/api/nudge` |
| `agent-route` | The gateway's `/agent/` route: tailnet login only, no timeout on held requests |
| `ask-page` | The `/ask` route: pending questions, answer forms, Sessions |
| `ask-live` | The event stream, the worker's message to open windows, the visibility refetch (the requirement above) |
| `ask-feed` | Waiting questions in Home's "Needs you today" |
| `nudge-push` | A new question calls `notify`'s send |
| `client-move` | `tools/nudge`: `ask`, `mcp`, `setup` and the skill; `go install` path and releases from Bakhyard |
| `deploy-nudge` | Bakhyard's manifests carry the service and the route; the Flux image automation covers it (homelab repo) |
| `nudge-cutover` | Each machine runs `nudge setup`; the phone enables notifications in Bakhyard and the nudge Home Screen app goes; `cluster/nudge`, its namespace, PVC and Secrets, the Gitea mirror and the image go; this repo is archived (homelab repo, then here) |

`notify-service` comes first because Bakhyard's reminders need it too.

## Risks and open questions

1. **Questions are lost when a pod restarts, and Bakhyard rolls out often.** Flux rolls a service out on each push of its image. The nudge service has its own image, so only changes to it or to the gateway drop waiting questions, but the gateway also carries every other app's changes. Options: keep questions in memory (as today, and AGENTS.md's rule here), or keep pending questions in Postgres so a restart loses nothing and the agent's request re-attaches. For the user to decide.
2. **A phone subscription belongs to one origin.** The iPhone subscribes again from the Bakhyard app; the old subscriptions and VAPID key aren't carried over.
3. **One lock for two doors.** The `/agent/` route skips Google sign-in, so it must serve only those two paths and answer 403 without the tailnet login, with a test for each.
4. **Quality gates differ.** Bakhyard's file and function limits, its web gates (Biome, Playwright) and its Sonar rules apply to the moved code; the nudge page's vanilla JavaScript is rewritten in Svelte, not copied.
5. **The cluster Secret `nudge`** (the user's login) has no counterpart: Bakhyard's gateway already knows the user.
