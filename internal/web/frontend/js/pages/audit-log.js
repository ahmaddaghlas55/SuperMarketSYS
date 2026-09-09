import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
import { DataTable } from '../components/DataTable.js';
async function init() { if (!auth.requireAuth('admin')) return; const data = await api.get('/api/audit-log'); new DataTable(document.querySelector('#table'), { columns: [{ key: 'created_at', label: 'التاريخ' }, { key: 'module', label: 'الوحدة' }, { key: 'action', label: 'الإجراء' }, { key: 'user_id', label: 'المستخدم' }, { key: 'description', label: 'الوصف' }], rows: data.audit || [] }); }
document.addEventListener('DOMContentLoaded', init);
