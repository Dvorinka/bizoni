'use strict';
(function(){
  function h(el, attrs={}, children=[]) {
    const e = document.createElement(el);
    for (const [k,v] of Object.entries(attrs||{})) {
      if (k === 'class') e.className = v; else if (k === 'html') e.innerHTML = v; else e.setAttribute(k, v);
    }
    for (const c of (children||[])) e.appendChild(c);
    return e;
  }

  function renderItem(item){
    const col = h('div', {class: 'col-xl-4 col-lg-6 col-md-6 col-sm-12 col-xs-12 item div-thumbnail'});
    const article = h('article', {class: 'post-25620 post type-post status-publish format-standard has-post-thumbnail hentry'});
    const aPhoto = h('a', {href: item.link, class: 'lte-photo'});
    const img = h('img', {
      src: item.image,
      width: '500', height: '300', decoding: 'async', fetchpriority: 'high',
      class: 'attachment-atleticos-blog size-atleticos-blog wp-post-image', alt: ''
    });
    aPhoto.appendChild(img);
    aPhoto.appendChild(h('span', {class: 'lte-photo-overlay'}));
    const descr = h('div', {class: 'lte-description'});
    const aHeader = h('a', {href: item.link, class: 'lte-header'});
    aHeader.appendChild(h('h3', {html: item.title || ('Článek ' + item.id)}));
    descr.appendChild(aHeader);
    article.appendChild(aPhoto);
    article.appendChild(descr);
    col.appendChild(article);
    return col;
  }

  function numericDesc(a,b){
    const ai = parseInt(a.id,10); const bi = parseInt(b.id,10);
    if (!isNaN(ai) && !isNaN(bi)) return bi - ai;
    return (b.id||'').localeCompare(a.id||'');
  }

  function getQueryParam(name){
    const url = new URL(window.location.href);
    return url.searchParams.get(name);
  }

  function normalize(s){ return (s||'').toString().trim().toLowerCase(); }

  async function loadAll(){
    // Render into the primary masonry grid, overwriting any static items
    const mount = document.querySelector('.lte-blog-wrap .blog .row.masonry');
    if (!mount) return;
    mount.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#888;">Načítání…</div>';
    try {
      const res = await fetch('/api/blog/latest?limit=10000', {credentials: 'omit'});
      if (!res.ok) throw new Error('HTTP '+res.status);
      let items = await res.json();
      if (!Array.isArray(items)) items = [];
      // Sort by numeric ID desc to ensure largest number is latest
      items.sort(numericDesc);

      // Optional filter by category via ?category=XYZ
      const qCat = getQueryParam('category');
      if (qCat) {
        const want = normalize(qCat);
        items = items.filter(it => Array.isArray(it.categories) && it.categories.some(c => normalize(c) === want));
      }

      mount.innerHTML = '';
      if (items.length === 0) {
        mount.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#888;">Žádné příspěvky zatím nejsou.</div>';
        return;
      }
      const frag = document.createDocumentFragment();
      items.forEach(it => frag.appendChild(renderItem(it)));
      mount.appendChild(frag);
    } catch (e) {
      console.error('Load blog list error', e);
      mount.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#c00;">Nepodařilo se načíst seznam článků.</div>';
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', loadAll);
  } else {
    loadAll();
  }
})();
