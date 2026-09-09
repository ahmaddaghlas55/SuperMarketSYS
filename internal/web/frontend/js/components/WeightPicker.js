export function WeightPicker(container, props = {}) {
  const render = () => {
    container.innerHTML = '';
    container.className = 'cluster';
    for (const weight of props.weights || [0.25, 0.5, 1, 2]) {
      const button = document.createElement('button');
      button.className = 'btn';
      button.textContent = `${weight} kg`;
      button.onclick = () => props.onChange?.(weight);
      container.append(button);
    }
  };
  render();
  return { el: container, update(next = {}) { props = next; render(); }, destroy() { container.innerHTML = ''; } };
}
