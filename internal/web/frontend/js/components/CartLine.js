export function CartLine(container, props = {}) {
  const render = () => {
    const item = props.item || {};
    container.innerHTML = `<div><strong>${item.name || ''}</strong><div class="muted">${item.quantity || 0} × ${item.unit_price ?? ''}</div></div><strong>${item.line_total ?? ''}</strong>`;
    container.className = 'list-item cluster';
  };
  render();
  return { el: container, update(next = {}) { props = next; render(); }, destroy() { container.innerHTML = ''; } };
}
