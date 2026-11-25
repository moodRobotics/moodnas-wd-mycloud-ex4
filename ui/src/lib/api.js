// Thin fetch wrapper: JSON in/out, CSRF header on writes, errors carry an
// i18n key (`err.key`) plus params from the server.
let csrf = '';

export function setCsrf(token) {
  csrf = token || '';
}

export class ApiError extends Error {
  constructor(status, key, params) {
    super(key);
    this.status = status;
    this.key = key;
    this.params = params || {};
  }
}

let onUnauthorized = () => {};
export function setUnauthorizedHandler(fn) {
  onUnauthorized = fn;
}

export async function api(method, path, body) {
  const headers = {};
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (method !== 'GET' && csrf) headers['X-CSRF-Token'] = csrf;
  let res;
  try {
    res = await fetch('/api/v1' + path, {
      method,
      headers,
      credentials: 'same-origin',
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, 'error.network');
  }
  let data = null;
  try {
    data = await res.json();
  } catch {
    /* empty body */
  }
  if (!res.ok) {
    if (res.status === 401 && data?.error === 'auth.required') onUnauthorized();
    throw new ApiError(res.status, data?.error || 'error.unknown', data?.params);
  }
  return data;
}

export const get = (p) => api('GET', p);
export const post = (p, b = {}) => api('POST', p, b);
export const put = (p, b = {}) => api('PUT', p, b);
export const del = (p) => api('DELETE', p);
