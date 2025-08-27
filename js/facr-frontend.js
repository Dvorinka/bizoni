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
      const dt = new Date(Date.UTC(year, month-1, day, hour, minute));
      // adjust to Prague timezone visually
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
      @media (max-width: 480px){
        #facr-mid{ font-size:28px !important; min-width:100px; }
        .facr-tab{ padding:4px 8px; font-size:14px; }
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
      // pick next upcoming if available; otherwise latest recent within window
      let pick = candidates.find(x=> x.dt >= now);
      if(!pick) pick = candidates[candidates.length - 1];
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

    // Prefer first future match; if none, show the latest recent within 3 days
    const now = new Date();
    let autoIdx = items.findIndex(it => it.dt >= now);
    if(autoIdx === -1) autoIdx = items.length - 1;
    const idx = Math.min(state.matchIndex || autoIdx, items.length-1);
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
    const midText = (!isToday && diffMsPre > 0) ? `Za ${fmtCountdownLong(diffMsPre)}` : (m.score || '-');

    root.innerHTML = `
      <div class="lte-football-upcoming">
        <div class="facr-comp-title lte-football-date" style="text-align:center; margin-bottom:6px;">${compName}</div>
        <div class="facr-upcoming-header">
          <button id="facr-prev" class="facr-nav">◀</button>
          <span class="lte-header lte-header-upcoming">Zápasy (${idx+1}/${items.length})</span>
          <button id="facr-next" class="facr-nav">▶</button>
        </div>
        <div class="lte-teams">
          <span class="lte-team-name lte-team-1 lte-header" title="${escapeHTML(m.home)}">
            <span class="lte-team-logo"><img decoding="async" src="${homeLogo}" alt="${escapeHTML(m.home)}"></span>${homeName}
          </span>
          <span class="lte-team-count"><span id="facr-mid" style="font-size:32px; line-height:1; font-weight:700; display:inline-block; min-width:120px; text-align:center;">${midText}</span></span>
          <span class="lte-team-name lte-team-2 lte-header" title="${escapeHTML(m.away)}">
            ${awayName}<span class="lte-team-logo"><img decoding="async" src="${awayLogo}" alt="${escapeHTML(m.away)}"></span>
          </span>
        </div>
        <span class="lte-football-date" style="text-align:center;" title="${escapeHTML(m.date_time + (m.venue?`, ${m.venue}`:''))}">${dateVenue}</span>
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
    if(cd){
      const now2 = new Date();
      const startMs = m.date_time ? parseCZDate(m.date_time).getTime() : 0;
      const diff = startMs - now2.getTime();
      const twoH = 2*60*60*1000;
      const threeD = 3*24*60*60*1000;
      if(diff > 0){
        // Always show a visible countdown, including on non-today future matches
        cd.textContent = `Začátek za ${fmtCountdown(diff)}`;
      }else if(Math.abs(diff) <= twoH){
        cd.textContent = 'Právě probíhá';
      }else if(-diff < threeD){
        cd.textContent = (m.score ? `Výsledek: ${m.score}` : 'Ukončeno');
      }else{
        cd.textContent = '';
      }
    }

    // Live countdown in the middle area when future and not today
    const midEl = document.getElementById('facr-mid');
    if(midEl){
      const startTime = m.date_time ? parseCZDate(m.date_time).getTime() : 0;
      function tick(){
        const now = Date.now();
        const diff = startTime - now;
        if(diff > 0){
          midEl.textContent = `Za ${fmtCountdownLong(diff)}`;
        }else{
          // switch to score at/after kickoff
          midEl.textContent = m.score || '-';
          if(state.upcomingTimerId){ clearInterval(state.upcomingTimerId); state.upcomingTimerId = null; }
        }
      }
      // Only run live countdown if the match is in the future and not today
      if(startTime > Date.now() && !isToday){
        tick();
        state.upcomingTimerId = setInterval(tick, 1000);
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
