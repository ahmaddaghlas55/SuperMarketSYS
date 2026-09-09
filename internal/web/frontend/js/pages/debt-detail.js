import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
import { TimelineLedger } from '../components/TimelineLedger.js';
async function init() {
  if (!auth.requireAuth('staff')) return;
  const id = new URLSearchParams(location.search).get('id'); const app = document.querySelector('#app');
  try {
    const data = await api.get(`/api/customers/${id}/debt`);
    app.querySelector('#customer').innerHTML = `<h2>${data.customer.name}</h2><p class="muted">${data.customer.phone || ''}</p>`;
    const sales = [...(data.sales || [])].sort((a, b) => new Date(a.created_at) - new Date(b.created_at));
    TimelineLedger(app.querySelector('#ledger'), { entries: sales.map(sale => ({ created_at: sale.created_at, paid: Number(sale.remaining || 0) <= 0, aging: Number(sale.remaining || 0) > 0 && Date.now() - new Date(sale.created_at).getTime() > 30 * 86400000, title: `فاتورة ${sale.invoice_number || sale.id}`, description: `المتبقي: ${sale.remaining || 0}` })) });
    app.querySelector('#payment').innerHTML = `<div class="stack"><h3>تسجيل دفعة</h3><form id="general-payment" class="stack"><label class="label">المبلغ العام (الأقدم أولاً)<input class="input" name="amount" type="number" min="0.01" step="0.01" required></label><label class="label">طريقة الدفع<select class="select" name="method"><option value="cash">نقداً</option><option value="card">بطاقة</option></select></label><button class="btn btn-primary">تطبيق على الأقدم أولاً</button></form><button class="btn" id="pay-total">دفع كامل الرصيد (${data.customer.outstanding || 0})</button><div id="date-payments">${sales.filter(sale => Number(sale.remaining || 0) > 0).map(sale => `<form class="cluster date-payment" data-sale-id="${sale.id}"><span>فاتورة ${sale.invoice_number || sale.id}: ${sale.remaining}</span><input class="input" name="amount" type="number" min="0.01" max="${sale.remaining}" step="0.01" placeholder="المبلغ" required><select class="select" name="method"><option value="cash">نقداً</option><option value="card">بطاقة</option></select><button class="btn">دفع هذا التاريخ</button></form>`).join('')}</div><div id="message"></div></div>`;
    const pay = async (saleId, amount, method) => { try { await api.post(`/api/customers/${id}/payments`, { sale_id: saleId, amount, method }); window.location.reload(); } catch (error) { app.querySelector('#message').className = 'error'; app.querySelector('#message').textContent = `تعذر تسجيل الدفعة: ${error.message}`; } };
    app.querySelector('#general-payment').onsubmit = async event => { event.preventDefault(); const form = new FormData(event.target); await pay(0, Number(form.get('amount')), form.get('method')); };
    app.querySelector('#pay-total').onclick = async () => pay(0, Number(data.customer.outstanding), 'cash');
    app.querySelectorAll('.date-payment').forEach(form => form.onsubmit = async event => { event.preventDefault(); const values = new FormData(form); await pay(Number(form.dataset.saleId), Number(values.get('amount')), values.get('method')); });
  } catch (error) { app.querySelector('#customer').innerHTML = `<p class="error">${error.message}</p>`; }
}
document.addEventListener('DOMContentLoaded', init);
