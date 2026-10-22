// Validation harness: runs the real js/facr-frontend.js with a minimal DOM stub
// and prints the rendered order of matches in #facr-all-matches.
const fs = require('fs');
const vm = require('vm');

const JS_FILE = 'W:/Downloads/HTML_projekty/bizoni/js/facr-frontend.js';
const LOCAL_JSON = 'W:/Downloads/HTML_projekty/bizoni/data/club.json';
const LIVE_URL = 'https://bizoniuh.cz/data/club.json';

async function run(data, label) {
  const code = fs.readFileSync(JS_FILE, 'utf8');
  const container = { innerHTML: '' };
  let initFn = null;

  const sandbox = {
    console,
    setInterval: () => 0,
    clearInterval: () => {},
    Date,
    document: {
      addEventListener: (ev, cb) => { if (ev === 'DOMContentLoaded') initFn = cb; },
      getElementById: (id) => (id === 'facr-all-matches' ? container : null),
      createElement: () => ({ appendChild() {}, style: {} }),
      head: { appendChild() {} },
    },
    fetch: async () => ({ ok: true, json: async () => data }),
  };
  sandbox.window = sandbox;
  vm.createContext(sandbox);
  vm.runInContext(code, sandbox);

  await initFn();

  const dates = container.innerHTML.match(/\d{2}\.\d{2}\.\d{4} \d{2}:\d{2}/g) || [];
  console.log(`\n=== ${label} ===`);
  dates.forEach((d, i) => console.log(`${String(i + 1).padStart(2)}. ${d}`));
}

(async () => {
  const local = JSON.parse(fs.readFileSync(LOCAL_JSON, 'utf8'));
  await run(local, 'local data/club.json');
  const live = await (await fetch(LIVE_URL)).json();
  await run(live, 'live bizoniuh.cz/data/club.json');
})().catch(e => { console.error('FAILED:', e); process.exit(1); });
