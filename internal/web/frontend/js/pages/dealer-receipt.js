import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
async function init() {
  if (!auth.requireAuth('staff')) return;
  const app = document.querySelector('#app'); const form = app.querySelector('#form');
  const [dealers, products] = await Promise.all([api.get('/api/dealers'), api.get('/api/products')]);
  form.dealer_id.innerHTML = (dealers.dealers || []).map(dealer => `<option value="${dealer.id}">${dealer.name}</option>`).join('');
  const addItem = () => { const row = document.createElement('div'); row.className = 'cluster'; row.innerHTML = `<select class="select product"><option value="">المنتج</option>${(products.products || []).map(product => `<option value="${product.id}">${product.name}</option>`).join('')}</select><input class="input quantity" type="number" min="0.01" step="0.01" placeholder="الكمية"><input class="input cost" type="number" min="0" step="0.01" placeholder="كلفة الوحدة"><button type="button" class="btn btn-danger">حذف</button>`; row.querySelector('button').onclick = () => row.remove(); app.querySelector('#items').append(row); };
  addItem(); app.querySelector('#add').onclick = addItem;
  form.onsubmit = async event => { event.preventDefault(); const rows = [...app.querySelectorAll('#items > div')]; const data = new FormData(form); await api.post('/api/purchases', { dealer_id: Number(data.get('dealer_id')), dealer_invoice_number: data.get('dealer_invoice_number') || undefined, discount: Number(data.get('discount') || 0), items: rows.map(row => ({ product_id: Number(row.querySelector('.product').value), quantity: Number(row.querySelector('.quantity').value), unit_cost: Number(row.querySelector('.cost').value) })) }); app.querySelector('#message').textContent = 'تم حفظ الإيصال.'; form.reset(); };
}
document.addEventListener('DOMContentLoaded', init);
