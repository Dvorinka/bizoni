// team-switcher.js
// Loads team data from XML and populates the team slider. Adds a men/women switcher.

(function () {
  const XML_URL = 'data/team.xml';
  const SWITCHER_ID = 'gender-switcher';
  const WRAPPER_ID = 'team-swiper-wrapper-1';
  const PRELOADER_ID = 'team-preloader-1';
  const SECTION_ID = 'team-section-1';

  let teamData = null; // cached parsed XML data
  let currentGender = 'men';
  // Autoscroll timers
  let autoTimer = null;
  let resumeTimer = null;
  const AUTO_DELAY = 2000; // 5s
  const RESUME_AFTER = 10000; // resume 10s after user interaction

  function qs(sel, root = document) { return root.querySelector(sel); }
  function qsa(sel, root = document) { return Array.from(root.querySelectorAll(sel)); }

  async function loadXML() {
    if (teamData) return teamData;
    const res = await fetch(XML_URL, { cache: 'no-cache' });
    if (!res.ok) throw new Error('Failed to fetch team.xml');
    const text = await res.text();
    const parser = new DOMParser();
    const xml = parser.parseFromString(text, 'application/xml');
    const parseError = xml.querySelector('parsererror');
    if (parseError) throw new Error('Invalid XML in team.xml');
    teamData = xml;
    return xml;
  }

  function getMembersByCategory(xml, categoryName) {
    const cat = Array.from(xml.querySelectorAll('team > category'))
      .find(c => (c.getAttribute('name') || '').toLowerCase() === categoryName);
    if (!cat) return [];
    return Array.from(cat.querySelectorAll('member')).map(m => ({
      name: (m.querySelector('name')?.textContent || '').trim(),
      number: (m.querySelector('number')?.textContent || '').trim(),
      role: (m.querySelector('role')?.textContent || '').trim(),
      image: (m.querySelector('image')?.textContent || '').trim(),
    }));
  }

  function slideHTML(member) {
    const numHTML = member.number ? `<div class="lte-num">${member.number}</div>` : '<div class="lte-num"></div>';
    const safeImg = member.image || '';
    return (
      `<div class="lte-item swiper-slide">
        <div class="lte-team-item">
          <a class="lte-image" style="background-image: url()">
            <img loading="lazy" decoding="async" width="800" height="1200" src="${safeImg}" class="attachment-full size-full" />
          </a>
          <div class="lte-descr">
            ${numHTML}
            <a href="${safeImg}" target="_blank">
              <h4 class="lte-header">${member.name}</h4>
            </a>
            <p class="lte-subheader" style="color: #c42221">${member.role}
            </p>
          </div>
        </div>
      </div>`
    );
  }

  function renderMembers(members) {
    const wrapper = document.getElementById(WRAPPER_ID);
    if (!wrapper) return;
    const swiperEl = wrapper.closest('.swiper-container');
    const swiper = swiperEl && swiperEl.swiper;

    // Use DOM-based rendering to match theme's slider expectations
    wrapper.innerHTML = members.map(slideHTML).join('');

    // Strong refresh
    if (swiper) {
      try {
        if (typeof swiper.updateSlides === 'function') swiper.updateSlides();
        if (typeof swiper.updateSize === 'function') swiper.updateSize();
        if (typeof swiper.updateAutoHeight === 'function') swiper.updateAutoHeight(0);
        if (typeof swiper.slideTo === 'function') swiper.slideTo(0, 0, false);
        if (typeof swiper.update === 'function') swiper.update();
      } catch (e) {}
    }
    // Ask the theme to re-init this slider completely so arrows/loop/order are consistent
    const sliderContainer = wrapper.closest('.lte-swiper-slider');
    // For consistent sequential navigation, disable coverflow loop and multi-view
    if (sliderContainer && sliderContainer.dataset) {
      sliderContainer.dataset.effect = 'slide'; // avoid forced loop in coverflow
      sliderContainer.dataset.loop = '0';
      sliderContainer.dataset.breakpoints = '1;1;1;1;1;1'; // 1 per view on all widths
    }
    if (sliderContainer) sliderContainer.classList.remove('lte-inited');
    if (typeof window.initSwiperWrappers === 'function') {
      try { window.initSwiperWrappers(); } catch (_) {}
    }
    // Remove any duplicate arrow bars the theme may have added on re-init
    cleanupDuplicateArrows();
    setTimeout(() => window.dispatchEvent(new Event('resize')), 0);

    // Ensure arrows exist and are bound; manual endless wrap
    setupEndlessNavigation(swiperEl);
    setupDragWrap(swiper);
    // Restart autoscroll on fresh render
    stopAutoScroll();
    startAutoScroll();
  }

  // Keep only one arrows bar; prefer the one whose anchors already have our data-ts-bound
  function cleanupDuplicateArrows() {
    const wrapper = document.getElementById(WRAPPER_ID);
    if (!wrapper) return;
    const slider = wrapper.closest('.lte-swiper-slider');
    if (!slider) return;

    // Arrows can be siblings of slider or children inside slider depending on theme config
    const candidates = [];
    const parent = slider.parentElement;
    if (parent) {
      Array.from(parent.children).forEach((el) => { if (el.classList && el.classList.contains('lte-arrows')) candidates.push(el); });
    }
    Array.from(slider.children).forEach((el) => { if (el.classList && el.classList.contains('lte-arrows')) candidates.push(el); });

    if (candidates.length <= 1) return;

    // Prefer the one that already has data-ts-bound anchors
    const hasBound = candidates.find(a => a.querySelector('a[data-ts-bound="1"]'));
    const keep = hasBound || candidates[0];
    candidates.forEach((a) => { if (a !== keep && a.parentElement) a.parentElement.removeChild(a); });
  }

  async function switchGender(gender) {
    currentGender = gender;
    try {
      showPreloader();
      const xml = await loadXML();
      // Keep the order as in XML so the first visible is the first listed (e.g., Janečka Martin)
      const list = getMembersByCategory(xml, gender);
      renderMembers(list);
      updateActiveButton();
      markReady();
      hidePreloader();
    } catch (e) {
      console.error(e);
      hidePreloader();
    }
  }

  function updateActiveButton() {
    const container = document.getElementById(SWITCHER_ID);
    if (!container) return;
    qsa('button[data-gender]', container).forEach(btn => {
      btn.classList.toggle('active', btn.dataset.gender === currentGender);
    });
  }

  function bindUI() {
    const container = document.getElementById(SWITCHER_ID);
    if (!container) return;
    container.addEventListener('click', (e) => {
      const btn = e.target.closest('button[data-gender]');
      if (!btn) return;
      const gender = btn.dataset.gender;
      if (gender && gender !== currentGender) switchGender(gender);
    });
  }

  function ensureBasicStyles() {
    const css = `
      #${SWITCHER_ID}{display:flex;gap:.5rem;justify-content:center;margin:10px 0}
      #${SWITCHER_ID} .switch-btn{background:#eee;border:1px solid #ccc;border-radius:20px;padding:.35rem .9rem;font-weight:600;cursor:pointer}
      #${SWITCHER_ID} .switch-btn.active{background:#111;color:#fff;border-color:#111}
      #${PRELOADER_ID}{display:none;align-items:center;justify-content:center;gap:.6rem;color:#fff;padding:8px 0}
      #${PRELOADER_ID}.visible{display:flex}
      #${PRELOADER_ID} .spinner{width:16px;height:16px;border:2px solid rgba(255,255,255,.3);border-top-color:#fff;border-radius:50%;animation:ts-spin .8s linear infinite}
      @keyframes ts-spin{to{transform:rotate(360deg)}}
      #${SECTION_ID}.not-ready .lte-swiper-slider-wrapper{visibility:hidden}
    `;
    const style = document.createElement('style');
    style.textContent = css;
    document.head.appendChild(style);
  }

  function getSwiperInstance() {
    const wrapper = document.getElementById(WRAPPER_ID);
    const swiperEl = wrapper && wrapper.closest('.swiper-container');
    return swiperEl && swiperEl.swiper ? { el: swiperEl, api: swiperEl.swiper } : null;
  }

  function setupEndlessNavigation(swiperContainerEl) {
    const inst = getSwiperInstance();
    if (!inst) return;
    const { el, api } = inst;

    // Ensure only one set of arrows remains before binding
    cleanupDuplicateArrows();

    // Theme uses .lte-arrow-left / .lte-arrow-right (see frontend.js init)
    let nextBtn = el.parentElement && el.parentElement.querySelector('.lte-arrows .lte-arrow-right');
    let prevBtn = el.parentElement && el.parentElement.querySelector('.lte-arrows .lte-arrow-left');
    // Fallback to common Swiper classes if theme structure changes
    if (!nextBtn) nextBtn = el.querySelector('.swiper-button-next, .lte-swiper-button-next, .lte-next, .lte-arrow-next, .lte-arrow-right');
    if (!prevBtn) prevBtn = el.querySelector('.swiper-button-prev, .lte-swiper-button-prev, .lte-prev, .lte-arrow-prev, .lte-arrow-left');

    // Do not create fallback arrows; rely on theme arrows only

    function bind(btn, dir) {
      if (!btn || btn.dataset.tsBound) return;
      btn.addEventListener('click', (e) => {
        e.preventDefault();
        if (!api) return;
        // User override: pause and schedule resume
        stopAutoScroll();
        scheduleAutoResume();
        if (dir === 'next') {
          if (typeof api.slideNext === 'function') api.slideNext(400);
          else api.slideTo((api.activeIndex || 0) + 1, 400, false);
        } else {
          if (typeof api.slidePrev === 'function') api.slidePrev(400);
          else api.slideTo(Math.max((api.activeIndex || 0) - 1, 0), 400, false);
        }
      });
      btn.dataset.tsBound = '1';
    }

    bind(nextBtn, 'next');
    bind(prevBtn, 'prev');

    // Hover pause/resume on the whole slider area
    if (el && !el.__tsHoverBound) {
      el.addEventListener('mouseenter', () => stopAutoScroll());
      el.addEventListener('mouseleave', () => startAutoScroll());
      el.__tsHoverBound = true;
    }
  }

  function setupDragWrap(swiper) {
    if (!swiper || !swiper.on) return;
    if (!swiper.__tsWrapBound) {
      swiper.on('reachEnd', () => { swiper.slideTo(0, 400, false); });
      swiper.on('reachBeginning', () => {
        const last = (swiper.slides && swiper.slides.length ? swiper.slides.length - 1 : 0);
        swiper.slideTo(last, 400, false);
      });
      swiper.__tsWrapBound = true;
    }

    // Pause autoscroll on user touch/drag and schedule resume on release
    if (!swiper.__tsAutoBound) {
      try {
        swiper.on('touchStart', () => { stopAutoScroll(); });
        swiper.on('touchEnd', () => { scheduleAutoResume(); });
        swiper.on('pointerDown', () => { stopAutoScroll(); });
        swiper.on('pointerUp', () => { scheduleAutoResume(); });
      } catch (_) {}
      swiper.__tsAutoBound = true;
    }
  }

  function startAutoScroll() {
    const inst = getSwiperInstance();
    if (!inst) return;
    const { api } = inst;
    stopAutoScroll();
    autoTimer = window.setInterval(() => {
      if (!api) return;
      try {
        // If not looping, wrap to first when at end
        const loop = api.params && api.params.loop;
        if (!loop && api.isEnd) {
          api.slideTo(0, 600, false);
        } else if (typeof api.slideNext === 'function') {
          api.slideNext(600);
        }
      } catch (_) {}
    }, AUTO_DELAY);
  }

  function stopAutoScroll() {
    if (autoTimer) {
      clearInterval(autoTimer);
      autoTimer = null;
    }
    if (resumeTimer) {
      clearTimeout(resumeTimer);
      resumeTimer = null;
    }
  }

  function scheduleAutoResume() {
    if (resumeTimer) {
      clearTimeout(resumeTimer);
      resumeTimer = null;
    }
    resumeTimer = window.setTimeout(() => {
      startAutoScroll();
    }, RESUME_AFTER);
  }

  function showPreloader() {
    const el = document.getElementById(PRELOADER_ID);
    if (el) el.classList.add('visible');
  }

  function hidePreloader() {
    const el = document.getElementById(PRELOADER_ID);
    if (el) el.classList.remove('visible');
  }

  function markNotReady() {
    const sec = document.getElementById(SECTION_ID);
    if (sec) sec.classList.add('not-ready');
  }

  function markReady() {
    const sec = document.getElementById(SECTION_ID);
    if (sec) sec.classList.remove('not-ready');
  }

  document.addEventListener('DOMContentLoaded', () => {
    ensureBasicStyles();
    markNotReady();
    showPreloader();
    bindUI();
  });

  // Defer initial population until all assets and theme scripts (e.g., sliders) are fully initialized
  window.addEventListener('load', () => {
    switchGender(currentGender);
  });
})();
