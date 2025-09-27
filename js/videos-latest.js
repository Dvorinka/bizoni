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

  function ytUrl(videoId){
    return 'https://www.youtube.com/watch?v=' + videoId;
  }

  // Convert common English relative time strings to Czech
  function enRelativeToCz(s) {
    if (!s || typeof s !== 'string') return s;
    const t = s.trim().toLowerCase();
    if (t === 'just now') return 'právě teď';
    // minutes
    let m = t.match(/^(\d+)\s+minute(s)?\s+ago$/);
    if (m) {
      const n = parseInt(m[1], 10);
      return n === 1 ? 'před 1 minutou' : `před ${n} minutami`;
    }
    // hours
    m = t.match(/^(\d+)\s+hour(s)?\s+ago$/);
    if (m) {
      const n = parseInt(m[1], 10);
      return n === 1 ? 'před 1 hodinou' : `před ${n} hodinami`;
    }
    // days
    m = t.match(/^(\d+)\s+day(s)?\s+ago$/);
    if (m) {
      const n = parseInt(m[1], 10);
      return n === 1 ? 'před 1 dnem' : `před ${n} dny`;
    }
    // weeks
    m = t.match(/^(\d+)\s+week(s)?\s+ago$/);
    if (m) {
      const n = parseInt(m[1], 10);
      return n === 1 ? 'před 1 týdnem' : `před ${n} týdny`;
    }
    // months
    m = t.match(/^(\d+)\s+month(s)?\s+ago$/);
    if (m) {
      const n = parseInt(m[1], 10);
      return n === 1 ? 'před 1 měsícem' : `před ${n} měsíci`;
    }
    // years
    m = t.match(/^(\d+)\s+year(s)?\s+ago$/);
    if (m) {
      const n = parseInt(m[1], 10);
      return n === 1 ? 'před 1 rokem' : `před ${n} lety`;
    }
    return s; // fallback unchanged
  }

  function renderFeatured(v) {
    const article = h('article', {class: 'post format-video has-post-thumbnail hentry'});
    const wrap = h('div', {class: 'lte-wrapper'});
    const a = h('a', {href: ytUrl(v.video_id), target: '_blank', class: 'lte-photo lte-video-popup swipebox'});
    const img = h('img', {loading: 'lazy', decoding: 'async', width: '1600', height: '969', src: v.thumbnail_url, class: 'attachment-full size-full wp-post-image', alt: ''});
    const iconWrap = h('span', {class: 'lte-icon-video'});
    iconWrap.appendChild(h('ion-icon', {name: 'play-circle-outline', size: 'large'}));
    iconWrap.appendChild(h('span', {html: v.length || ''}));
    a.appendChild(img);
    a.appendChild(iconWrap);
    wrap.appendChild(a);
    const descr = h('div', {class: 'lte-description'});
    const dateTop = h('span', {class: 'lte-date-top'});
    const dateA = h('a', {href: '', class: 'lte-date'});
    const relText = enRelativeToCz(v.published_text || '') || v.published_date || '';
    dateA.appendChild(h('span', {class: 'dt', html: relText}));
    dateTop.appendChild(dateA);
    const headerA = h('a', {href: ytUrl(v.video_id), class: 'lte-header', target: '_blank'});
    headerA.appendChild(h('h3', {html: v.title || ''}));
    descr.appendChild(dateTop);
    descr.appendChild(headerA);
    // keep layout spacing consistent
    descr.appendChild(h('div', {class: 'lte-excerpt'}));
    article.appendChild(wrap);
    article.appendChild(descr);
    return article;
  }

  function renderGridItem(v){
    const col = h('div', {class: 'items col-xl-6 col-lg-6 col-md-6 col-sm-6 col-ms-12 col-xs-12'});
    const article = h('article', {class: 'post format-video has-post-thumbnail hentry'});
    const wrap = h('div', {class: 'lte-wrapper'});
    const a = h('a', {href: ytUrl(v.video_id), target: '_blank', class: 'lte-photo lte-video-popup swipebox'});
    const img = h('img', {loading: 'lazy', decoding: 'async', width: '1600', height: '969', src: v.thumbnail_url, class: 'attachment-full size-full wp-post-image', alt: ''});
    const iconWrap = h('span', {class: 'lte-icon-video'});
    iconWrap.appendChild(h('ion-icon', {name: 'play-circle-outline', size: 'large'}));
    iconWrap.appendChild(h('span', {html: v.length || ''}));
    a.appendChild(img);
    a.appendChild(iconWrap);
    wrap.appendChild(a);
    const descr = h('div', {class: 'lte-description'});
    const dateTop = h('span', {class: 'lte-date-top'});
    const dateA = h('a', {href: '', class: 'lte-date'});
    const relText2 = enRelativeToCz(v.published_text || '') || v.published_date || '';
    dateA.appendChild(h('span', {class: 'dt', html: relText2}));
    dateTop.appendChild(dateA);
    const headerA = h('a', {href: ytUrl(v.video_id), class: 'lte-header', target: '_blank'});
    headerA.appendChild(h('h3', {html: v.title || ''}));
    descr.appendChild(dateTop);
    descr.appendChild(headerA);
    // keep layout spacing consistent
    descr.appendChild(h('div', {class: 'lte-excerpt'}));
    article.appendChild(wrap);
    article.appendChild(descr);
    col.appendChild(article);
    return col;
  }

  async function loadVideos(){
    const featureMount = document.getElementById('latest-video-feature');
    const gridMount = document.getElementById('latest-videos-grid');
    if (!featureMount && !gridMount) return;
    if (featureMount) featureMount.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#888;">Načítání…</div>';
    if (gridMount) gridMount.innerHTML = '';
    try {
      // Try fast local JSON first
      let data = null;
      const tryUrls = ['/data/video.json', '/data/videos.json', '/api/videos/latest'];
      for (const u of tryUrls){
        try {
          const res = await fetch(u, {credentials: 'omit', cache: 'no-store'});
          if (res.ok) { data = await res.json(); break; }
        } catch (_) {}
      }
      if (!data) throw new Error('No videos data available');
      let items = Array.isArray(data.items) ? data.items : data.Items || [];
      // ensure most recent first: sort by published_date or published_text desc
      const parseDate = (v) => {
        const s = v && (v.published_date || v.published_text || '').trim();
        // try ISO/date parsing
        const t = Date.parse(s);
        return isNaN(t) ? 0 : t;
      };
      items = items.slice().sort((a,b) => parseDate(b) - parseDate(a));
      if (featureMount) featureMount.innerHTML = '';
      if (!items || items.length === 0){
        if (featureMount) featureMount.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#888;">Žádná videa zatím nejsou.</div>';
        return;
      }
      const [first, ...rest] = items;
      if (featureMount && first) {
        const container = document.createElement('div');
        container.className = 'items col-xl-12 col-lg-12 col-md-12 col-sm-12 col-ms-12 col-xs-12';
        container.appendChild(renderFeatured(first));
        featureMount.appendChild(container);
      }
      if (gridMount && rest.length){
        const frag = document.createDocumentFragment();
        rest.forEach(v => frag.appendChild(renderGridItem(v)));
        gridMount.appendChild(frag);
      }
    } catch (e) {
      console.error('videos load error', e);
      if (featureMount) featureMount.innerHTML = '<div style="width:100%;text-align:center;padding:12px;color:#c00;">Nepodařilo se načíst videa.</div>';
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', loadVideos);
  } else {
    loadVideos();
  }
})();
