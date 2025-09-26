// /web/assets/products.js

async function api(p) {
  const r = await fetch(p, { credentials: "include" });
  const text = await r.text();
  let data;
  try { data = JSON.parse(text); } catch { data = text; }
  if (!r.ok) throw new Error((data && data.error) || ("HTTP " + r.status));
  return data;
}

const msg = (s, ok = true) => {
  const el = document.getElementById("msg");
  if (el) { el.textContent = s; el.style.color = ok ? "#16a34a" : "#ef4444"; }
};

function pick(v, ...alts) {
  for (const k of [v, ...alts]) if (k !== undefined && k !== null) return k;
  return undefined;
}

async function loadProducts() {
  try {
    const data = await api("/api/products");
    const list = document.getElementById("list");
    if (!Array.isArray(data)) throw new Error("Beklenmeyen yanıt");

    const hasExt = (s) => /\.(png|jpe?g|webp|gif|svg)$/i.test(s);

    list.innerHTML = data.map(p => {
      const id    = pick(p.ID, p.id);
      const name  = pick(p.Name, p.name) || "Ürün";

      // Görsel ham değeri al ve temizle
      let raw = pick(p.ImageURL, p.image_url, p.imageUrl, p.image, p.img);
      raw = (raw ?? "").toString().trim();

      let img;
      if (raw && /^https?:\/\//.test(raw)) {
        // Tam URL ise (http/https)
        img = hasExt(raw) ? raw : "/assets/img/placeholder.png";
      } else if (raw && raw.startsWith("/assets/")) {
        // /assets/... ise
        img = hasExt(raw) ? raw : "/assets/img/placeholder.png";
      } else if (raw) {
        // Dosya adı veya görece yol geldiyse normalize et
        const fname = raw.replace(/^\/+/, "").replace(/^assets\/img\/?/, "");
        img = hasExt(fname) ? `/assets/img/${fname}` : "/assets/img/placeholder.png";
      } else {
        // Hiç görsel yoksa
        img = "/assets/img/placeholder.png";
      }

      const cents = pick(p.PriceCents, p.price_cents, p.priceCents, 0) || 0;
      const price = (Number(cents) / 100).toFixed(2);

      return `
        <div class="card">
          <img src="${img}" alt="${name}" loading="lazy"
               onerror="this.onerror=null;this.src='/assets/img/placeholder.png'">
          <div class="b">
            <div class="t">${name}</div>
            <div class="price">${price} ₺</div>
            <button onclick="addToCart(${id})">Sepete Ekle</button>
          </div>
        </div>
      `;
    }).join("");

  } catch (e) {
    msg("Ürünler yüklenemedi: " + e.message, false);
  }
}
async function(){
  const r = await fetch('/api/me');
  if (r.status === 401) { location.href = '/login.html'; return; }
  // giriş varsa mevcut loadProducts()'ı çağır
  if (typeof loadProducts === 'function') loadProducts();
}();
async function addToCart(id) {
  try {
    const r = await fetch("/api/cart/add", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ product_id: id, qty: 1 })
    });
    if (!r.ok) throw new Error(`HTTP ${r.status}`);
    msg("Sepete eklendi");
  } catch (e) {
    msg("Giriş yapmanız gerekebilir: " + e.message, false);
  }
}

document.addEventListener("DOMContentLoaded", loadProducts);
