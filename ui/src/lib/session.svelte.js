import { get, post, setCsrf, setUnauthorizedHandler } from './api.js';

export const app = $state({
  ready: false,
  boot: null, // bootstrap info
  me: null, // {user, role, csrf, totp}
  status: null, // latest snapshot
  scenario: null,
});

export const isAdmin = () => app.me?.role === 'admin';

export function signedIn(me) {
  setCsrf(me.csrf);
  app.me = me;
}

setUnauthorizedHandler(() => {
  app.me = null;
  setCsrf('');
});

export async function init() {
  app.boot = await get('/bootstrap');
  if (!app.boot.setupRequired) {
    try {
      signedIn(await get('/me'));
    } catch {
      app.me = null;
    }
  }
  app.ready = true;
}

export async function logout() {
  try {
    await post('/logout');
  } finally {
    app.me = null;
    setCsrf('');
  }
}

export async function refreshStatus() {
  if (!app.me) return;
  try {
    const r = await get('/status');
    app.status = r.snapshot;
    app.scenario = r.scenario ?? null;
  } catch {
    /* keep last snapshot */
  }
}
