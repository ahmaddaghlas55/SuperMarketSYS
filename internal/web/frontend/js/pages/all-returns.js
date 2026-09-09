import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
import { TimelineLedger } from '../components/TimelineLedger.js';
async function init() { if (!auth.requireAuth('admin')) return; const data = await api.get('/api/purchase-returns'); TimelineLedger(document.querySelector('#timeline'), { entries: (data.returns || []).map(item => ({ created_at: item.created_at, title: `مرتجع شراء ${item.id}`, description: `${item.reason} · ${item.total_value || ''}` })) }); }
document.addEventListener('DOMContentLoaded', init);
