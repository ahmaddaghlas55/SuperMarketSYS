import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
import { barcodeListener } from '../core/barcodeListener.js';
import { CartLine } from '../components/CartLine.js';
import { WeightPicker } from '../components/WeightPicker.js';
import { CustomerPicker } from '../components/CustomerPicker.js';

const money = value => Number(value || 0).toFixed(2);

async function init() {
  if (!auth.requireAuth('staff')) return;
  const app = document.querySelector('#app');
  const state = { products: [], customers: [], cart: [], selected: null, customer: null };
  const search = app.querySelector('#search');
  const matches = app.querySelector('#matches');
  const detail = app.querySelector('#detail');
  const cart = app.querySelector('#cart');
  const summary = app.querySelector('#summary');
  const showCustomerPicker = async () => {
    const host = summary.querySelector('#customer-handoff');
    if (!host || host.dataset.open === 'true') return;
    host.dataset.open = 'true';
    if (!state.customers.length) state.customers = (await api.get('/api/customers')).customers || [];
    host.innerHTML = '<h3>اختر عميلاً لتسجيل المبلغ المتبقي كدين</h3><div id="picker"></div><p id="customer-selected" class="success"></p>';
    new CustomerPicker(host.querySelector('#picker'), {
      customers: state.customers,
      onChange: customer => { state.customer = customer; host.querySelector('#customer-selected').textContent = `العميل المختار: ${customer.name}`; },
      onCreate: async name => { const data = await api.post('/api/customers', { name }); const customer = data.customer || data; state.customers.push(customer); return customer; }
    });
  };
  const refreshCart = () => {
    cart.innerHTML = '';
    state.cart.forEach((item, index) => {
      const row = document.createElement('div');
      new CartLine(row, { item: { ...item, line_total: money(item.quantity * item.unit_price) } });
      row.insertAdjacentHTML('beforeend', '<button class="btn btn-ghost">حذف</button>');
      row.querySelector('button').onclick = () => { state.cart.splice(index, 1); refreshCart(); };
      cart.append(row);
    });
    const subtotal = state.cart.reduce((sum, item) => sum + item.quantity * item.unit_price, 0);
    summary.innerHTML = `<div class="stack"><div class="cluster"><span>الإجمالي</span><strong>${money(subtotal)}</strong></div><label class="label">طريقة الدفع<select class="select" id="method"><option value="cash">نقداً</option><option value="card">بطاقة</option><option value="debt">دين</option></select></label><label class="label">المبلغ المستلم<input class="input" id="received" type="number" min="0" step="0.01"></label><p id="change" class="muted">الباقي: 0.00</p><div id="customer-handoff"></div><button class="btn btn-primary" id="submit">إتمام البيع</button></div>`;
    summary.querySelector('#received').oninput = event => { const amount = Number(event.target.value || 0); summary.querySelector('#change').textContent = `الباقي: ${money(Math.max(0, amount - subtotal))}`; if (amount < subtotal) showCustomerPicker(); };
    summary.querySelector('#method').onchange = () => { if (summary.querySelector('#method').value === 'debt') showCustomerPicker(); };
    summary.querySelector('#submit').onclick = async () => {
      const method = summary.querySelector('#method').value;
      const received = Number(summary.querySelector('#received').value || 0);
      if ((method === 'debt' || (method === 'cash' && received < subtotal)) && !state.customer) { await showCustomerPicker(); return; }
      await api.post('/api/sales', { request_id: crypto.randomUUID(), customer_id: state.customer?.id, payment_method: method === 'debt' ? 'cash' : method, amount_received: received || undefined, amount_paid: method === 'debt' ? 0 : undefined, items: state.cart.map(({ product_id, quantity, price_override }) => ({ product_id, quantity, price_override })) });
      state.cart = []; state.customer = null; refreshCart();
    };
  };
  const selectProduct = product => {
    detail.innerHTML = `<h3>${product.name}</h3><p class="muted">${product.unit_type}</p><label class="label">الكمية<input id="quantity" class="input" type="number" min="0.01" step="0.01" value="1"></label><div id="weights"></div><label class="label">سعر مخصص للوحدة<input id="override" class="input" type="number" min="0" step="0.01"></label><button class="btn btn-primary" id="add">إضافة إلى السلة</button>`;
    if (product.unit_type === 'weight') new WeightPicker(detail.querySelector('#weights'), { onChange: value => detail.querySelector('#quantity').value = value });
    detail.querySelector('#add').onclick = () => { const quantity = Number(detail.querySelector('#quantity').value); const override = Number(detail.querySelector('#override').value); const defaultPrice = product.sale_price ?? product.price_per_kg ?? product.price_per_piece ?? 0; state.cart.push({ product_id: product.id, name: product.name, quantity, unit_price: override || defaultPrice, price_override: override || undefined }); refreshCart(); };
  };
  const renderMatches = list => { matches.innerHTML = list.slice(0, 12).map(product => `<button class="list-item" data-id="${product.id}"><strong>${product.name}</strong><span class="muted">${product.barcode || 'بدون باركود'} · ${product.quantity}</span></button>`).join(''); matches.querySelectorAll('[data-id]').forEach(button => button.onclick = () => selectProduct(state.products.find(product => product.id === Number(button.dataset.id)))); };
  search.oninput = () => renderMatches(state.products.filter(product => `${product.name} ${product.barcode || ''}`.toLowerCase().includes(search.value.toLowerCase())));
  app.querySelector('#clear').onclick = () => { state.cart = []; state.customer = null; refreshCart(); };
  app.addEventListener('scan', event => { search.value = event.detail; renderMatches(state.products.filter(product => product.barcode === event.detail)); });
  barcodeListener.startListening(app);
  try { state.products = (await api.get('/api/products')).products || []; renderMatches(state.products); } catch (error) { matches.innerHTML = `<p class="error">${error.message}</p>`; }
  refreshCart();
}
document.addEventListener('DOMContentLoaded', init);
