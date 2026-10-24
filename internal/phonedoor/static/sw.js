// The service worker (decision 15). It shows each push as a notification, and a tap on one opens
// the page with the pending questions.

// Safari revokes a subscription whose pushes show nothing, so a push that can't be read shows this.
const fallback = { title: 'nudge', body: 'A new question' };

// A new binary's worker takes over the open pages at once, so a tap can load them again.
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (event) => event.waitUntil(self.clients.claim()));

self.addEventListener('push', (event) => {
  const { title, body } = message(event.data);
  event.waitUntil(self.registration.showNotification(title, { body }));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  event.waitUntil(open());
});

// message reads a push's payload: JSON with a title and a body, both strings.
function message(data) {
  try {
    const { title, body } = data.json();
    if (typeof title === 'string' && typeof body === 'string') {
      return { title, body };
    }
  } catch {
    // A push without a payload, or with one that isn't JSON.
  }
  return fallback;
}

// open focuses a window of the page and loads the list again, or opens one.
async function open() {
  const [page] = await self.clients.matchAll({ type: 'window' });
  if (!page) {
    await self.clients.openWindow('/');
    return;
  }
  await page.focus();
  await page.navigate('/');
}
