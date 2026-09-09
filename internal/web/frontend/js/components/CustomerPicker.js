export function CustomerPicker(container, props = {}) {
  const render = () => {
    container.innerHTML = `<div class="stack"><label class="label">بحث عن العميل<input class="input" id="customer-search" type="search"></label><div id="customer-options" class="list"></div><form id="new-customer" class="cluster"><input class="input" name="name" placeholder="اسم عميل جديد" required><button class="btn" type="submit">إضافة عميل</button></form></div>`;
    const search = container.querySelector('#customer-search');
    const options = container.querySelector('#customer-options');
    const draw = () => {
      const query = search.value.trim().toLowerCase();
      options.innerHTML = (props.customers || []).filter(customer => customer.name.toLowerCase().includes(query)).map(customer => `<button type="button" class="list-item" data-id="${customer.id}">${customer.name}${customer.phone ? ` · ${customer.phone}` : ''}</button>`).join('') || '<p class="empty">لا يوجد عميل مطابق</p>';
      options.querySelectorAll('[data-id]').forEach(button => button.onclick = () => props.onChange?.((props.customers || []).find(customer => customer.id === Number(button.dataset.id))));
    };
    search.oninput = draw;
    container.querySelector('#new-customer').onsubmit = async event => {
      event.preventDefault();
      const name = new FormData(event.target).get('name');
      const customer = await props.onCreate?.(name);
      if (customer) props.onChange?.(customer);
    };
    draw();
  };
  render();
  return { el: container, update(next = {}) { props = next; render(); }, destroy() { container.innerHTML = ''; } };
}
