import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
async function init() {
  if (!auth.requireAuth('staff')) return;
  const app = document.querySelector('#app'); const customersEl = app.querySelector('#customers'); let customers = [];
  const render = () => { const query = app.querySelector('#search').value.toLowerCase(); customersEl.innerHTML = customers.filter(customer => customer.name.toLowerCase().includes(query)).map(customer => `<a class="list-item" href="/staff/debt-detail?id=${customer.id}"><strong>${customer.name}</strong><span class="muted">${customer.phone || ''}</span></a>`).join('') || '<p class="empty">لا يوجد عملاء</p>'; };
  try { customers = (await api.get('/api/customers')).customers || []; render(); app.querySelector('#search').oninput = render; } catch (error) { customersEl.innerHTML = `<p class="error">${error.message}</p>`; }
  app.querySelector('#add').onclick = async () => { const name = window.prompt('اسم العميل'); if (!name) return; await api.post('/api/customers', { name }); window.location.reload(); };
}
document.addEventListener('DOMContentLoaded', init);
