import assert from 'node:assert/strict';
import { test } from 'node:test';

test('the page starts the notifications button', async () => {
  const state = { hidden: true, textContent: '' };
  globalThis.window = {
    document: { body: { dataset: {} }, getElementById: (id) => (id === 'push-state' ? state : null) },
  };

  await import('./static/main.js');

  assert.deepEqual(state, { hidden: false, textContent: 'Notifications need the Home Screen app' });
});
