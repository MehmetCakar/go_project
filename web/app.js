// basit durum
const state = { token: null };

// genel fetch helper (aynı origin)
async function api(path, opts = {}) {
  const headers = Object.assign({ 'Content-Type': 'application/json' }, opts.headers || {});
  // JWT varsa Authorization ekle (cookie de var ama ikisi birden çalışır)
  if (state.token) headers['Authorization'] = 'Bearer ' + state.token;

  const res = await fetch(path, {
    method: opts.method || 'GET',
    headers,
    body: opts.body || null,
    credentials: 'include', // httpOnly cookie otomatik taşınsın
  });
  const text = await res.text();
  let data;
  try { data = JSON.parse(text); } catch { data = text; }
  if (!res.ok) throw new Error((data && data.error) || ('HTTP ' + res.status));
  return data;
}

// mesaj göster
function setMsg(t, ok = true) {
  const el = document.getElementById('msg');
  el.textContent = t;
  el.style.color = ok ? 'green' : 'crimson';
}

// Kayıt
async function register() {
  const email = document.getElementById('regEmail').value.trim();
  const password = document.getElementById('regPass').value;
  try {
    await api('/api/auth/register', { method: 'POST', body: JSON.stringify({ email, password }) });
    setMsg('Kayıt başarılı! MailHog’dan verify linkine tıkla: http://localhost:8025', true);
  } catch (e) { setMsg('Kayıt hata: ' + e.message, false); }
}

// Login
async function login() {
  const email = document.getElementById('logEmail').value.trim();
  const password = document.getElementById('logPass').value;
  try {
    const r = await api('/api/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) });
    state.token = r.token; // ayrıca httpOnly cookie de set edildi
    document.getElementById('who').textContent = 'Giriş yapıldı: ' + email;
    setMsg('Giriş başarılı', true);
    loadCart();
  } catch (e) { setMsg('Giriş hata: ' + e.message, false); }
}

async function logout() {
  try {
    await api('/api/auth/logout', { method: 'POST' });
  } catch (_) {}
  state.token = null;
  document.getElementById('who').textContent = '';
  setMsg('Çıkış yapıldı', true);
}

// Ürünleri yükle
async function loadProducts() {
  try {
    const data = await api('/api/products');
    const root = document.getElementById('list');
    root.innerHTML = data.map(p => `
      <div class="p">
        <img src="${p.ImageURL}" alt="${p.Name}">
        <div class="b">
          <div><b>${p.Name}</b></div>
          <div class="price">${(p.PriceCents/100).toFixed(2)} ₺</div>
          <button onclick="addToCart(${p.ID})">Sepete Ekle</button>
        </div>
      </div>
    `).join('');
  } catch (e) { setMsg('Ürün yükleme hata: ' + e.message, false); }
}

// Sepete ekle
async function addToCart(pid) {
  try {
    await api('/api/cart/add', { method: 'POST', body: JSON.stringify({ product_id: pid, qty: 1 }) });
    setMsg('Sepete eklendi', true);
    loadCart();
  } catch (e) { setMsg('Sepete ekleme hata: ' + e.message + ' (Giriş yaptıktan sonra deneyin.)', false); }
}

// Sepeti getir
async function loadCart() {
  try {
    const items = await api('/api/cart');
    const txt = items.map(it => `${it.Qty} x ${it.Product.Name}  = ${(it.Product.PriceCents*it.Qty/100).toFixed(2)} ₺`).join('\n');
    document.getElementById('cartBox').textContent = items.length ? txt : 'Boş';
  } catch (e) {
    document.getElementById('cartBox').textContent = 'Sepet yüklenemedi (giriş gerekli).';
  }
}

// Checkout
async function checkout() {
  try {
    const order = await api('/api/checkout', { method: 'POST' });
    setMsg(`Sipariş oluşturuldu #${order.ID} — Toplam: ${(order.TotalCents/100).toFixed(2)} ₺`, true);
    loadCart();
  } catch (e) { setMsg('Checkout hata: ' + e.message, false); }
}

loadProducts();
