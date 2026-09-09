import { api } from '../core/api.js';
import { auth } from '../core/auth.js';

const money = value => Number(value || 0).toFixed(2);
async function init() {
  if (!auth.requireAuth('staff')) return;
  const content = document.querySelector('#content');
  let shift = null;
  const render = () => {
    content.innerHTML = shift?.id ? `<section class="surface panel stack"><div class="cluster"><div><h2>الوردية الحالية</h2><p class="muted">الرصيد الافتتاحي: ${money(shift.opening_balance)}</p></div><strong>${shift.closed_at ? 'مغلقة' : 'مفتوحة'}</strong></div>${shift.closed_at ? `<p>الرصيد الفعلي: ${money(shift.actual_closing)} — الفرق: ${money(shift.difference)}</p>` : `<div class="cluster"><button class="btn btn-primary" id="movement">تسجيل حركة</button><button class="btn" id="close">إغلاق الوردية</button></div><div id="action"></div>`}</section>` : `<section class="surface panel"><h2>فتح وردية</h2><form id="open" class="stack"><label class="label">الرصيد الافتتاحي<input class="input" name="opening_balance" type="number" min="0" step="0.01" required></label><label class="label">ملاحظات<textarea class="input" name="opening_notes"></textarea></label><button class="btn btn-primary">فتح الوردية</button></form></section>`;
    content.querySelector('#open')?.addEventListener('submit', async event => { event.preventDefault(); const data = new FormData(event.target); shift = await api.post('/api/cash-shifts', { opening_balance: Number(data.get('opening_balance')), opening_notes: data.get('opening_notes') || undefined }); render(); });
    content.querySelector('#movement')?.addEventListener('click', () => { content.querySelector('#action').innerHTML = `<form class="surface panel stack" id="movement-form"><label class="label">الاتجاه<select class="select" name="direction"><option value="in">إيداع</option><option value="out">سحب</option></select></label><label class="label">المبلغ<input class="input" name="amount" type="number" min="0.01" step="0.01" required></label><label class="label">ملاحظات<textarea class="input" name="notes"></textarea></label><button class="btn btn-primary">حفظ الحركة</button></form>`; content.querySelector('#movement-form').onsubmit = async event => { event.preventDefault(); const data = new FormData(event.target); await api.post('/api/cash/movements', { direction: data.get('direction'), amount: Number(data.get('amount')), notes: data.get('notes') || undefined }); content.querySelector('#action').innerHTML = '<p class="success">تم تسجيل الحركة.</p>'; }; });
    content.querySelector('#close')?.addEventListener('click', () => { content.querySelector('#action').innerHTML = `<form class="surface panel stack" id="close-form"><label class="label">المبلغ المعدود<input class="input" name="actual_closing" type="number" min="0" step="0.01" required></label><label class="label">ملاحظات<textarea class="input" name="closing_notes"></textarea></label><button class="btn btn-primary">إغلاق الوردية</button></form>`; content.querySelector('#close-form').onsubmit = async event => { event.preventDefault(); const data = new FormData(event.target); shift = await api.post(`/api/cash-shifts/${shift.id}/close`, { actual_closing: Number(data.get('actual_closing')), closing_notes: data.get('closing_notes') || undefined }); render(); }; });
  };
  try { const current = await api.get('/api/cash/shifts/current'); shift = current?.id ? current : null; } catch {}
  render();
}
document.addEventListener('DOMContentLoaded', init);
