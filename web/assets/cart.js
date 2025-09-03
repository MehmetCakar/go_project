const api=async(p,o={})=>{const r=await fetch(p,{method:o.method||'GET',headers:{'Content-Type':'application/json'},body:o.body||null,credentials:'include'});const t=await r.text();let d;try{d=JSON.parse(t)}catch{d=t}if(!r.ok)throw new Error(d?.error||('HTTP '+r.status));return d;};
const msg=(s,ok=true)=>{const el=document.getElementById('msg');el.textContent=s;el.style.color=ok?'#16a34a':'#ef4444';};
async function loadCart(){ try{ const items=await api('/api/cart'); document.getElementById('cartBox').textContent=items.length?items.map(it=>`${it.Qty} x ${it.Product.Name} = ${(it.Product.PriceCents*it.Qty/100).toFixed(2)} ₺`).join('\n'):'Boş'; }catch(e){ document.getElementById('cartBox').textContent='Sepet yüklenemedi: '+e.message; } }
async function checkout(){ try{ const o=await api('/api/checkout',{method:'POST'}); msg(`Sipariş #${o.ID} — Toplam ${(o.TotalCents/100).toFixed(2)} ₺`); loadCart(); }catch(e){ msg('Checkout hata: '+e.message,false); } }
loadCart();
