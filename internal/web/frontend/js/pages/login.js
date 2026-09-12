import { auth } from '../core/auth.js';

const USERS = [
  { username: 'بهاء', displayName: 'بهاء' },
  { username: 'يونس', displayName: 'يونس' },
  { username: 'ابو محمود', displayName: 'ابو محمود' },
  { username: 'محمود', displayName: 'محمود' },
];

let selected = null;
let pin = '';

function initial(name) { return name.trim().charAt(0); }

function showUserSelect() {
  document.querySelector('#password-screen').classList.add('hidden');
  document.querySelector('#user-select').classList.remove('hidden');
  pin = '';
  document.querySelector('#password-error').textContent = '';
}

function renderPinDots() {
  const dots = document.querySelector('#pin-dots');
  dots.innerHTML = '';
  for (let i = 0; i < pin.length; i++) {
    const dot = document.createElement('span');
    dot.className = 'pin-dot filled';
    dots.appendChild(dot);
  }
}

function showPasswordScreen(user) {
  selected = user;
  pin = '';
  document.querySelector('#user-select').classList.add('hidden');
  document.querySelector('#password-screen').classList.remove('hidden');
  document.querySelector('#password-avatar').textContent = initial(user.displayName);
  document.querySelector('#password-name').textContent = user.displayName;
  document.querySelector('#password-error').textContent = '';
  renderPinDots();
}

async function submitPassword() {
  const errorEl = document.querySelector('#password-error');
  errorEl.textContent = '';
  try {
    const user = await auth.login(selected.username, pin);
    window.location.href = user.role === 'admin' ? '/admin/dashboard' : '/staff/home';
  } catch (err) {
    errorEl.textContent = 'كلمة المرور غير صحيحة';
    pin = '';
    renderPinDots();
  }
}

function handlePinKey(key) {
  if (key === 'back') {
    pin = pin.slice(0, -1);
  } else if (key === 'submit') {
    submitPassword();
    return;
  } else {
    pin += key;
  }
  renderPinDots();
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
  document.querySelector('#pin-pad').addEventListener('click', e => {
    const btn = e.target.closest('.pin-key');
    if (btn) handlePinKey(btn.dataset.key);
  });
}

document.addEventListener('DOMContentLoaded', init);