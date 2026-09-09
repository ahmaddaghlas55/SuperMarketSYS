export function NumPad(container, props = {}) {
  let value = String(props.value || '');
  const render = () => {
    container.innerHTML = '';
    container.className = 'nav-grid';
    for (const key of ['1','2','3','4','5','6','7','8','9','0','.', '⌫']) {
      const button = document.createElement('button');
      button.className = 'btn';
      button.textContent = key;
      button.onclick = () => {
        value = key === '⌫' ? value.slice(0, -1) : value + key;
        props.onChange?.(value);
      };
      container.append(button);
    }
  };
  render();
  return { el: container, update(next = {}) { props = next; value = String(next.value ?? value); render(); }, destroy() { container.innerHTML = ''; } };
}
