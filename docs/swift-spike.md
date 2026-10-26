# Swift spike: a native iOS app

What a native iOS app, written in Swift, would bring to nudge and what it would cost. The user asked for it on 2026-10-25 ([the plan](plan.md), decision 32). This is desk research from Apple's documentation, checked on 2026-10-25. Nothing was built: this Mac has the Command Line Tools and no Xcode, so it can't build an iOS app, and the spike's folder holds no code.

## What the app would bring

- **Push without a web page's limits.** Apple Push Notification service (APNs) sends to a native app. The web app's push ([the plan](plan.md), fact 4) works already, and needs the Home Screen and a tailnet certificate; a native app needs neither.
- **Answers from the notification.** Actionable notifications show buttons that "send the selected action to the app, which then processes the action in the background", without opening the app. A category per question's options could answer a question from the lock screen. Options are dynamic, and a category is declared at launch, so the options would need a few generic slots, or a notification content extension; this is unchecked.
- **A native look**, and a place for later ideas (the plan's Ideas list), such as Live Activities. Not checked.

## What it would cost

- **A paid Apple Developer Program membership.** Apple lists Push Notifications under the membership's services. A free account gets a personal team, whose App IDs, devices and provisioning profiles expire after 7 days: the app must be rebuilt and reinstalled each week. Whether a personal team may use the push capability wasn't confirmed: Apple's capability table lost its checkmarks in the fetch. The annual fee wasn't checked.
- **Xcode**, which this Mac doesn't have, and an App Store or TestFlight route (or a weekly reinstall) to keep the app on the phone.
- **A new door on the server.** APNs needs the server to hold an APNs key and send to Apple's servers with it, and the app needs the server's address and the user's tailnet login (Tailscale's iOS app, or the same `tailscale serve`). The server's push package would get a second sender next to Web Push.
- **A second client to build, test and release**, in a second language, outside the Go gates in AGENTS.md (the lint, coverage and SonarQube rules don't cover Swift as they stand).

## What stays

The Go server, the agent door, the sessions and the tailnet lock stay as they are: the app would be another phone-side client of the phone door's pages or of a JSON API on it. A rewrite of the server in Swift would bring nothing for the phone, and isn't what the user asked for.

## Verdict

Not worth it yet. The web app already gets the push on the user's iPhone (risk 2 passed on 2026-10-25), so the native app's gain is answering from the lock screen, against a yearly fee, a new toolchain and a second client. Reconsider if the notification's tap-then-answer flow proves too slow in daily use. A cheaper first test, once Xcode is installed and the user has a membership: a small app that registers for APNs and shows one notification with buttons, to see whether options can be actionable.

## Sources

- [Declaring your actionable notification types](https://developer.apple.com/documentation/usernotifications/declaring-your-actionable-notification-types), Apple.
- [Developer account overview](https://developer.apple.com/help/account/basics/about-your-developer-account), Apple: what a free account gets.
- [Membership details](https://developer.apple.com/programs/whats-included/), Apple.
- [Supported capabilities (iOS)](https://developer.apple.com/help/account/reference/supported-capabilities-ios), Apple.
