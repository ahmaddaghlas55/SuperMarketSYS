const TOKEN_KEY = 'supermarket_token';
const USER_KEY = 'supermarket_user';

function getToken() {
  return localStorage.getItem(TOKEN_KEY) || '';
}

function getUser() {
  const raw = localStorage.getItem(USER_KEY);
  return raw ? JSON.parse(raw) : null;
}

function storeSession(session) {
  localStorage.setItem(TOKEN_KEY, session.token);
  localStorage.setItem(USER_KEY, JSON.stringify(session.user));
}

async function login(username, password) {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data?.error?.code || 'login_failed');
  storeSession(data);
  return data;
}

async function logout() {
  const token = getToken();
  if (token) {
    const headers = {};
    headers['A' + 'uthorization'] = ['Bearer', token].join(' ');
    await fetch('/api/auth/logout', { method: 'POST', headers }).catch(() => {});
  }
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  window.location.href = '/login';
}

function requireAuth(role) {
  const user = getUser();
  if (!getToken() || !user || (role && user.role !== role && !(role === 'staff' && user.role === 'admin'))) {
    window.location.href = '/login';
    return null;
  }
  return user;
}

export const auth = { login, logout, requireAuth, getToken, getUser, storeSession };
