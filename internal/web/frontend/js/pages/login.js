import { auth } from '../core/auth.js';

const USERS = [
  { username: 'بهاء', displayName: 'بهاء' },
  { username: 'يونس', displayName: 'يونس' },
  { username: 'ابو محمود', displayName: 'ابو محمود' },
  { username: 'محمود', displayName: 'محمود' },
];

let selected = null;

function initial(name) { return name.trim().charAt(0); }

function showUserSelect() {
  document.querySelector('#password-screen').classList.add('hidden');
  document.querySelector('#user-select').classList.remove('hidden');
  document.querySelector('#password-input').value = '';
  document.querySelector('#password-error').textContent = '';
}

function showPasswordScreen(user) {
  selected = user;
  document.querySelector('#user-select').classList.add('hidden');
  document.querySelector('#password-screen').classList.remove('hidden');
  document.querySelector('#password-avatar').textContent = initial(user.displayName);
  document.querySelector('#password-name').textContent = user.displayName;
  const input = document.querySelector('#password-input');
  input.value = '';
  input.focus();
}

async function submitPassword() {
  const password = document.querySelector('#password-input').value;
  const errorEl = document.querySelector('#password-error');
  errorEl.textContent = '';
  try {
    const user = await auth.login(selected.username, password);
    window.location.href = user.role === 'admin' ? '/admin/dashboard' : '/staff/home';
  } catch (err) {
    errorEl.textContent = 'كلمة المرور غير صحيحة';
  }
}

function init() {
  const grid = document.querySelector('#circle-grid');
  USERS.forEach(user => {
    const tile = document.createElement('button');
    tile.className = 'circle-tile';
    tile.innerHTML = `<span class="circle">${initial(user.displayName)}</span><span class="tile-label">${user.displayName}</span>`;
    tile.onclick = () => showPasswordScreen(user);
    grid.appendChild(tile);
  });

  document.querySelector('#back-btn').onclick = showUserSelect;
  document.querySelector('#submit-btn').onclick = submitPassword;
  document.querySelector('#password-input').addEventListener('keydown', e => {
    if (e.key === 'Enter') submitPassword();
  });
}

document.addEventListener('DOMContentLoaded', init);