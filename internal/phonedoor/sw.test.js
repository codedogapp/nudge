import assert from 'node:assert/strict';
import { beforeEach, test } from 'node:test';

// The worker's globals, faked once: the worker adds its listeners when it's imported.
const listeners = {};
const calls = [];
globalThis.self = {
  addEventListener: (type, listener) => {
    listeners[type] = listener;
  },
  skipWaiting: async () => calls.push('skip waiting'),
  registration: { showNotification: async (title, options) => calls.push(`show ${title}: ${options.body}`) },
};
await import('./static/sw.js');

// windows fakes the worker's clients, with the windows of the page that are open.
function windows(...urls) {
  const open = urls.map((url) => ({
    focus: async () => calls.push(`focus ${url}`),
    navigate: async (to) => calls.push(`navigate ${url} to ${to}`),
  }));
  return {
    claim: async () => calls.push('claim'),
    matchAll: async (options) => {
      calls.push(`match ${options.type}`);
      return open;
    },
    openWindow: async (url) => calls.push(`open ${url}`),
  };
}

// fire hands the worker an event, and waits for what the worker asks it to wait for.
async function fire(type, fields = {}) {
  const waits = [];
  listeners[type]({ ...fields, waitUntil: (promise) => waits.push(promise) });
  await Promise.all(waits);
}

// payload fakes a push's data, whose json() throws on text that isn't JSON.
const payload = (text) => ({ json: () => JSON.parse(text) });

beforeEach(() => {
  calls.length = 0;
  globalThis.self.clients = windows();
});

test('a new worker takes over the open pages at once', async () => {
  await fire('install');
  await fire('activate');

  assert.deepEqual(calls, ['skip waiting', 'claim']);
});

test('a push shows its title and text', async () => {
  await fire('push', { data: payload('{"title": "atlas", "body": "Ship the new ranking?"}') });

  assert.deepEqual(calls, ['show atlas: Ship the new ranking?']);
});

test("a push that can't be read still shows a notification", async (t) => {
  const cases = {
    'no payload': null,
    'text that is not JSON': payload('Test push message from DevTools.'),
    'JSON null': payload('null'),
    'no body': payload('{"title": "atlas"}'),
    'a title that is not a string': payload('{"title": 7, "body": "Ship?"}'),
  };
  for (const [name, data] of Object.entries(cases)) {
    await t.test(name, async () => {
      calls.length = 0;
      await fire('push', { data });

      assert.deepEqual(calls, ['show nudge: A new question']);
    });
  }
});

test('a tap focuses an open page and loads its list again', async () => {
  globalThis.self.clients = windows('https://nudge.example/', 'https://nudge.example/?older');
  const notification = { close: () => calls.push('close') };

  await fire('notificationclick', { notification });

  assert.deepEqual(calls, [
    'close',
    'match window',
    'focus https://nudge.example/',
    'navigate https://nudge.example/ to /',
  ]);
});

test('a tap opens the page when none is open', async () => {
  const notification = { close: () => calls.push('close') };

  await fire('notificationclick', { notification });

  assert.deepEqual(calls, ['close', 'match window', 'open /']);
});

// held stands in for a browser call that hasn't finished yet: its promise settles on release().
function held() {
  let release;
  const promise = new Promise((resolve) => {
    release = resolve;
  });
  return { call: () => promise, release };
}

test('each event keeps the worker alive until its work is done', async (t) => {
  const { registration } = globalThis.self;
  t.after(() => {
    globalThis.self.registration = registration;
  });
  const events = {
    activate: (call) => {
      globalThis.self.clients.claim = call;
      return {};
    },
    push: (call) => {
      globalThis.self.registration = { showNotification: call };
      return { data: payload('{"title": "atlas", "body": "Ship it?"}') };
    },
    notificationclick: (call) => {
      globalThis.self.clients.openWindow = call;
      return { notification: { close: () => {} } };
    },
  };
  for (const [type, fake] of Object.entries(events)) {
    await t.test(type, async () => {
      const work = held();
      let done = false;
      const firing = fire(type, fake(work.call)).then(() => {
        done = true;
      });

      await new Promise(setImmediate);
      assert.equal(done, false);
      work.release();
      await firing;
      assert.equal(done, true);
    });
  }
});
