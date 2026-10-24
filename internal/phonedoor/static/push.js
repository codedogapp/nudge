// The header's notifications button (decision 14): it subscribes this device to push with the
// page's VAPID key and gives the subscription to nudge serve.

const on = 'Notifications on';

// start shows the device's state, or the button that turns its notifications on. It takes the
// browser's globals, so the tests can pass fakes.
export async function start(window) {
  const page = elements(window.document);
  if (!('PushManager' in window)) {
    // A Safari tab on the iPhone: only the Home Screen app gets pushes.
    say(page, 'Notifications need the Home Screen app');
  } else if (window.Notification?.permission === 'denied') {
    say(page, 'Notifications blocked');
  } else {
    await connect(window, page).catch((error) => fail(page, error));
  }
}

function elements(document) {
  return {
    key: document.body.dataset.pushKey,
    state: document.getElementById('push-state'),
    button: document.getElementById('push'),
    failure: document.getElementById('push-failure'),
  };
}

// connect registers the service worker, which gives the page its push manager. A subscription made
// with the page's key goes to nudge serve again; one made with an earlier key is dropped.
async function connect(window, page) {
  const { serviceWorker } = window.navigator;
  await serviceWorker.register('/sw.js');
  const { pushManager } = await serviceWorker.ready;
  const subscription = await pushManager.getSubscription();
  if (subscription && sameKey(subscription, page.key)) {
    await send(window, subscription);
    say(page, on);
    return;
  }
  await subscription?.unsubscribe();
  page.button.hidden = false;
  page.button.addEventListener('click', () => subscribe(window, pushManager, page));
}

// subscribe asks for the permission, then subscribes. Safari allows it only within the tap, so
// nothing is awaited before pushManager.subscribe. The button waits meanwhile: a browser's first
// subscription can take half a minute.
async function subscribe(window, pushManager, page) {
  page.failure.hidden = true;
  page.button.disabled = true;
  try {
    const subscription = await pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: page.key });
    await send(window, subscription);
    page.button.hidden = true;
    say(page, on);
  } catch (error) {
    fail(page, error);
  } finally {
    page.button.disabled = false;
  }
}

// send gives the subscription to nudge serve. A refusal's reason is the server's text.
async function send(window, subscription) {
  const response = await window.fetch('/subscriptions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(subscription),
  });
  if (response.status !== 204) {
    const reason = (await response.text()).trim();
    throw new Error(reason || `status ${response.status}`);
  }
}

// sameKey reports whether the subscription was made with key, a base64url string.
function sameKey(subscription, key) {
  const used = subscription.options.applicationServerKey;
  return used !== null && new Uint8Array(used).toBase64({ alphabet: 'base64url', omitPadding: true }) === key;
}

function say(page, text) {
  page.state.textContent = text;
  page.state.hidden = false;
}

function fail(page, error) {
  page.failure.textContent = `Notifications failed: ${error.message}`;
  page.failure.hidden = false;
}
