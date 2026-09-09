import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
import { DataTable } from '../components/DataTable.js';
async function init() { if (!auth.requireAuth('admin')) return; const app = document.querySelector('#app'); const report = app.querySelector('#report'); const load = async type => { const data = await api.get(`/api/reports/${type}`); new DataTable(report, { columns: [{ key: 'key', label: 'المؤشر' }, { key: 'value', label: 'القيمة' }], rows: Object.entries(data).filter(([, value]) => !Array.isArray(value)).map(([key, value]) => ({ key, value })) }); }; app.querySelectorAll('[data-report]').forEach(button => button.onclick = () => load(button.dataset.report)); load('sales'); }
document.addEventListener('DOMContentLoaded', init);
