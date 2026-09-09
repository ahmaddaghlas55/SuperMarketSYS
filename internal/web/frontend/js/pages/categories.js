import { auth } from '../core/auth.js';
function init() { if (auth.requireAuth('admin')) window.location.replace('/admin/products'); }
document.addEventListener('DOMContentLoaded', init);
