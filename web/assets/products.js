// web/assets/products.js
async function api(p) {
  const r = await fetch(p, { credentials: "include" });
  const txt = await r.text(); let d; try { d = JSON.parse(txt); } catch { d = txt; }
  if (!r.ok) throw new Error(d?.error || ("HTTP " + r.status));
  return d;
}

const msg = (s, ok = true) => {
  const el = document.getElementById("msg");
  if (el) { el.textContent = s; el.style.color = ok ? "#16a34a" : "#ef4444"; }
};

async function loadProducts() {
  try {
    const data = await api("/api/products");
    const list = document.getElementById("list");
    list.innerHTML = data.map(p => `
      <div class="card">
        <img src="${p.ImageURL}" alt="">
        <div class="b">
          <div class="t">${p.Name}</div>
          <div class="price">${(p.PriceCents/100).toFixed(2)} ₺</div>
          <button onclick="addToCart(${p.ID})">Sepete Ekle</button>
        </div>
      </div>
    `).join("");
  } catch (e) {
    msg("Ürünler yüklenemedi: " + e.message, false);
  }
}

async function addToCart(id) {
  try {
    await fetch("/api/cart/add", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ product_id: id, qty: 1 })
    });
    msg("Sepete eklendi");
  } catch (e) {
    msg("Giriş yapmanız gerekebilir: " + e.message, false);
  }
}

document.addEventListener("DOMContentLoaded", loadProducts);
loadProducts();