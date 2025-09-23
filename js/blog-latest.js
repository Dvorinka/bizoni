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
    const col = h('div', {class: 'items col-xl-6 col-lg-6 col-md-6 col-sm-6 col-ms-6 col-xs-12'});
    const article = h('article', {class: 'post-25620 post type-post status-publish format-standard has-post-thumbnail hentry'});

    const aPhoto = h('a', {href: item.link, class: 'lte-photo'});
    const img = h('img', {
      src: item.image,
      width: '500',
      height: '300',
      decoding: 'async',
      fetchpriority: 'high',
      class: 'attachment-atleticos-blog size-atleticos-blog wp-post-image',
      alt: ''
    });
    aPhoto.appendChild(img);
    aPhoto.appendChild(h('span', {class: 'lte-photo-overlay'}));

    const descr = h('div', {class: 'lte-description'});
    const aHeader = h('a', {href: item.link, class: 'lte-header'});
    const h3 = h('h3', {html: item.title || ('Článek ' + item.id)});
    aHeader.appendChild(h3);
    // const excerpt = h('div', {class: 'lte-excerpt', html: ''});

    descr.appendChild(aHeader);
    // descr.appendChild(excerpt);

    article.appendChild(aPhoto);
    article.appendChild(descr);
    col.appendChild(article);
    return col;
  }

  async function loadLatest(attempt = 0) {
    const primary = document.getElementById('latest-blog-items');
    const secondary = document.getElementById('other-blog-items');
    if (!primary && !secondary) return;
    if (primary) primary.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#888;">Načítání…</div>';
    if (secondary) secondary.innerHTML = '';
    try {
      const res = await fetch('/api/blog/latest?limit=12', {credentials: 'omit'});
      if (!res.ok) throw new Error('HTTP '+res.status);
      let items = await res.json();
      if (primary) primary.innerHTML = '';
      if (!Array.isArray(items) || items.length === 0) {
        if (primary) primary.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#888;">Žádné příspěvky zatím nejsou.</div>';
        return;
      }

      // Exclude carousel-used posts and pinned 0000 from the 4-grid
      const hasCarouselIds = Array.isArray(window.CAROUSEL_BLOG_IDS) && window.CAROUSEL_BLOG_IDS.length > 0;
      // If carousel IDs are not ready yet, retry shortly (max 5 attempts)
      if (!hasCarouselIds && attempt < 5) {
        setTimeout(() => loadLatest(attempt + 1), 300);
        return;
      }
      const excluded = new Set(hasCarouselIds ? window.CAROUSEL_BLOG_IDS : []);
      const four = items.filter(it => it && it.id !== '0000' && !excluded.has(it.id)).slice(0, 4);

      // Prefer rendering into primary; keep secondary empty to avoid duplicates
      const target = primary || secondary;
      if (target) {
        const frag = document.createDocumentFragment();
        four.forEach(it => frag.appendChild(renderItem(it)));
        target.appendChild(frag);
      }
      if (secondary && secondary !== target) secondary.innerHTML = '';
    } catch (e) {
      console.error('Load latest blog error', e);
      if (primary) primary.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#c00;">Nepodařilo se načíst novinky.</div>';
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => loadLatest(0));
  } else {
    loadLatest(0);
  }
})();
