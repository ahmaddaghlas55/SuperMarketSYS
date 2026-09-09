import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
import { print } from '../core/print.js';
async function init() {
  if (!auth.requireAuth('staff')) return;
  const app = document.querySelector('#app'); const data = await api.get('/api/products'); const list = app.querySelector('#list');
  const render = () => { const q = app.querySelector('#search').value.toLowerCase(); list.innerHTML = (data.products || []).filter(product => product.name.toLowerCase().includes(q)).map(product => `<button class="list-item" data-id="${product.id}">${product.name} — ${product.barcode || 'لا يوجد'}</button>`).join(''); list.querySelectorAll('[data-id]').forEach(button => button.onclick = () => { const product = data.products.find(item => item.id === Number(button.dataset.id)); app.querySelector('#detail').innerHTML = `<article class="surface panel stack"><h2>${product.name}</h2><div>${product.barcode || 'لا يوجد باركود'}</div><button class="btn btn-primary">طباعة الملصق</button></article>`; app.querySelector('#detail button').onclick = () => print.printBarcodeLabel(product); }); };
  app.querySelector('#search').oninput = render; render();
}
document.addEventListener('DOMContentLoaded', init);
