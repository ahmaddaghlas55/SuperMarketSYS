import { api } from '../core/api.js';
import { auth } from '../core/auth.js';
import { barcodeListener } from '../core/barcodeListener.js';
import { CartLine } from '../components/CartLine.js';
import { WeightPicker } from '../components/WeightPicker.js';
import { CustomerPicker } from '../components/CustomerPicker.js';

const money = value => Number(value || 0).toFixed(2);

// --- Simple audio feedback for barcode scans (no external asset needed) ---
// A short, pleasant beep on a successful scan match; a lower double-buzz on a failed/no-match scan.
let audioCtx = null;
function beep({ frequency = 880, duration = 0.09, type = 'sine' } = {}) {
  try {
    audioCtx = audioCtx || new (window.AudioContext || window.webkitAudioContext)();
    const oscillator = audioCtx.createOscillator();
    const gain = audioCtx.createGain();
    oscillator.type = type;
    oscillator.frequency.value = frequency;
    gain.gain.setValueAtTime(0.15, audioCtx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, audioCtx.currentTime + duration);
    oscillator.connect(gain).connect(audioCtx.destination);
    oscillator.start();
    oscillator.stop(audioCtx.currentTime + duration);
  } catch (error) { /* audio not available — silently skip, never block the sale */ }
}
const playScanSuccess = () => beep({ frequency: 1046, duration: 0.08 });
const playScanNotFound = () => { beep({ frequency: 220, duration: 0.12 }); setTimeout(() => beep({ frequency: 220, duration: 0.12 }), 140); };

async function init() {
  if (!auth.requireAuth('staff')) return;
  const app = document.querySelector('#app');
  const state = { products: [], customers: [], cart: [], selected: null, customer: null, requestId: crypto.randomUUID() };
  const search = app.querySelector('#search');
  const matches = app.querySelector('#matches');
  const detail = app.querySelector('#detail');
  const quickSell = app.querySelector('#quick-sell');
  const cart = app.querySelector('#cart');
  const summary = app.querySelector('#summary');

  const resetCheckout = () => {
    state.cart = [];
    state.customer = null;
    state.requestId = crypto.randomUUID(); // fresh id for the next, separate sale
    refreshCart();
  };

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

    summary.innerHTML = `<div class="stack">
      <div class="cluster"><span>الإجمالي</span><strong>${money(subtotal)}</strong></div>
      <label class="label">طريقة الدفع
        <select class="select" id="method">
          <option value="cash">نقداً</option>
          <option value="card">بطاقة</option>
          <option value="debt">دين</option>
        </select>
      </label>
      <div id="cash-fields">
        <label class="label">المبلغ المستلم<input class="input" id="received" type="number" min="0" step="0.01"></label>
        <p id="change" class="muted">الباقي: 0.00</p>
      </div>
      <div id="customer-handoff"></div>
      <button class="btn btn-primary" id="submit">إتمام البيع</button>
    </div>`;

    const methodEl = summary.querySelector('#method');
    const cashFields = summary.querySelector('#cash-fields');
    const receivedEl = summary.querySelector('#received');
    const submitBtn = summary.querySelector('#submit');

    const updateCashFieldsVisibility = () => {
      const method = methodEl.value;
      cashFields.style.display = method === 'card' ? 'none' : '';
    };

    const checkShortfall = () => {
      const method = methodEl.value;
      const received = method === 'card' ? subtotal : Number(receivedEl.value || 0);
      if (method === 'debt' || received < subtotal) showCustomerPicker();
    };

    updateCashFieldsVisibility();

    receivedEl.oninput = () => {
      const amount = Number(receivedEl.value || 0);
      summary.querySelector('#change').textContent = `الباقي: ${money(Math.max(0, amount - subtotal))}`;
      checkShortfall();
    };

    methodEl.onchange = () => {
      updateCashFieldsVisibility();
      checkShortfall();
    };

    submitBtn.onclick = async () => {
      const method = methodEl.value;
      const received = method === 'card' ? subtotal : Number(receivedEl.value || 0);

      const needsCustomer = method === 'debt' || received < subtotal;
      if (needsCustomer && !state.customer) { await showCustomerPicker(); return; }

      submitBtn.disabled = true;
      submitBtn.textContent = 'جارٍ الإرسال...';
      try {
        await api.post('/api/sales', {
          request_id: state.requestId,
          customer_id: state.customer?.id,
          payment_method: method === 'debt' ? 'cash' : method,
          amount_received: received,
          items: state.cart.map(({ product_id, quantity, price_override }) => ({ product_id, quantity, price_override }))
        });
        resetCheckout();
      } catch (error) {
        submitBtn.disabled = false;
        submitBtn.textContent = 'إتمام البيع';
        summary.insertAdjacentHTML('beforeend', `<p class="error">${error.message}</p>`);
      }
    };
  };

  const selectProduct = product => {
    detail.innerHTML = `<h3>${product.name}</h3><p class="muted">${product.unit_type}</p><label class="label">الكمية<input id="quantity" class="input" type="number" min="0.01" step="0.01" value="1"></label><div id="weights"></div><label class="label">سعر مخصص للوحدة<input id="override" class="input" type="number" min="0" step="0.01"></label><button class="btn btn-primary" id="add">إضافة إلى السلة</button>`;
    if (product.unit_type === 'weight') new WeightPicker(detail.querySelector('#weights'), { onChange: value => detail.querySelector('#quantity').value = value });
    detail.querySelector('#add').onclick = () => { const quantity = Number(detail.querySelector('#quantity').value); const override = Number(detail.querySelector('#override').value); const defaultPrice = product.sale_price ?? product.price_per_kg ?? product.price_per_piece ?? 0; state.cart.push({ product_id: product.id, name: product.name, quantity, unit_price: override || defaultPrice, price_override: override || undefined }); refreshCart(); };
  };

  // --- Quick-sell buttons ---
  // Heuristic for now (no backend "quick_sell" flag exists yet): weight-type products
  // (produce, coffee, nuts — the original motivating case for this feature) get a button.
  // TODO: replace with a real `quick_sell` boolean on the product once the backend adds it,
  // so piece-type favorites (e.g. a specific cigarette brand) can be included too.
  const renderQuickSell = () => {
    const favorites = state.products.filter(product => product.unit_type === 'weight').slice(0, 8);
    if (!favorites.length) { quickSell.innerHTML = ''; return; }
    quickSell.innerHTML = `<h3 class="quick-sell-title">أصناف سريعة</h3><div class="quick-sell-grid"></div>`;
    const grid = quickSell.querySelector('.quick-sell-grid');
    favorites.forEach(product => {
      const button = document.createElement('button');
      button.className = 'btn quick-sell-btn';
      button.textContent = product.name;
      button.onclick = () => selectProduct(product);
      grid.append(button);
    });
  };

  const renderMatches = list => { matches.innerHTML = list.slice(0, 12).map(product => `<button class="list-item" data-id="${product.id}"><strong>${product.name}</strong><span class="muted">${product.barcode || 'بدون باركود'} · ${product.quantity}</span></button>`).join(''); matches.querySelectorAll('[data-id]').forEach(button => button.onclick = () => selectProduct(state.products.find(product => product.id === Number(button.dataset.id)))); };
  search.oninput = () => renderMatches(state.products.filter(product => `${product.name} ${product.barcode || ''}`.toLowerCase().includes(search.value.toLowerCase())));
  app.querySelector('#clear').onclick = resetCheckout;

  app.addEventListener('scan', event => {
    search.value = event.detail;
    const found = state.products.filter(product => product.barcode === event.detail);
    renderMatches(found);
    if (found.length) {
      playScanSuccess();
      selectProduct(found[0]); // jump straight to the detail/add step — no extra tap needed for a real scan
    } else {
      playScanNotFound();
    }
  });
  barcodeListener.startListening(app);

  try {
    state.products = (await api.get('/api/products')).products || [];
    renderMatches(state.products);
    renderQuickSell();
  } catch (error) { matches.innerHTML = `<p class="error">${error.message}</p>`; }
  refreshCart();
}
document.addEventListener('DOMContentLoaded', init);