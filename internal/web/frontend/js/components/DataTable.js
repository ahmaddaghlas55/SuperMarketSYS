export function DataTable(container, props = {}) {
  const render = () => {
    const columns = props.columns || [];
    container.innerHTML = `<div class="table-wrap"><table><thead><tr>${columns.map(column => `<th>${column.label}</th>`).join('')}</tr></thead><tbody>${(props.rows || []).map(row => `<tr>${columns.map(column => `<td>${column.render ? column.render(row) : row[column.key] ?? ''}</td>`).join('')}</tr>`).join('')}</tbody></table></div>`;
  };
  render();
  return { el: container, update(next = {}) { props = next; render(); }, destroy() { container.innerHTML = ''; } };
}
