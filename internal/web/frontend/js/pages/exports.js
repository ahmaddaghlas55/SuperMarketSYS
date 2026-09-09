import { auth } from '../core/auth.js';

function init() {
  auth.requireAuth('admin');
}

document.addEventListener('DOMContentLoaded', init);
