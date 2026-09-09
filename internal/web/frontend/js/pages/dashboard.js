import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
async function init() {
  if (!auth.requireAuth('admin')) return;
  const app = document.querySelector('#app'); app.querySelector('#logout').onclick = () => auth.logout();
  try { const data = await api.get('/api/reports/dashboard'); const labels = [['today_sales_total','مبيعات اليوم'],['today_profit','ربح اليوم'],['low_stock_count','منخفض المخزون'],['today_expenses','مصروفات اليوم'],['today_returns_count','مرتجعات اليوم']]; app.querySelector('#metrics').innerHTML = labels.map(([key, label]) => `<article class="surface metric"><span class="muted">${label}</span><strong>${data[key] ?? 0}</strong></article>`).join('') + `<article class="surface metric"><span class="muted">الوردية</span><strong>${data.open_shift ? 'مفتوحة' : 'مغلقة'}</strong></article>`; } catch (error) { app.querySelector('#metrics').innerHTML = `<p class="error">تعذر تحميل لوحة اليوم: ${error.message}</p>`; }
}
document.addEventListener('DOMContentLoaded', init);
