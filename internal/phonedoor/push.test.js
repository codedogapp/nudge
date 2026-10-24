import assert from 'node:assert/strict';
import { test } from 'node:test';

import { start } from './static/push.js';

const base64url = (bytes) => bytes.toBase64({ alphabet: 'base64url', omitPadding: true });
const pageKey = new Uint8Array(65).fill(4);
const oldKey = new Uint8Array(65).fill(5);
const accepted = () => new Response(null, { status: 204 });

// pushSubscription fakes a subscription made with key.
function pushSubscription(key, calls) {
  return {
    options: { applicationServerKey: key?.buffer ?? null },
    toJSON: () => ({ endpoint: 'https://push.example/1', keys: { p256dh: 'BP', auth: 'AU' } }),
    unsubscribe: async () => calls.push('unsubscribe'),
  };
}

// browser fakes the globals that start uses, and records the calls that the page makes.
function browser({ push = true, permission = 'default', subscribed, reply = accepted } = {}) {
  const calls = [];
  const posts = [];
  const element = () => ({
    hidden: true,
    textContent: '',
    addEventListener(type, listener) {
      this[type] = listener;
    },
  });
  const elements = { 'push-state': element(), push: element(), 'push-failure': element() };
  const pushManager = {
    getSubscription: async () => (subscribed === undefined ? null : pushSubscription(subscribed, calls)),
    subscribe: async (options) => {
      calls.push(`subscribe ${options.userVisibleOnly} ${options.applicationServerKey}`);
      return pushSubscription(pageKey, calls);
    },
  };
  const register = async (url) => calls.push(`register ${url}`);
  const window = {
    document: { body: { dataset: { pushKey: base64url(pageKey) } }, getElementById: (id) => elements[id] },
    navigator: { serviceWorker: { register, ready: Promise.resolve({ pushManager }) } },
    Notification: { permission },
    fetch: async (url, init) => {
      calls.push(`${init.method} ${url}`);
      posts.push(init);
      return reply();
    },
  };
  if (push) {
    window.PushManager = class PushManager {};
  }
  return { window, calls, posts, elements, pushManager };
}

// view is what the page shows: the state line, whether the button is there, and the alert.
function view({ elements }) {
  const shown = (element) => (element.hidden ? null : element.textContent);
  return {
    state: shown(elements['push-state']),
    button: !elements.push.hidden,
    failure: shown(elements['push-failure']),
  };
}

test('a browser without a push manager needs the Home Screen app', async () => {
  const page = browser({ push: false });
  await start(page.window);

  assert.deepEqual(view(page), { state: 'Notifications need the Home Screen app', button: false, failure: null });
  assert.deepEqual(page.calls, []);
});

test('blocked notifications show no button', async () => {
  const page = browser({ permission: 'denied' });
  await start(page.window);

  assert.deepEqual(view(page), { state: 'Notifications blocked', button: false, failure: null });
  assert.deepEqual(page.calls, []);
});

test('a tap subscribes with the page key and sends the subscription', async () => {
  const page = browser();
  await start(page.window);

  assert.deepEqual(view(page), { state: null, button: true, failure: null });
  assert.deepEqual(page.calls, ['register /sw.js']);

  await page.elements.push.click();

  assert.deepEqual(view(page), { state: 'Notifications on', button: false, failure: null });
  assert.deepEqual(page.calls, ['register /sw.js', `subscribe true ${base64url(pageKey)}`, 'POST /subscriptions']);
  assert.deepEqual(page.posts, [
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{"endpoint":"https://push.example/1","keys":{"p256dh":"BP","auth":"AU"}}',
    },
  ]);
});

test('a subscription made with the page key is sent again on each visit', async () => {
  const page = browser({ subscribed: pageKey });
  await start(page.window);

  assert.deepEqual(view(page), { state: 'Notifications on', button: false, failure: null });
  assert.deepEqual(page.calls, ['register /sw.js', 'POST /subscriptions']);
});

test('a subscription made with another key is dropped', async () => {
  for (const key of [oldKey, null]) {
    const page = browser({ subscribed: key });
    await start(page.window);

    assert.deepEqual(view(page), { state: null, button: true, failure: null });
    assert.deepEqual(page.calls, ['register /sw.js', 'unsubscribe']);
  }
});

test('a failure shows its reason', async (t) => {
  const tests = [
    {
      name: 'the worker',
      page: () => {
        const page = browser();
        page.window.navigator.serviceWorker.register = async () => {
          throw new TypeError('Failed to register a ServiceWorker');
        };
        return page;
      },
      failure: 'Notifications failed: Failed to register a ServiceWorker',
    },
    {
      name: 'a refused visit',
      page: () =>
        browser({ subscribed: pageKey, reply: () => new Response('the endpoint is not https\n', { status: 400 }) }),
      failure: 'Notifications failed: the endpoint is not https',
    },
    {
      name: 'a lost server',
      page: () => browser({ subscribed: pageKey, reply: () => new Response('', { status: 502 }) }),
      failure: 'Notifications failed: status 502',
    },
  ];
  for (const { name, page: make, failure } of tests) {
    await t.test(name, async () => {
      const page = make();
      await start(page.window);

      assert.deepEqual(view(page), { state: null, button: false, failure });
    });
  }
});

test('the button waits while a tap is under way', async () => {
  const page = browser();
  await start(page.window);
  let subscribed;
  page.pushManager.subscribe = () =>
    new Promise((resolve) => {
      subscribed = resolve;
    });

  const tap = page.elements.push.click();
  assert.equal(page.elements.push.disabled, true);

  subscribed(pushSubscription(pageKey, page.calls));
  await tap;
  assert.equal(page.elements.push.disabled, false);
  assert.deepEqual(view(page), { state: 'Notifications on', button: false, failure: null });
});

test('a failed tap leaves the button for another try', async () => {
  const page = browser();
  await start(page.window);
  page.pushManager.subscribe = async () => {
    throw new DOMException('Registration failed - permission denied', 'NotAllowedError');
  };

  await page.elements.push.click();

  assert.deepEqual(view(page), {
    state: null,
    button: true,
    failure: 'Notifications failed: Registration failed - permission denied',
  });
  assert.equal(page.elements.push.disabled, false);

  page.window.fetch = async () => {
    throw new TypeError('Failed to fetch');
  };
  page.pushManager.subscribe = async () => pushSubscription(pageKey, page.calls);
  await page.elements.push.click();

  assert.deepEqual(view(page), { state: null, button: true, failure: 'Notifications failed: Failed to fetch' });

  page.window.fetch = async () => accepted();
  await page.elements.push.click();

  assert.deepEqual(view(page), { state: 'Notifications on', button: false, failure: null });
});
