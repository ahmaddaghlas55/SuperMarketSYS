import { api } from '../core/api.js';
import { auth } from '../core/auth.js';

async function init() {
  const user = auth.requireAuth('staff');
  if (!user) return;

  if (user.role === 'admin') {
    window.location.replace('/admin/dashboard');
    return;
  }

  document.querySelector('#welcome').textContent = `مرحباً ${user.username}`;
  document.querySelector('#logout').onclick = () => auth.logout();

  try {
    const current = await api.get('/api/cash/shifts/current');
    if (!current || !current.id) {
      document.querySelector('#cash-badge').classList.remove('hidden');
    }
  } catch {
    document.querySelector('#cash-badge').classList.remove('hidden');
  }
}
document.addEventListener('DOMContentLoaded', init);
