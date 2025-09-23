'use strict';
(function(){
  async function updateHeroFromLatest() {
    try {
      const res = await fetch('/api/blog/latest?limit=8', {credentials:'omit'});
      if (!res.ok) throw new Error('HTTP '+res.status);
      let items = await res.json();
      if (!Array.isArray(items) || items.length === 0) items = [];

      // Update background images of zoom slider slides if present
      const slides = document.querySelectorAll('.lte-slider-zoom .zs-slides .zs-slide');
      // If slides are not yet initialized by the plugin, try again shortly
      if (!slides || slides.length === 0) {
        setTimeout(updateHeroFromLatest, 600);
        return;
      }

      // Track which IDs are used in the carousel
      const usedIds = [];

      // Slide 0 is always the intro post 0000
      const slide0 = slides[0];
      if (slide0) {
        usedIds.push('0000');
        slide0.style.backgroundImage = `url('img/blog/0000.png')`;
        const content0 = document.querySelector(`.lte-zs-slider-inner.lte-zs-slide-0`);
        if (content0) {
          const btn0 = content0.querySelector('a.lte-btn');
          if (btn0) btn0.setAttribute('href', 'blog/0000.html');
          // Do not alter existing H2 text; designers may have custom text
        }
      }

      // Fill remaining slides with latest posts, skipping 0000
      const rest = items.filter(it => it && it.id !== '0000');
      const max = Math.min(rest.length, Math.max(0, slides.length - 1));
      for (let i = 0; i < max; i++) {
        const it = rest[i];
        if (it && it.id) usedIds.push(it.id);
        const slide = slides[i+1];
        if (!slide) continue;
        slide.style.backgroundImage = `url('${it.image}')`;
        const content = document.querySelector(`.lte-zs-slider-inner.lte-zs-slide-${i+1}`);
        if (content) {
          const header = content.querySelector('h2.lte-header');
          if (header) header.textContent = it.title || '';
          const btn = content.querySelector('a.lte-btn');
          if (btn) btn.setAttribute('href', it.link);
        }
      }

      // Expose used IDs so other widgets can exclude them (e.g., 4-post grid)
      window.CAROUSEL_BLOG_IDS = usedIds;
    } catch (e) {
      console.error('home-autofill hero update error', e);
    }
  }

  function onReady(fn){
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', fn); else fn();
  }

  onReady(() => {
    // Delay a bit to allow zoomslider to initialize, then update
    setTimeout(updateHeroFromLatest, 500);
  });
})();
