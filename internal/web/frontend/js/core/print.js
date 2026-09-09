function printBarcodeLabel(product) {
  const popup = window.open('', '_blank', 'width=500,height=300');
  if (!popup) return;
  popup.document.write(`<main dir="rtl"><strong>${product.name || ''}</strong><div>${product.barcode || ''}</div></main>`);
  popup.document.close();
  popup.focus();
  popup.print();
}

function printReceipt(content) {
  const popup = window.open('', '_blank', 'width=500,height=700');
  if (!popup) return;
  popup.document.write(`<main dir="rtl">${content || ''}</main>`);
  popup.document.close();
  popup.focus();
  popup.print();
}

export const print = { printBarcodeLabel, printReceipt };
