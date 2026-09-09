export function TimelineLedger(container, props = {}) {
  const render = entries => {
    container.innerHTML = '';
    const groups = new Map();
    for (const entry of entries || []) {
      const date = String(entry.date || entry.created_at || '').slice(0, 10) || '—';
      if (!groups.has(date)) groups.set(date, []);
      groups.get(date).push(entry);
    }
    container.className = 'timeline';
    for (const [date, dayEntries] of groups) {
      const section = document.createElement('section');
      section.innerHTML = `<div class="timeline-date">${date}</div>`;
      for (const entry of dayEntries) {
        const row = document.createElement('article');
        row.className = `timeline-entry ${entry.paid ? 'paid' : ''} ${entry.aging ? 'aging' : ''}`;
        row.innerHTML = entry.html || `<strong>${entry.title || ''}</strong><div>${entry.description || ''}</div>`;
        section.append(row);
      }
      container.append(section);
    }
  };
  render(props.entries);
  return { el: container, update(next = {}) { render(next.entries); }, destroy() { container.innerHTML = ''; } };
}
