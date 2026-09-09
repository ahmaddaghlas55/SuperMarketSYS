import { auth } from '../core/auth.js';
function init() { if (auth.requireAuth('admin')) window.location.replace('/admin/dealer-ledger'); }
document.addEventListener('DOMContentLoaded', init);
