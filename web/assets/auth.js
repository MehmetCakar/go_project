// web/assets/auth.js

// Basit fetch wrapper
async function api(path, opts = {}) {
  const res = await fetch(path, {
    method: opts.method || "POST",
    headers: {"Content-Type": "application/json"},
    body: opts.body ?? null,
    credentials: "include",
  });
  const txt = await res.text();
  let data; try { data = JSON.parse(txt); } catch { data = txt; }
  if (!res.ok) throw new Error(data?.error || ("HTTP " + res.status));
  return data;
}

function setMsg(s, ok = true) {
  const el = document.getElementById("msg");
  if (el) {
    el.textContent = s;
    el.style.color = ok ? "#16a34a" : "#ef4444";
  }
}

window.verifyCode = async function(){
  const email = document.getElementById('codeEmail').value.trim();
  const code  = document.getElementById('codeInput').value.trim();
  try{
    const r = await fetch('/api/auth/verify-code',{
      method:'POST',
      headers:{'Content-Type':'application/json'},
      credentials:'include',
      body: JSON.stringify({email, code})
    });
    const data = await r.json();
    if(!r.ok) throw new Error(data?.error || r.statusText);
    // Başarılı: ana sayfaya (ürünler) yönlendir
    location.href = '/';
  }catch(e){
    setMsg('Doğrulama hatası: '+e.message, false);
  }
}


// Kayıt
async function register() {
  const email = document.getElementById("regEmail")?.value.trim();
  const password = document.getElementById("regPass")?.value || "";
  if (!email || !password) return setMsg("Email ve şifre gerekli.", false);

  try {
    await api("/api/auth/register", { body: JSON.stringify({ email, password }) });
    setMsg("Kayıt başarılı. E-postadaki doğrulama linkine tıkla.");
  } catch (e) {
    setMsg("Kayıt hata: " + e.message, false);
  }
}

// Giriş (başarılıysa ürünlere yönlendir)
async function login() {
  const email = document.getElementById("logEmail")?.value.trim();
  const password = document.getElementById("logPass")?.value || "";
  if (!email || !password) return setMsg("Email ve şifre gerekli.", false);

  try {
    await api("/api/auth/login", { body: JSON.stringify({ email, password }) });
    setMsg("Giriş başarılı.");
    // varsa ?redirect=... parametresine, yoksa ana sayfaya
    const params = new URLSearchParams(window.location.search);
    const to = params.get("redirect") || "/";
    window.location.href = to;
  } catch (e) {
    setMsg("Giriş hata: " + e.message, false);
  }
}

// Çıkış
async function logout() {
  try {
    await api("/api/auth/logout", {});
    setMsg("Çıkış yapıldı.");
    // İstersen doğrudan ana sayfaya da alabilirsin:
    // window.location.href = "/";
  } catch (e) {
    setMsg("Çıkış hata: " + e.message, false);
  }
}

// DOM hazır olunca butonlara bağla + Enter kısayolları
document.addEventListener("DOMContentLoaded", () => {
  const $ = (id) => document.getElementById(id);

  $("btnRegister")?.addEventListener("click", register);
  $("btnLogin")?.addEventListener("click", login);
  $("btnLogout")?.addEventListener("click", logout);

  // Enter ile kayıt
  $("regPass")?.addEventListener("keydown", (e) => {
    if (e.key === "Enter") register();
  });
  // Enter ile giriş
  $("logPass")?.addEventListener("keydown", (e) => {
    if (e.key === "Enter") login();
  });

  // Teşhis için (konsolda function görmelisin)
  console.log("auth.js yüklendi — register:", typeof register, "login:", typeof login);
});
