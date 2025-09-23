'use strict';
(function(){
  const KEY = 'adminAuthB64'; // stores base64 of user:pass

  function b64(u, p){
    try { return btoa(`${u}:${p}`); } catch { return ''; }
  }

  function get(){
    try { return localStorage.getItem(KEY) || ''; } catch { return ''; }
  }
  function set(val){
    try { localStorage.setItem(KEY, val||''); } catch {}
  }
  function clear(){
    try { localStorage.removeItem(KEY); } catch {}
  }

  window.AdminAuth = {
    has(){ return !!get(); },
    getHeaders(){ const v = get(); return v ? { 'Authorization': 'Basic '+v } : {}; },
    setCreds(user, pass){ set(b64(user, pass)); },
    clear(){ clear(); }
  };

  // small UI helper (optional) – appears bottom-left
  function ensureWidget(){
    if (document.getElementById('admin-auth-widget')) return;
    const wrap = document.createElement('div');
    wrap.id = 'admin-auth-widget';
    wrap.style.position = 'fixed';
    wrap.style.left = '12px';
    wrap.style.bottom = '12px';
    wrap.style.zIndex = '9999';
    wrap.style.display = 'flex';
    wrap.style.gap = '6px';

    const btnSet = document.createElement('button');
    btnSet.textContent = 'Přihlásit';
    btnSet.style.padding = '6px 10px';
    btnSet.style.borderRadius = '8px';
    btnSet.style.border = '1px solid #cbd5e1';
    btnSet.style.background = '#fff';
    btnSet.addEventListener('click', () => {
      const u = prompt('Uživatel (e-mail):');
      if (!u) return;
      const p = prompt('Heslo:');
      if (p == null) return;
      window.AdminAuth.setCreds(u, p);
      alert('Přihlašovací údaje uloženy do tohoto prohlížeče.');
    });

    const btnClr = document.createElement('button');
    btnClr.textContent = 'Odhlásit';
    btnClr.style.padding = '6px 10px';
    btnClr.style.borderRadius = '8px';
    btnClr.style.border = '1px solid #cbd5e1';
    btnClr.style.background = '#fff';
    btnClr.addEventListener('click', () => {
      window.AdminAuth.clear();
      alert('Odhlášeno – uložené údaje odstraněny.');
    });

    wrap.appendChild(btnSet);
    wrap.appendChild(btnClr);
    document.body.appendChild(wrap);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', ensureWidget);
  } else {
    ensureWidget();
  }
})();
