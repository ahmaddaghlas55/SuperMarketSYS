let listener = null;
let buffer = '';
let lastKeyAt = 0;

function startListening(target = document) {
  stopListening(target);
  listener = event => {
    const now = Date.now();
    if (now - lastKeyAt > 80) buffer = '';
    lastKeyAt = now;
    if (event.key === 'Enter') {
      if (buffer) target.dispatchEvent(new CustomEvent('scan', { detail: buffer }));
      buffer = '';
      return;
    }
    if (event.key.length === 1) buffer += event.key;
  };
  target.addEventListener('keydown', listener);
}

function stopListening(target = document) {
  if (listener) target.removeEventListener('keydown', listener);
  listener = null;
  buffer = '';
}

export const barcodeListener = { startListening, stopListening };
