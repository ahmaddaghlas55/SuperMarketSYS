import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
import { TimelineLedger } from '../components/TimelineLedger.js';
async function init() {
  if (!auth.requireAuth('staff')) return;
  const app = document.querySelector('#app'); const list = app.querySelector('#list'); const data = await api.get('/api/dealers');
  const load = async id => { const [dealer, purchases] = await Promise.all([api.get(`/api/dealers/${id}`), api.get(`/api/purchases?dealer_id=${id}`)]); const detail = app.querySelector('#detail'); detail.innerHTML = `<h2>${dealer.name}</h2><p>المستحق: ${dealer.outstanding || 0} · الرصيد الدائن: ${dealer.credit || 0}</p><div id="timeline"></div><form id="payment" class="stack"><h3>تسجيل دفعة</h3><input class="input" name="amount" type="number" min="0.01" step="0.01" placeholder="المبلغ" required><select class="select" name="method"><option value="cash">نقداً</option><option value="card">بطاقة</option><option value="transfer">تحويل</option></select><button class="btn btn-primary">حفظ</button></form>`; TimelineLedger(detail.querySelector('#timeline'), { entries: (purchases.purchases || []).map(p => ({ created_at: p.created_at, title: `شراء ${p.id}`, description: `المتبقي: ${p.remaining || 0}` })) }); detail.querySelector('#payment').onsubmit = async event => { event.preventDefault(); const form = new FormData(event.target); const first = (purchases.purchases || [])[0]; if (first) await api.post(`/api/purchases/${first.id}/payments`, { amount: Number(form.get('amount')), method: form.get('method') }); }; };
  const render = () => { const q = app.querySelector('#search').value.toLowerCase(); list.innerHTML = (data.dealers || []).filter(d => d.name.toLowerCase().includes(q)).map(d => `<button class="list-item" data-id="${d.id}">${d.name}</button>`).join(''); list.querySelectorAll('[data-id]').forEach(b => b.onclick = () => load(Number(b.dataset.id))); };
  app.querySelector('#search').oninput = render; render();
}
document.addEventListener('DOMContentLoaded', init);
