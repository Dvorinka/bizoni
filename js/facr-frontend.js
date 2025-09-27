(function(){
  const DATA_URL_JSON = '/data/club.json';
  const TZ = 'Europe/Prague';

  let state = {
    data: null,
    compIndex: 0,
    matchIndex: 0,
    intervalId: null,
    upcomingTimerId: null,
  };

  function parseCZDate(s){
    try{
      // format: 02.01.2006 15:04
      const [d, t] = s.split(' ');
      const [day, month, year] = d.split('.').map(Number);
      const [hour, minute] = t.split(':').map(Number);
      // Interpret as local time (Europe/Prague). Using local constructor avoids UTC offset skew.
      const dt = new Date(year, month-1, day, hour, minute);
      return dt;
    }catch(e){ return null; }
  }

  // using original logo URLs; no cleaning/proxy

  function ensureFacrStyles(){
    if(document.getElementById('facr-styles')) return;
    const css = `
      /* logo background unchanged */
      .facr-nav{ background:#ffffff22; border:1px solid #ffffff55; color:#fff; padding:6px 10px; border-radius:6px; cursor:pointer; backdrop-filter: blur(2px); }
      .facr-nav:hover{ background:#ffffff40; }
      .facr-nav:disabled{ opacity:.5; cursor:default; }
      .facr-tab{ padding:6px 10px; margin:4px; border-radius:16px; border:1px solid #c42221; color:#c42221; background:#ffffff; font-weight:600; }
      .facr-tab.active{ background:#c42221; color:#ffffff; }
      .facr-tab:hover{ background:#c42221cc; color:#ffffff; }
      .facr-inline-status{ margin-left:8px; font-weight:700; font-size:14px; white-space:nowrap; color:inherit; display:inline-block; vertical-align:middle; }
      /* Default (desktop): show middle only */
      #facr-countdown{ display:none !important; }
      .facr-inline-status{ display:none !important; }
      .facr-mob-center-score{ display:none; font-weight:700; font-size:24px; line-height:1; }
      /* Mobile: keep only the bottom countdown by default */
      @media (max-width: 767px){
        #facr-mid{ display:none !important; }
        #facr-countdown{ display:block !important; }
        .facr-inline-status{ display:none !important; }
        .facr-mob-center-score{ display:inline-block !important; }
        /* If finished, hide the bottom countdown */
        .facr-finished #facr-countdown{ display:none !important; }
      }
      @media (max-width: 480px){
        #facr-mid{ font-size:28px !important; min-width:100px; }
        .facr-tab{ padding:4px 8px; font-size:14px; }
        .facr-inline-status{ font-size:12px; margin-left:6px; }
        .facr-nav{ padding:4px 8px; }
      }
    `;
    const style = document.createElement('style');
    style.id = 'facr-styles';
    style.type = 'text/css';
    style.appendChild(document.createTextNode(css));
    document.head.appendChild(style);
  }

  function fmtCountdown(ms){
    if(ms <= 0) return '0m';
    const totalMin = Math.floor(ms/60000);
    const d = Math.floor(totalMin/(60*24));
    const h = Math.floor((totalMin - d*60*24)/60);
    const m = totalMin % 60;
    const parts = [];
    if(d) parts.push(`${d}d`);
    if(h || d) parts.push(`${h}h`);
    parts.push(`${m}m`);
    return parts.join(' ');
  }

  function fmtCountdownLong(ms){
    if(ms < 0) ms = 0;
    const totalSec = Math.floor(ms/1000);
    const d = Math.floor(totalSec / (24*3600));
    const h = Math.floor((totalSec % (24*3600)) / 3600);
    const m = Math.floor((totalSec % 3600) / 60);
    const s = totalSec % 60;
    const dd = d > 0 ? `${d}d ` : '';
    const hh = String(h).padStart(2,'0');
    const mm = String(m).padStart(2,'0');
    const ss = String(s).padStart(2,'0');
    return `${dd}${hh}:${mm}:${ss}`.trim();
  }

  function todayCZ(){
    const now = new Date();
    const dd = String(now.getDate()).padStart(2,'0');
    const mm = String(now.getMonth()+1).padStart(2,'0');
    const yyyy = now.getFullYear();
    return `${dd}.${mm}.${yyyy}`;
  }

  async function fetchData(){
    const res = await fetch(DATA_URL_JSON, { cache: 'no-cache' });
    if(!res.ok) throw new Error('Failed to fetch data');
    const data = await res.json();
    state.data = data;
    return data;
  }

  function escapeHTML(s){
    if(s == null) return '';
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function truncate(s, max){
    const str = s == null ? '' : String(s);
    if(max <= 0) return '';
    return str.length > max ? str.slice(0, max - 1) + '…' : str;
  }

  function isWithinMatchWindow(){
    if(!state.data) return false;
    const now = new Date();
    for(const comp of state.data.club_detail.competitions || []){
      for(const m of comp.matches || []){
        const dt = parseCZDate(m.date_time);
        if(!dt) continue;
        const diffMs = Math.abs(now - dt);
        if(diffMs <= 2*60*60*1000) return true;
      }
    }
    return false;
  }

  function upcomingMatchesAll(){
    const list = [];
    if(!state.data) return list;
    const now = new Date();
    const windowStart = new Date(now.getTime() - 3*24*60*60*1000);
    for(const comp of state.data.club_detail.competitions || []){
      const candidates = (comp.matches || [])
        .map(m=>({ comp, match: m, dt: parseCZDate(m.date_time) }))
        .filter(x=> x.dt && x.dt >= windowStart)
        .sort((a,b)=> a.dt - b.dt);
      if(candidates.length === 0) continue;
      // Prefer the latest recently finished match within the 3-day window;
      // if none, then choose the next upcoming; if none, fall back to the latest overall in window
      let pick = null;
      // find latest finished (dt < now) within window
      for(let i = candidates.length - 1; i >= 0; i--){
        if(candidates[i].dt < now){ pick = candidates[i]; break; }
      }
      if(!pick){
        pick = candidates.find(x=> x.dt >= now) || candidates[candidates.length - 1];
      }
      list.push(pick);
    }
    // sort resulting per-competition picks by time ascending for navigation
    list.sort((a,b)=> a.dt - b.dt);
    return list;
  }

  function renderUpcoming(){
    const root = document.getElementById('facr-upcoming');
    if(!root) return;
    ensureFacrStyles();

    // clear any previous per-second timer
    if(state.upcomingTimerId){
      clearInterval(state.upcomingTimerId);
      state.upcomingTimerId = null;
    }

    const items = upcomingMatchesAll();
    if(items.length === 0){
      root.innerHTML = '<div class="lte-football-upcoming"><span class="lte-header lte-header-upcoming">Žádné nadcházející zápasy</span></div>';
      return;
    }

    // Selection policy:
    // 1) If there is any finished match within the last 3 days across competitions,
    //    show the latest such finished match (keep result visible for 3 days)
    // 2) Otherwise, show the first future match
    // 3) If none, show the latest overall (shouldn't happen as items filtered by 3d window per-comp)
    const now = new Date();
    const threeDms = 3*24*60*60*1000;
    let latestRecentIdx = -1;
    let latestRecentTime = -Infinity;
    items.forEach((it, i) => {
      const dtms = it.dt.getTime();
      if(dtms <= now.getTime() && now.getTime() - dtms <= threeDms){
        if(dtms > latestRecentTime){ latestRecentTime = dtms; latestRecentIdx = i; }
      }
    });
    let preferredIdx = latestRecentIdx;
    if(preferredIdx === -1){
      preferredIdx = items.findIndex(it => it.dt >= now);
      if(preferredIdx === -1) preferredIdx = items.length - 1;
    }
    const idx = Math.min(state.matchIndex || preferredIdx, items.length-1);
    const { comp, match:m } = items[idx];
    const compName = truncate(escapeHTML(comp.name || comp.code || 'Soutěž'), 60);

    const homeLogo = m.home_logo_url || 'img/logo.png';
    const awayLogo = m.away_logo_url || 'img/logo.png';
    const facrLink = m.facr_link || comp.matches_link || state.data.club_detail.url || '#';
    const UP_MAX = 24;
    const homeName = truncate(escapeHTML(m.home), UP_MAX);
    const awayName = truncate(escapeHTML(m.away), UP_MAX);
    const dateVenue = truncate(escapeHTML(m.date_time + (m.venue?`, ${m.venue}`:'')), 40);

    // Determine if match is today (CZ date) and precompute mid display text
    const matchDayCZ = (m.date_time || '').split(' ')[0];
    const isToday = matchDayCZ === todayCZ();
    const startDt = m.date_time ? parseCZDate(m.date_time) : null;
    const diffMsPre = startDt ? (startDt.getTime() - new Date().getTime()) : 0;
    const midText = (diffMsPre > 0) ? `Za ${fmtCountdownLong(diffMsPre)}` : (m.score || '-');

    // Determine status flags and parse score parts if any
    const now2_forTpl = new Date();
    const startMs_forTpl = m.date_time ? parseCZDate(m.date_time).getTime() : 0;
    const diff_forTpl = startMs_forTpl - now2_forTpl.getTime();
    const twoH_forTpl = 2*60*60*1000;
    const threeD_forTpl = 3*24*60*60*1000;
    const isFuture = diff_forTpl > 0;
    const isLive = Math.abs(diff_forTpl) <= twoH_forTpl;
    const isRecentFinished = (!isFuture && !isLive && -diff_forTpl < threeD_forTpl);
    const scoreStr = m.score || '';
    const s1 = scoreStr && scoreStr.includes(':') ? escapeHTML(scoreStr.split(':')[0]) : '';
    const s2 = scoreStr && scoreStr.includes(':') ? escapeHTML(scoreStr.split(':')[1]) : '';
    // Date/time formatting for display lines
    const dtForDisp = startDt || (m.date_time ? parseCZDate(m.date_time) : null);
    const dd = dtForDisp ? String(dtForDisp.getDate()).padStart(1,'') : '';
    const mm = dtForDisp ? String(dtForDisp.getMonth()+1).padStart(1,'') : '';
    const yyyy = dtForDisp ? dtForDisp.getFullYear() : '';
    const HH = dtForDisp ? String(dtForDisp.getHours()).padStart(2,'0') : '';
    const MM = dtForDisp ? String(dtForDisp.getMinutes()).padStart(2,'0') : '';
    const dateOnly = dtForDisp ? `${dd}. ${mm}. ${yyyy}` : '';
    const timeToken = (m.date_time || '').split(' ')[1] || '';
    const timeOnly = dtForDisp ? `${HH}:${MM}` : '';
    const timeDisplay = dtForDisp ? ((timeToken === '' || timeToken === '00:00') ? 'Bude upřesněno' : timeOnly) : '';
    const venue = m.venue || '';
    const wrapperExtraClass = (isRecentFinished && s1 && s2) ? ' facr-finished' : '';

    const headerLabel = isLive ? 'Aktuální zápas' : (isFuture ? 'Nadcházející zápas' : (isRecentFinished ? 'Poslední zápas' : `Zápasy (${idx+1}/${items.length})`));

    root.innerHTML = `
      <div class="lte-football-upcoming${wrapperExtraClass}">
        <div class="facr-comp-title lte-football-date" style="text-align:center; margin-bottom:6px;">${compName}</div>
        <div class="facr-upcoming-header">
          <button id="facr-prev" class="facr-nav">◀</button>
          <span class="lte-header lte-header-upcoming">${headerLabel}</span>
          <button id="facr-next" class="facr-nav">▶</button>
        </div>
        <div class="lte-teams">
          <span class="lte-team-name lte-team-1 lte-header" title="${escapeHTML(m.home)}">
            <span class="lte-team-logo"><img decoding="async" src="${homeLogo}" alt="${escapeHTML(m.home)}"></span>${homeName}
            <span id="facr-inline-status" class="facr-inline-status" aria-live="polite"></span>
            ${isRecentFinished && s1 ? `<span class="lte-team-count-mob">${s1}</span>` : ''}
          </span>
          <span class="lte-team-count">
            <span id="facr-mid" style="font-size:32px; line-height:1; font-weight:700; display:inline-block; min-width:120px; text-align:center;">${midText}</span>
            ${isRecentFinished && s1 && s2 ? `<span class="facr-mob-center-score">${s1}<span>:</span>${s2}</span>` : ''}
          </span>
          <span class="lte-team-name lte-team-2 lte-header" title="${escapeHTML(m.away)}">
            ${isRecentFinished && s2 ? `<span class=\"lte-team-count-mob\">${s2}</span>` : ''}${awayName}<span class="lte-team-logo"><img decoding="async" src="${awayLogo}" alt="${escapeHTML(m.away)}"></span>
          </span>
        </div>
        <span class="lte-football-date" style="text-align:center;" title="${escapeHTML(m.date_time + (m.venue?`, ${m.venue}`:''))}">${escapeHTML(dateOnly + (venue?`, ${venue}`:''))}</span>
        ${timeDisplay ? `<span class="lte-football-time" style="display:block; text-align:center;">${escapeHTML(timeDisplay)}</span>` : ''}
        <span id="facr-countdown" class="lte-football-date" style="display:block; text-align:center;"></span>
        <br>
        <a class="lte-football-date" target="_blank" href="${facrLink}" style="text-align:center; background-color:#c42221; color:#ffffff; opacity:1;">Detail na FACR</a>
        <span style="display:block; margin-top:6px;"></span>
        <a class="lte-football-date" href="#tabulka" style="text-align:center; background-color:#ffffff43; color:#ffffff; opacity:1; width:49%; display:inline-block;">Tabulka bodů</a>
        <a class="lte-football-date" href="/zapasy/vsechny.html" style="text-align:center; background-color:#ffffff43; color:#ffffff; opacity:1; width:49%; display:inline-block;">Všechny zápasy</a>
      </div>`;

    const prev = document.getElementById('facr-prev');
    const next = document.getElementById('facr-next');
    if(prev) prev.onclick = ()=>{ state.matchIndex = (idx - 1 + items.length) % items.length; renderUpcoming(); };
    if(next) next.onclick = ()=>{ state.matchIndex = (idx + 1) % items.length; renderUpcoming(); };

    // setup countdown / status text
    const cd = document.getElementById('facr-countdown');
    const inlineStatus = document.getElementById('facr-inline-status');
    if(cd || inlineStatus){
      const now2 = new Date();
      const startMs = m.date_time ? parseCZDate(m.date_time).getTime() : 0;
      const diff = startMs - now2.getTime();
      const twoH = 2*60*60*1000;
      const threeD = 3*24*60*60*1000;
      let text = '';
      if(diff > 0){
        text = `Začátek za ${fmtCountdown(diff)}`;
      }else if(Math.abs(diff) <= twoH){
        text = 'Právě probíhá';
      }else if(-diff < threeD){
        text = (m.score ? `Výsledek: ${m.score}` : 'Ukončeno');
      }else{
        text = '';
      }
      if(cd) cd.textContent = text;
      if(inlineStatus) inlineStatus.textContent = text;
    }

    // Live countdown in the middle area when future and not today
    const midEl = document.getElementById('facr-mid');
    if(midEl){
      const startTime = m.date_time ? parseCZDate(m.date_time).getTime() : 0;
      function tick(){
        const now = Date.now();
        const diff = startTime - now;
        if(diff > 0){
          const longTxt = `Za ${fmtCountdownLong(diff)}`;
          midEl.textContent = longTxt;
          const inlineEl = document.getElementById('facr-inline-status');
          if(inlineEl) inlineEl.textContent = `Začátek za ${fmtCountdown(diff)}`;
          const cdEl = document.getElementById('facr-countdown');
          if(cdEl) cdEl.textContent = `Začátek za ${fmtCountdown(diff)}`;
        }else{
          // switch to score at/after kickoff
          const scoreTxt = m.score || '-';
          midEl.textContent = scoreTxt;
          const inlineEl = document.getElementById('facr-inline-status');
          if(inlineEl) inlineEl.textContent = (m.score ? `Výsledek: ${m.score}` : 'Ukončeno');
          const cdEl = document.getElementById('facr-countdown');
          if(cdEl) cdEl.textContent = (m.score ? `Výsledek: ${m.score}` : 'Ukončeno');
          if(state.upcomingTimerId){ clearInterval(state.upcomingTimerId); state.upcomingTimerId = null; }
        }
      }
      // Run live countdown for any future match (including today)
      if(startTime > Date.now()){
        tick();
        state.upcomingTimerId = setInterval(tick, 1000);
      } else {
        // Ensure inline status reflects finished state on initial render
        const inlineEl = document.getElementById('facr-inline-status');
        if(inlineEl) inlineEl.textContent = (m.score ? `Výsledek: ${m.score}` : 'Ukončeno');
      }
    }
  }

  function renderCompetitionTabs(){
    const tabs = document.getElementById('facr-comp-tabs');
    if(!tabs || !state.data) return;
    ensureFacrStyles();
    const comps = state.data.club_table.competitions || [];
    tabs.innerHTML = comps.map((c,i)=>
      `<button class="facr-tab ${i===state.compIndex?'active':''}" data-idx="${i}">${c.name || c.code || 'Soutěž'}</button>`
    ).join('');
    tabs.querySelectorAll('button').forEach(btn=>{
      btn.addEventListener('click', ()=>{
        state.compIndex = Number(btn.dataset.idx)||0;
        renderCompetitionTabs();
        renderTable();
      });
    });
  }

  function renderTable(){
    const tbody = document.getElementById('facr-table-body');
    const badge = document.getElementById('facr-comp-badge');
    if(!tbody || !state.data) return;
    const comps = state.data.club_table.competitions || [];
    if(comps.length === 0){ tbody.innerHTML = ''; return; }
    const comp = comps[Math.min(state.compIndex, comps.length-1)];
    if(badge){ badge.textContent = comp.name || comp.code || 'Soutěž'; }
    const rows = comp.table && comp.table.overall ? comp.table.overall : [];
    tbody.innerHTML = rows.map(r=>`
      <tr>
        <td class="lte-row"><span>${r.rank}</span></td>
        <td class="lte-club-logo"><img decoding="async" src="${r.team_logo_url || 'img/logo.png'}"></td>
        <td class="lte-name">${r.team}</td>
        <td class="lte-rate">${r.played}</td>
        <td class="lte-rate">${r.wins}</td>
        <td class="lte-rate">${r.draws}</td>
        <td class="lte-rate">${r.losses}</td>
        <td class="lte-rate">${r.score}</td>
        <td class="lte-summary">${r.points}</td>
      </tr>
    `).join('');
  }

  function renderAllMatches(){
    const container = document.getElementById('facr-all-matches');
    if(!container || !state.data) return;
    const comps = state.data.club_detail.competitions || [];
    const sections = [];
    for(const comp of comps){
      const matches = (comp.matches || [])
        .map(m=>({ m, dt: parseCZDate(m.date_time) }))
        .filter(x=>!!x.dt)
        .sort((a,b)=> b.dt - a.dt);
      if(matches.length === 0) continue;
      const compName = truncate(escapeHTML(comp.name || comp.code || 'Soutěž'), 40);
      const itemsHtml = matches.map(({m})=>{
        const homeLogo = m.home_logo_url || '../img/logo.png';
        const awayLogo = m.away_logo_url || '../img/logo.png';
        const facrLink = m.report_url || comp.matches_link || state.data.club_detail.url || '#';
        const score = m.score || '-';
        const GRID_MAX = 22;
        const home = truncate(escapeHTML(m.home), GRID_MAX);
        const away = truncate(escapeHTML(m.away), GRID_MAX);
        const dateVenue = truncate(escapeHTML(m.date_time + (m.venue?`, ${m.venue}`:'')), 36);
        const s1 = escapeHTML((score.split(':')[0]||'-'));
        const s2 = escapeHTML((score.split(':')[1]||'-'));
        return `
          <a href="${facrLink}" target="_blank" class="lte-item swiper-slide">
            <div class="lte-teams lte-match-time-public">
              <span class="lte-team-name lte-team-1 lte-header" title="${escapeHTML(m.home)}">
                <span class="lte-team-logo"><img src="${homeLogo}" alt="${escapeHTML(m.home)}"></span>${home}</span>
              <span class="lte-score-mob lte-score-1">${s1}</span>
              <span class="lte-team-count">
                <span class="lte-c lte-score-1">${s1}</span>
                <span class="lte-d">:</span>
                <span class="lte-c lte-score-4">${s2}</span>
              </span>
              <span class="lte-team-name lte-team-2 lte-header" title="${escapeHTML(m.away)}">${away}
                <span class="lte-team-logo"><img src="${awayLogo}" alt="${escapeHTML(m.away)}"></span>
              </span>
              <span class="lte-score-mob lte-score-4">${s2}</span>
            </div>
            <div class="lte-footer">
              <span class="lte-football-date" title="${escapeHTML(m.date_time + (m.venue?`, ${m.venue}`:''))}">${dateVenue}</span>
            </div>
          </a>`;
      }).join('');
      sections.push(`
        <div class="lte-section">
          <h3 class="lte-header" style="margin: 20px 0 10px;">${compName}</h3>
          <div class="lte-football-matches inner-page">${itemsHtml}</div>
        </div>
      `);
    }
    container.innerHTML = sections.join('');
  }

  function schedule(){
    if(state.intervalId) clearInterval(state.intervalId);
    const intervalMs = isWithinMatchWindow() ? 2*60*1000 : 30*60*1000;
    state.intervalId = setInterval(async ()=>{
      try{
        await fetchData();
        renderUpcoming();
        renderCompetitionTabs();
        renderTable();
        renderAllMatches();
        schedule(); // reevaluate interval if window changed
      }catch(e){ console.warn('refresh failed', e); }
    }, intervalMs);
  }

  async function init(){
    try{
      await fetchData();
      renderUpcoming();
      renderCompetitionTabs();
      renderTable();
      renderAllMatches();
      schedule();
    }catch(e){
      console.error('FACR init failed', e);
    }
  }

  document.addEventListener('DOMContentLoaded', init);
})();
