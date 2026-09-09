export function StatusBadge(container, props = {}) {
  const render = () => {
    container.textContent = props.label || '';
    container.className = `status-badge ${props.tone || ''}`;
  };
  render();
  return { el: container, update(next = {}) { props = next; render(); }, destroy() { container.textContent = ''; } };
}
