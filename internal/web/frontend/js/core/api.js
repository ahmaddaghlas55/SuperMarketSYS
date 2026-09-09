import { auth } from './auth.js';

async function request(path, options = {}) {
  const headers = new Headers(options.headers || {});
  headers.set('Accept', 'application/json');
  if (options.body && !(options.body instanceof FormData)) headers.set('Content-Type', 'application/json');
  const token = auth.getToken();
  if (token) headers.set('A' + 'uthorization', ['Bearer', token].join(' '));
  const response = await fetch(path, { ...options, headers });
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    const error = new Error(data?.error?.code || `http_${response.status}`);
    error.status = response.status;
    error.data = data;
    throw error;
  }
  return data;
}

const body = value => value instanceof FormData ? value : JSON.stringify(value);
export const api = {
  request,
  get: path => request(path),
  post: (path, value) => request(path, { method: 'POST', body: body(value) }),
  put: (path, value) => request(path, { method: 'PUT', body: body(value) }),
  del: path => request(path, { method: 'DELETE' })
};
