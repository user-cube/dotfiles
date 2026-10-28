// Tests of colors.js, Hue without VS Code: node --test (CI runs them).
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const {
  DEFAULT_COLORS, OWNED, PARTS,
  hueOfHex, hueOf, autoHue, folderHue, hex, palette, ansiFor, globRegex, isDanger, nextCustomizations, readonlyGlobs, nextReadonly,
} = require('./colors');
const { panelState, applyMessage, panelHtml, WINDOW_PARTS } = require('./panel');
const { mainItems, colorItems, settingsItems, swatchSvg } = require('./search');

test('hueOfHex: the hue of a #rrggbb, any case; grey is 0; anything else undefined', () => {
  assert.equal(hueOfHex('#ff0000'), 0);
  assert.equal(hueOfHex('#00FF00'), 120);
  assert.equal(hueOfHex('#0000ff'), 240);
  assert.equal(hueOfHex('#ff00ff'), 300);
  assert.equal(hueOfHex('#808080'), 0);
  for (const bad of ['ff0000', '#f00', '#ff00000', '#gg0000', '']) assert.equal(hueOfHex(bad), undefined, bad);
});

test('hueOf: a color name, a hue (wrapped to 0-359) or a #rrggbb', () => {
  assert.equal(hueOf('blue', DEFAULT_COLORS), DEFAULT_COLORS.blue);
  assert.equal(hueOf(400, DEFAULT_COLORS), 40);
  assert.equal(hueOf(-30, DEFAULT_COLORS), 330);
  assert.equal(hueOf(' #00ff00 ', DEFAULT_COLORS), 120);
  assert.equal(hueOf('brand', { ...DEFAULT_COLORS, brand: '#0000ff' }), 240);
  assert.equal(hueOf('nope', DEFAULT_COLORS), undefined);
  assert.equal(hueOf(undefined, DEFAULT_COLORS), undefined);
});

test('autoHue: the same name always gets the same named color', () => {
  const hues = new Set(Object.values(DEFAULT_COLORS));
  for (const name of ['homelab-ansible', 'dotfiles', 'x']) {
    const h = autoHue(name, DEFAULT_COLORS);
    assert.ok(hues.has(h), `${name} -> ${h}`);
    assert.equal(autoHue(name, DEFAULT_COLORS), h);
  }
  assert.equal(autoHue('dotfiles', {}), undefined);
});

test('folderHue: its own color, else one from its name with hue.auto, none with Hue off', () => {
  const colors = DEFAULT_COLORS;
  assert.equal(folderHue('infra', { folders: { infra: 'blue' }, colors }), DEFAULT_COLORS.blue);
  assert.equal(folderHue('infra', { folders: {}, colors }), autoHue('infra', colors));
  assert.equal(folderHue('infra', { folders: {}, colors, auto: false }), undefined);
  assert.equal(folderHue('infra', { folders: { infra: 'blue' }, colors, auto: false }), DEFAULT_COLORS.blue);
  assert.equal(folderHue('infra', { folders: { infra: 'nope' }, colors }), undefined, 'a bad color is not replaced by the automatic one');
  assert.equal(folderHue('infra', { folders: { infra: 'blue' }, colors, enabled: false }), undefined);
});

test('hex: HSL to #rrggbb, saturation capped at 100', () => {
  assert.equal(hex(0, 100, 50), '#ff0000');
  assert.equal(hex(120, 100, 50), '#00ff00');
  assert.equal(hex(240, 100, 50), '#0000ff');
  assert.equal(hex(0, 0, 100), '#ffffff');
  assert.equal(hex(200, 150, 40), hex(200, 100, 40));
});

test('palette: the same keys in dark and light, all of them owned', () => {
  const dark = palette(212, false, 1), light = palette(212, true, 1);
  assert.deepEqual(Object.keys(dark), PARTS);
  for (const part of PARTS) assert.deepEqual(Object.keys(light[part]), Object.keys(dark[part]));
  const keys = Object.values(dark).flatMap((p) => Object.keys(p));
  assert.deepEqual([...keys].sort(), [...OWNED].sort());
  assert.notEqual(dark.titleBar['titleBar.activeBackground'], light.titleBar['titleBar.activeBackground']);
});

test('ansiFor: the nearest ANSI color of a hue, red on both ends of the circle', () => {
  assert.equal(ansiFor(0), 'terminal.ansiRed');
  assert.equal(ansiFor(350), 'terminal.ansiRed');
  assert.equal(ansiFor(46), 'terminal.ansiYellow');
  assert.equal(ansiFor(140), 'terminal.ansiGreen');
  assert.equal(ansiFor(176), 'terminal.ansiCyan');
  assert.equal(ansiFor(212), 'terminal.ansiBlue');
  assert.equal(ansiFor(268), 'terminal.ansiMagenta');
  assert.equal(ansiFor(335), 'terminal.ansiBrightMagenta');
});

test('globRegex: ** crosses directories, * and ? do not, the rest is literal', () => {
  assert.ok(globRegex('envs/**/prod').test('envs/prod'));
  assert.ok(globRegex('envs/**/prod').test('envs/eu/west/prod'));
  assert.ok(globRegex('envs/*/prod').test('envs/eu/prod'));
  assert.ok(!globRegex('envs/*/prod').test('envs/eu/west/prod'));
  assert.ok(globRegex('prod-?').test('prod-1'));
  assert.ok(!globRegex('prod-?').test('prod-12'));
  assert.ok(!globRegex('a.b').test('axb'));
});

test('isDanger: a whole folder, or a path inside it and what is under it', () => {
  const entries = ['infra', 'unifi-home-network/terrakube', 'k8s/clusters/*/prod/'];
  assert.equal(isDanger(entries, 'infra', undefined), 'infra');
  assert.equal(isDanger(entries, 'infra', 'any/file.tf'), 'infra');
  assert.equal(isDanger(entries, 'unifi-home-network', 'terrakube'), 'unifi-home-network/terrakube');
  assert.equal(isDanger(entries, 'unifi-home-network', 'terrakube/main.tf'), 'unifi-home-network/terrakube');
  assert.equal(isDanger(entries, 'unifi-home-network', 'terrakube-modules/main.tf'), undefined);
  assert.equal(isDanger(entries, 'unifi-home-network', 'other/main.tf'), undefined);
  assert.equal(isDanger(entries, 'unifi-home-network', undefined), undefined); // no file open
  assert.equal(isDanger(entries, 'k8s', 'clusters/eu/prod/app.yaml'), 'k8s/clusters/*/prod/');
  assert.equal(isDanger(entries, 'k8s', 'clusters/eu/dev/app.yaml'), undefined);
  assert.equal(isDanger(entries, 'terrakube', 'x'), undefined); // a path entry's folder is its first part
  assert.equal(isDanger([], 'infra', 'x'), undefined);
});

test('nextCustomizations: only Hue\'s keys change, the others are left alone', () => {
  const current = { 'editor.background': '#000000', 'titleBar.activeBackground': '#123456' };
  const next = nextCustomizations(current, { hue: 212, light: false });
  assert.equal(next['editor.background'], '#000000');
  assert.equal(next['titleBar.activeBackground'], palette(212, false, 1).titleBar['titleBar.activeBackground']);
  assert.equal(current['titleBar.activeBackground'], '#123456', 'current is not changed');
  assert.deepEqual(nextCustomizations(current, { hue: undefined, light: false }), { 'editor.background': '#000000' });
});

test('nextCustomizations: only the parts chosen, tabs always', () => {
  const next = nextCustomizations({}, { hue: 140, light: true, parts: ['statusBar'] });
  const p = palette(140, true, 1);
  assert.deepEqual(next, { ...p.statusBar, ...p.tabs });
});

test('nextCustomizations: strength scales the palette', () => {
  const subtle = nextCustomizations({}, { hue: 4, light: false, k: 0.55 });
  assert.deepEqual(subtle, Object.assign({}, ...Object.values(palette(4, false, 0.55))));
});

test('nextCustomizations: warnings win over the folder color, with or without one', () => {
  for (const hue of [212, undefined]) {
    const dark = nextCustomizations({}, { hue, light: false, onProtected: true, danger: true });
    assert.equal(dark['statusBar.background'], '#8a5a00');
    assert.equal(dark['titleBar.activeBackground'], '#8a1c1c');
    assert.equal(dark['titleBar.inactiveBackground'], '#4a1212');
    const light = nextCustomizations({}, { hue, light: true, onProtected: true, danger: true });
    assert.equal(light['statusBar.background'], '#ffd58a');
    assert.equal(light['titleBar.activeBackground'], '#f5b5b5');
    assert.equal(light['titleBar.activeForeground'], '#1f2328');
  }
});

test('install.sh packages every local module extension.js, colors.js and panel.js require', () => {
  // the cp into the .vsix, and the sha that says when to reinstall
  const install = fs.readFileSync(path.join(__dirname, 'install.sh'), 'utf8');
  const copied = install.split('\n').find((l) => l.startsWith('cp ') && l.includes('/vsix/extension/'));
  const hashed = install.split('\n').find((l) => l.startsWith('sha=$(cat '));
  assert.ok(copied && hashed, 'install.sh still has its cp and sha lines');
  for (const file of ['extension.js', 'colors.js', 'panel.js']) {
    const src = fs.readFileSync(path.join(__dirname, file), 'utf8');
    for (const [, mod] of src.matchAll(/require\('\.\/([^']+)'\)/g)) {
      const name = mod.endsWith('.js') ? mod : `${mod}.js`;
      assert.ok(copied.includes(`"$dir/${name}"`), `${file} requires ./${mod}, which install.sh doesn't put in the .vsix`);
      assert.ok(hashed.includes(`"$dir/${name}"`), `${file} requires ./${mod}, whose changes install.sh doesn't notice`);
    }
  }
});

test('readonlyGlobs: absolute globs, a whole folder or a path and what is under it', () => {
  const folders = [{ name: 'infra', fsPath: '/w/infra' }, { name: 'app', fsPath: '/w/app/' }];
  assert.deepEqual(readonlyGlobs(['infra'], folders), ['/w/infra/**']);
  assert.deepEqual(readonlyGlobs(['app/prod/'], folders), ['/w/app/prod', '/w/app/prod/**']);
  assert.deepEqual(readonlyGlobs(['app/clusters/*/prod'], folders), ['/w/app/clusters/*/prod', '/w/app/clusters/*/prod/**']);
  assert.deepEqual(readonlyGlobs(['gone', 'gone/x'], folders), [], 'a folder not in the workspace gives none');
  assert.deepEqual(readonlyGlobs(['infra', 'infra'], folders), ['/w/infra/**']);
  assert.deepEqual(readonlyGlobs([], folders), []);
});

test('nextReadonly: Hue\'s old globs out, the new in, the user\'s left alone', () => {
  const user = { '/w/vendor/**': true };
  let r = nextReadonly(user, [], ['/w/infra/**']);
  assert.deepEqual(r, { next: { '/w/vendor/**': true, '/w/infra/**': true }, owned: ['/w/infra/**'] });
  r = nextReadonly(r.next, r.owned, ['/w/app/prod', '/w/app/prod/**']);
  assert.deepEqual(r.next, { '/w/vendor/**': true, '/w/app/prod': true, '/w/app/prod/**': true });
  assert.deepEqual(nextReadonly(r.next, r.owned, []), { next: user, owned: [] });
  // the user's own glob is not taken over, so turning Hue's off keeps it
  r = nextReadonly(user, [], ['/w/vendor/**']);
  assert.deepEqual(r, { next: user, owned: [] });
  assert.deepEqual(nextReadonly({ '/w/infra/**': false }, [], ['/w/infra/**']).next, { '/w/infra/**': false }, 'a user\'s false wins');
});

// panel.js: the page's state and the settings its messages change
const PAGE_SETTINGS = { folders: { infra: 'blue', app: '#00ff00' }, danger: ['infra', 'app/prod', 'app/eu/*/db', 'gone/x'] };

test('panelState: each folder\'s color, preview and PROD places, and the settings with their defaults', () => {
  const s = panelState(['infra', 'app', 'docs'], PAGE_SETTINGS, DEFAULT_COLORS);
  assert.deepEqual(s.folders.map((f) => [f.name, f.color, f.prod, f.prodPaths]), [
    ['infra', 'blue', true, []],
    ['app', '#00ff00', false, ['prod', 'eu/*/db']],
    ['docs', null, false, []],
  ]);
  assert.equal(s.folders[0].preview, palette(DEFAULT_COLORS.blue, false, 1).titleBar['titleBar.activeBackground']);
  assert.equal(s.folders[2].preview, palette(autoHue('docs', DEFAULT_COLORS), false, 1).titleBar['titleBar.activeBackground'], 'automatic');
  assert.equal(panelState(['docs'], { auto: false }, DEFAULT_COLORS).folders[0].preview, undefined, 'no color');
  assert.deepEqual(s.settings, { enabled: true, auto: true, dangerReadonly: false, strength: 'normal', parts: WINDOW_PARTS, protectedBranches: ['main', 'master'] });
  assert.deepEqual(s.colors.map((c) => c.name), Object.keys(DEFAULT_COLORS));
  assert.ok(!WINDOW_PARTS.includes('tabs'), 'tabs are always tinted, not a choice');
  assert.equal(s.folders[0].swatch, palette(DEFAULT_COLORS.blue, false, 1).activityBar['activityBar.activeBorder'], 'the bright color for the swatch');
  assert.ok(s.colors.every((c) => /^#[0-9a-f]{6}$/.test(c.swatch)));
  assert.equal(s.prodBar, '#8a1c1c');
  assert.equal(panelState(['infra'], {}, DEFAULT_COLORS, true).prodBar, '#f5b5b5', 'pale in a light theme');
});

test('applyMessage: colors, PROD and settings, each to the setting it changes', () => {
  const folders = ['infra', 'app'];
  const apply = (m) => applyMessage(m, PAGE_SETTINGS, folders, DEFAULT_COLORS);
  assert.deepEqual(apply({ type: 'setColor', folder: 'infra', value: 'teal' }), { key: 'folders', value: { infra: 'teal', app: '#00ff00' } });
  assert.deepEqual(apply({ type: 'setColor', folder: 'app', value: null }), { key: 'folders', value: { infra: 'blue' } }, 'Automatic');
  assert.deepEqual(apply({ type: 'setColor', folder: 'infra', value: '#123ABC' }).value.infra, '#123ABC');
  assert.deepEqual(apply({ type: 'setProd', folder: 'infra', on: false }).value, ['app/prod', 'app/eu/*/db', 'gone/x'], 'the paths stay');
  assert.deepEqual(apply({ type: 'setProd', folder: 'app', on: true }).value, [...PAGE_SETTINGS.danger, 'app']);
  assert.deepEqual(apply({ type: 'addProdPath', folder: 'app', rel: 'k8s/prod/' }).value, [...PAGE_SETTINGS.danger, 'app/k8s/prod']);
  assert.deepEqual(apply({ type: 'addProdPath', folder: 'app', rel: 'prod' }).value, PAGE_SETTINGS.danger, 'no duplicate');
  assert.deepEqual(apply({ type: 'removeProdPath', folder: 'app', rel: 'eu/*/db' }).value, ['infra', 'app/prod', 'gone/x']);
  assert.deepEqual(apply({ type: 'setSetting', key: 'dangerReadonly', value: true }), { key: 'dangerReadonly', value: true });
  assert.deepEqual(apply({ type: 'setSetting', key: 'strength', value: 'strong' }), { key: 'strength', value: 'strong' });
  assert.deepEqual(apply({ type: 'setSetting', key: 'parts', value: ['statusBar', 'titleBar'] }).value, ['titleBar', 'statusBar'], 'in their order');
  assert.deepEqual(apply({ type: 'setSetting', key: 'protectedBranches', value: [' main', 'release ', '', 'main'] }).value, ['main', 'release']);
  assert.deepEqual(PAGE_SETTINGS.folders, { infra: 'blue', app: '#00ff00' }, 'the settings given are not changed');
});

test('applyMessage: anything the page sends that Hue doesn\'t know changes nothing', () => {
  const folders = ['infra', 'app'];
  const bad = [
    null, 'setColor', {}, { type: 'nope' },
    { type: 'setColor', folder: 'gone', value: 'blue' },
    { type: 'setColor', folder: 'infra', value: 'chartreuse' },
    { type: 'setColor', folder: 'infra', value: '#12345' },
    { type: 'setColor', folder: '__proto__', value: 'blue' },
    { type: 'setProd', folder: 'infra', on: 'yes' },
    { type: 'addProdPath', folder: 'app', rel: '../etc' },
    { type: 'addProdPath', folder: 'app', rel: 'a/../../b' },
    { type: 'addProdPath', folder: 'app', rel: '' },
    { type: 'addProdPath', folder: 'app', rel: 'a//b' },
    { type: 'removeProdPath', folder: 'app', rel: 'not-there' },
    { type: 'setSetting', key: 'folders', value: {} },
    { type: 'setSetting', key: 'danger', value: [] },
    { type: 'setSetting', key: 'enabled', value: 'false' },
    { type: 'setSetting', key: 'strength', value: 'max' },
    { type: 'setSetting', key: 'parts', value: ['tabs'] },
    { type: 'setSetting', key: 'parts', value: 'titleBar' },
    { type: 'setSetting', key: 'protectedBranches', value: [1] },
  ];
  for (const m of bad) assert.equal(applyMessage(m, PAGE_SETTINGS, folders, DEFAULT_COLORS), undefined, JSON.stringify(m));
});

test('panelHtml: no remote content, script and style by nonce only, and a script that compiles', () => {
  const html = panelHtml('n0nce');
  const csp = /http-equiv="Content-Security-Policy" content="([^"]+)"/.exec(html)[1];
  assert.equal(csp, "default-src 'none'; style-src 'nonce-n0nce'; script-src 'nonce-n0nce';");
  assert.ok(!/https?:\/\//.test(html), 'nothing loaded from the network');
  assert.equal((html.match(/<script nonce="n0nce">/g) || []).length, 1);
  assert.equal((html.match(/<script/g) || []).length, 1, 'no script without the nonce');
  assert.ok(!html.includes('innerHTML'), 'the page draws with DOM calls, never HTML strings');
  const script = html.split('<script nonce="n0nce">')[1].split('</script>')[0];
  new (require('node:vm').Script)(script);
});

const SEARCH = { folders: ['infra', 'app', 'docs'], active: 'infra', colors: DEFAULT_COLORS,
  settings: { folders: { infra: 'blue', app: '#00ff00' }, danger: ['app', 'docs/prod', 'docs/eu'], dangerReadonly: true, strength: 'subtle' } };
const lines = (items) => items.filter((i) => !i.separator);
const accepted = (items, settings) => lines(items).filter((i) => i.message)
  .every((i) => applyMessage(i.message, settings, SEARCH.folders, DEFAULT_COLORS) !== undefined);

test('mainItems: a line per folder, its color as the icon, ⚠ for PROD, with buttons; then Hue\'s own', () => {
  const items = mainItems(SEARCH);
  assert.deepEqual(items.filter((i) => i.separator).map((i) => i.separator), ['Folders', 'Hue']);
  const [infra, app, docs] = lines(items);
  assert.deepEqual([infra.label, infra.description, infra.run], ['infra', 'blue · you are here', 'goTo']);
  assert.deepEqual(infra.icon, { swatch: palette(DEFAULT_COLORS.blue, false, 1).activityBar['activityBar.activeBorder'] });
  assert.deepEqual([app.description, app.icon], ['PROD', { prod: true }]);
  assert.equal(docs.description, 'auto · 2 PROD paths');
  assert.deepEqual(infra.buttons.map((b) => b.id), ['terminal', 'color', 'prod']);
  assert.equal(infra.buttons[2].tooltip, 'Mark infra as PROD');
  assert.equal(app.buttons[2].tooltip, 'Unmark app as PROD');
  assert.deepEqual(lines(items).slice(3).map((i) => i.run), ['colorMenu', 'toggleDanger', 'settingsMenu', 'open']);
  assert.equal(items.find((i) => i.run === 'settingsMenu').description, 'on · subtle · PROD read-only');
  const away = mainItems({ ...SEARCH, active: undefined, settings: { auto: false } });
  assert.ok(!away.some((i) => i.run === 'colorMenu'), 'no Color of… when you are in no folder');
  assert.deepEqual([lines(away)[0].description, lines(away)[0].icon], ['no color', 'circle-large-outline']);
  assert.equal(lines(away).length, 6, 'three folders and three entries: short');
});

test('colorItems: the colors as dots, Custom and Automatic, each one a setting applyMessage takes', () => {
  const infra = colorItems({ ...SEARCH, folder: 'infra' });
  assert.deepEqual(lines(infra).map((i) => i.label), [...Object.keys(DEFAULT_COLORS), 'Custom…', 'Automatic']);
  assert.equal(infra.find((i) => i.label === 'blue').description, 'current');
  assert.ok(lines(infra).slice(0, 8).every((i) => /^#[0-9a-f]{6}$/.test(i.icon.swatch)));
  assert.ok(accepted(infra, SEARCH.settings));
  const app = colorItems({ ...SEARCH, folder: 'app' });
  assert.deepEqual(app.find((i) => i.run === 'customColor'),
    { label: 'Custom #00ff00', description: 'current', icon: { swatch: '#00ff00' }, run: 'customColor', folder: 'app' });
  assert.equal(colorItems({ ...SEARCH, folder: 'docs' }).find((i) => i.label === 'Automatic').description, 'current');
});

test('settingsItems: switches with a check when on, the strength, the parts, the branches', () => {
  const items = settingsItems(SEARCH);
  const line = (label) => items.find((i) => i.label === label);
  assert.deepEqual([line('Hue in this workspace').icon, line('Hue in this workspace').message.value], ['check', false]);
  assert.deepEqual([line('PROD opens read-only').icon, line('PROD opens read-only').message.value], ['check', false]);
  assert.deepEqual([line('Automatic colors').icon, line('Automatic colors').message.value], ['check', false]);
  assert.equal(line('subtle').icon, 'circle-filled');
  assert.equal(line('normal').icon, 'circle-outline');
  assert.deepEqual(line('sideBar').message.value, WINDOW_PARTS.filter((p) => p !== 'sideBar'));
  assert.equal(line('main, master').run, 'branches');
  assert.ok(accepted(items, SEARCH.settings));
  const off = settingsItems({ settings: { parts: ['titleBar'], protectedBranches: [], dangerReadonly: false } });
  assert.deepEqual(off.find((i) => i.label === 'sideBar').message.value, ['titleBar', 'sideBar']);
  assert.equal(off.find((i) => i.label === 'PROD opens read-only').icon, 'circle-large-outline');
  assert.ok(off.some((i) => i.label === 'none' && i.run === 'branches'));
});

test('swatchSvg: a dot in the color, and nothing but a #rrggbb gets in', () => {
  assert.equal(swatchSvg('#12ab9F'),
    '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="6" fill="#12ab9F"/></svg>');
  for (const bad of ['red', '#fff', '#12345g', '#123456"/><script>']) assert.throws(() => swatchSvg(bad), /not a #rrggbb/, bad);
});

// extension.js with a fake `vscode`: folders infra and app, an editor whose
// file can be switched, the events and commands it registers, what it writes.
// git: what vscode.extensions.getExtension('vscode.git') returns
const settle = () => new Promise((resolve) => setImmediate(resolve));

async function loadExtension({ hue = {}, file = '/w/infra/main.tf', git, readonly } = {}) {
  const Module = require('node:module');
  const folders = ['infra', 'app'].map((name) => ({ name, uri: { fsPath: `/w/${name}`, scheme: 'file' } }));
  const ext = { inputs: [], quickPicks: [], informed: [], hueWrites: [], posted: [], picked: undefined, warned: [], updates: [], fileWrites: [], commands: {}, events: {}, terminals: [], folderWrites: 0, prod: false, executed: [], asked: [], answer: undefined };
  const editor = (fsPath) => ({ document: { uri: { scheme: 'file', fsPath } } });
  const on = (name) => (cb) => { ext.events[name] = cb; return { dispose() {} }; };
  const settings = { auto: true, protectedBranches: [], ...hue };
  // The workspace's settings, by section (what inspect().workspaceValue reads)
  const config = { workbench: { colorCustomizations: { 'editor.background': '#000000' } }, files: { readonlyInclude: readonly } };
  const fake = {
    window: {
      activeTextEditor: editor(file),
      activeColorTheme: { kind: 2 },
      createStatusBarItem: () => ({ show() { ext.prod = true; }, hide() { ext.prod = false; }, dispose() {} }),
      createTerminal: (options) => { ext.terminals.push(options); return { show() {} }; },
      showWarningMessage: async (message, options) => {
        if (options && options.modal) { ext.asked.push(message); return ext.answer; }
        ext.warned.push(message);
        return undefined;
      },
      showOpenDialog: async () => ext.picked,
      // choose: picks from the list (by label), or cancels when it returns undefined
      showQuickPick: async (items) => { ext.quickPicks.push(items); return ext.choose ? ext.choose(items) : undefined; },
      showInformationMessage: (message) => { ext.informed.push(message); },
      showInputBox: async () => ext.inputs.shift(),
      createQuickPick: () => {
        const qp = { items: [], buttons: [], selectedItems: [], value: '', on: {}, visible: false,
          onDidAccept: (cb) => { qp.on.accept = cb; }, onDidTriggerButton: (cb) => { qp.on.button = cb; },
          onDidTriggerItemButton: (cb) => { qp.on.itemButton = cb; }, onDidHide: (cb) => { qp.on.hide = cb; },
          show() { qp.visible = true; }, hide() { if (qp.visible) { qp.visible = false; qp.on.hide(); } }, dispose() { qp.disposed = true; } };
        ext.qp = qp;
        ext.quickPicks.push(qp);
        return qp;
      },
      createWebviewPanel: (id, title, column, options) => {
        const webview = {
          html: '', options,
          postMessage: async (m) => { ext.posted.push(m); return true; },
          onDidReceiveMessage: (cb) => { ext.fromPage = async (m) => { await cb(m); await settle(); }; return { dispose() {} }; },
        };
        ext.panels = (ext.panels || 0) + 1;
        return { webview, reveal() { ext.revealed = (ext.revealed || 0) + 1; }, onDidDispose: (cb) => { ext.closePanel = cb; } };
      },
      onDidChangeActiveTextEditor: on('editor'), onDidChangeActiveColorTheme: on('theme'), onDidChangeActiveTerminal: on('terminal'),
    },
    workspace: {
      workspaceFile: { fsPath: '/w/x.code-workspace' },
      workspaceFolders: folders,
      getWorkspaceFolder: (uri) => folders.find((f) => uri.fsPath === f.uri.fsPath || uri.fsPath.startsWith(`${f.uri.fsPath}/`)),
      getConfiguration: (section) => section === 'hue'
        ? {
            get: (key) => settings[key],
            // VS Code fires the config event after a write
            update: async (key, value) => { settings[key] = value; ext.hueWrites.push([key, value]); ext.events.config({ affectsConfiguration: (s) => s === 'hue' }); },
          }
        : {
            inspect: (key) => ({ workspaceValue: (config[section] || {})[key] }),
            update: async (key, value) => {
              (config[section] = config[section] || {})[key] = value;
              (section === 'workbench' ? ext.updates : ext.fileWrites).push(value);
            },
          },
      onDidChangeWorkspaceFolders: on('folders'), onDidChangeConfiguration: on('config'),
    },
    commands: {
      registerCommand: (name, fn) => { ext.commands[name] = fn; return { dispose() {} }; },
      executeCommand: async (...args) => { ext.executed.push(args); },
    },
    Uri: { file: (fsPath) => ({ scheme: 'file', fsPath }) },
    extensions: { getExtension: () => git },
    ViewColumn: { Active: -1 }, QuickPickItemKind: { Separator: -1 }, QuickInputButtons: { Back: { id: 'back' } }, StatusBarAlignment: { Left: 1 }, ColorThemeKind: { Light: 1, Dark: 2, HighContrast: 3, HighContrastLight: 4 },
    ConfigurationTarget: { Workspace: 2 },
    ThemeColor: class { constructor(id) { this.id = id; } },
    ThemeIcon: class { constructor(id, color) { this.id = id; this.color = color; } },
  };
  const state = new Map();
  const context = {
    globalStorageUri: { fsPath: fs.mkdtempSync(path.join(require('node:os').tmpdir(), 'hue-test-')) },
    subscriptions: [],
    workspaceState: { get: (k) => state.get(k), update: (k, v) => { if (k === 'folder') ext.folderWrites++; state.set(k, v); } },
  };
  const load = Module._load;
  Module._load = function (request, ...rest) { return request === 'vscode' ? fake : load.call(this, request, ...rest); };
  try {
    delete require.cache[require.resolve('./extension')];
    require('./extension').activate(context);
  } finally {
    Module._load = load;
  }
  await settle();
  ext.open = async (fsPath) => { fake.window.activeTextEditor = editor(fsPath); ext.events.editor(); await settle(); };
  ext.last = () => ext.updates[ext.updates.length - 1];
  // A terminal becomes the active one: opened in cwd, or now in shellCwd (shell integration)
  ext.focusTerminal = async ({ cwd, shellCwd } = {}) => {
    ext.events.terminal({
      creationOptions: { cwd: cwd && { scheme: 'file', fsPath: cwd } },
      shellIntegration: shellCwd ? { cwd: { scheme: 'file', fsPath: shellCwd } } : undefined,
    });
    await settle();
  };
  ext.readonly = () => config.files.readonlyInclude;
  ext.workspace = fake.workspace;
  // Hue: Search, driven like a person would: open it, pick a line (by label),
  // press a line's button, or Back
  ext.search = async () => { await ext.commands['hue.search'](); await settle(); };
  ext.item = (label) => ext.qp.items.find((i) => i.label === label);
  ext.accept = async (label) => {
    const item = ext.item(label);
    assert.ok(item, `no line ${label} in: ${ext.qp.items.map((i) => i.label).join(' | ')}`);
    ext.qp.selectedItems = [item];
    ext.qp.on.accept();
    await settle(); await settle();
  };
  ext.press = async (label, id) => {
    const item = ext.item(label);
    await ext.qp.on.itemButton({ item, button: item.buttons.find((b) => b.id === id) });
    await settle();
  };
  ext.back = async () => { ext.qp.on.button(ext.qp.buttons[0]); await settle(); };
  ext.state = () => [...ext.posted].reverse().find((m) => m.type === 'state')?.state;
  // Change Hue's settings the way the settings UI does: then the config event
  ext.setHue = async (changes) => {
    Object.assign(settings, changes);
    ext.events.config({ affectsConfiguration: (section) => section === 'hue' });
    await settle();
  };
  ext.saves = () => ext.executed.filter(([c]) => c === 'workbench.action.files.save').length;
  ext.context = (key) => [...ext.executed].reverse().find(([c, k]) => c === 'setContext' && k === key)?.[2];
  return ext;
}

// What the workspace should hold for these options (dark theme)
const expected = (options) => nextCustomizations({ 'editor.background': '#000000' }, { light: false, ...options });

test('extension.js: activate, then the active folder\'s colors land in the workspace', async () => {
  const ext = await loadExtension({
    hue: { folders: { infra: 'blue' }, danger: ['infra/prod'], parts: ['statusBar'] },
    file: '/w/infra/prod/main.tf',
  });
  assert.deepEqual(Object.keys(ext.commands).sort(),
    ['hue.clear', 'hue.goToFolder', 'hue.open', 'hue.openTerminal', 'hue.refresh', 'hue.save', 'hue.search', 'hue.setColor', 'hue.toggleDanger']);
  assert.equal(ext.updates.length, 1);
  assert.deepEqual(ext.last(), expected({ hue: DEFAULT_COLORS.blue, parts: ['statusBar'], danger: true }));
  assert.ok(ext.prod, 'the PROD marker is shown');
});

test('extension.js: switching folders while the git API loads ends on the new folder\'s colors', async () => {
  let ready;
  const gate = new Promise((resolve) => { ready = resolve; });
  const api = { repositories: [], onDidOpenRepository: () => ({ dispose() {} }), getRepository: () => undefined };
  const git = { isActive: false, activate: () => gate.then(() => ({ getAPI: () => api })) };
  const ext = await loadExtension({ hue: { folders: { infra: 'blue', app: 'green' }, protectedBranches: ['main'] }, git });
  await ext.open('/w/app/main.go'); // the first apply (infra) still waits for git
  ready();
  await settle(); await settle();
  assert.deepEqual(ext.last(), expected({ hue: DEFAULT_COLORS.green }));
});

test('extension.js: a refresh with nothing changed writes nothing', async () => {
  const ext = await loadExtension({ hue: { folders: { infra: 'blue' } } });
  for (let i = 0; i < 3; i++) { await ext.commands['hue.refresh'](); }
  await ext.open('/w/infra/other.tf');
  assert.equal(ext.updates.length, 1, 'colorCustomizations written once');
  assert.equal(ext.folderWrites, 1, 'the active folder saved once');
  await ext.open('/w/app/main.go');
  assert.equal(ext.folderWrites, 2, 'and again when it changes');
});

test('extension.js: ⌘⌥T\'s terminal follows hue.auto and hue.enabled like the window', async () => {
  let ext = await loadExtension({ hue: { auto: false } });
  await ext.commands['hue.openTerminal']();
  assert.equal(ext.terminals[0].color, undefined, 'no folder color with hue.auto off');

  ext = await loadExtension({ hue: { folders: { infra: 'blue' } } });
  await ext.commands['hue.openTerminal']();
  assert.equal(ext.terminals[0].color.id, 'terminal.ansiBlue');

  ext = await loadExtension({ hue: { enabled: false, folders: { infra: 'blue' }, danger: ['infra'] } });
  await ext.commands['hue.openTerminal']();
  assert.equal(ext.terminals[0].name, 'infra', 'no ⚠ with Hue off');
  assert.equal(ext.terminals[0].color, undefined, 'no color with Hue off');
});

test('extension.js: the window follows the active terminal\'s folder, until the next editor change', async () => {
  const ext = await loadExtension({ hue: { folders: { infra: 'blue', app: 'green' }, danger: ['app/prod'] } });
  assert.deepEqual(ext.last(), expected({ hue: DEFAULT_COLORS.blue }));
  await ext.focusTerminal({ cwd: '/w/app' });
  assert.deepEqual(ext.last(), expected({ hue: DEFAULT_COLORS.green }));
  assert.ok(!ext.prod);
  await ext.focusTerminal({ cwd: '/w/app', shellCwd: '/w/app/prod/eu' }); // cd'd into a danger place
  assert.deepEqual(ext.last(), expected({ hue: DEFAULT_COLORS.green, danger: true }));
  assert.ok(ext.prod, 'PROD from the terminal\'s directory');
  const writes = ext.updates.length;
  await ext.focusTerminal({ cwd: '/tmp' }); // no folder: the window stays
  assert.equal(ext.updates.length, writes);
  await ext.open('/w/infra/main.tf');
  assert.deepEqual(ext.last(), expected({ hue: DEFAULT_COLORS.blue }));
  assert.ok(!ext.prod);
});

test('extension.js: ⌘S in a danger place asks first, once per place', async () => {
  const ext = await loadExtension({ hue: { danger: ['infra/prod', 'app'] }, file: '/w/infra/prod/main.tf' });
  assert.equal(ext.context('hue.dangerFile'), true, 'the ⌘S keybinding is on');
  ext.answer = undefined; // Cancel
  await ext.commands['hue.save']();
  assert.equal(ext.asked.length, 1);
  assert.equal(ext.saves(), 0, 'cancelled: not saved');
  ext.answer = 'Save';
  await ext.commands['hue.save']();
  assert.equal(ext.saves(), 1);
  await ext.commands['hue.save']();
  assert.equal(ext.asked.length, 2, 'not asked again for infra/prod');
  assert.equal(ext.saves(), 2);
  await ext.open('/w/app/main.go'); // another place: asks again
  await ext.commands['hue.save']();
  assert.equal(ext.asked.length, 3);
  await ext.open('/w/infra/main.tf'); // not a danger place
  assert.equal(ext.context('hue.dangerFile'), false, 'the ⌘S keybinding is off');
  await ext.commands['hue.save']();
  assert.equal(ext.asked.length, 3, 'saved without asking');
  assert.equal(ext.saves(), 4);
});

test('extension.js: with Hue off, ⌘S never asks', async () => {
  const ext = await loadExtension({ hue: { enabled: false, danger: ['infra'] } });
  assert.equal(ext.context('hue.dangerFile'), false);
  await ext.commands['hue.save']();
  assert.equal(ext.asked.length, 0);
  assert.equal(ext.saves(), 1);
});

test('extension.js: hue.dangerReadonly makes the danger places read-only, and takes out only its globs', async () => {
  const ext = await loadExtension({
    hue: { dangerReadonly: true, danger: ['infra', 'app/prod'] },
    readonly: { '/w/vendor/**': true },
  });
  assert.deepEqual(ext.readonly(), { '/w/vendor/**': true, '/w/infra/**': true, '/w/app/prod': true, '/w/app/prod/**': true });
  await ext.setHue({ danger: ['app/prod'] });
  assert.deepEqual(ext.readonly(), { '/w/vendor/**': true, '/w/app/prod': true, '/w/app/prod/**': true });
  const writes = ext.fileWrites.length;
  await ext.setHue({ strength: 'strong' }); // nothing about danger changed
  assert.equal(ext.fileWrites.length, writes, 'no write when the globs are the same');
  await ext.setHue({ dangerReadonly: false });
  assert.deepEqual(ext.readonly(), { '/w/vendor/**': true }, 'off: only the user\'s glob is left');
  await ext.setHue({ dangerReadonly: true, enabled: false });
  assert.deepEqual(ext.readonly(), { '/w/vendor/**': true }, 'Hue off: none either');
});

test('extension.js: without hue.dangerReadonly, files.readonlyInclude is never written', async () => {
  const ext = await loadExtension({ hue: { danger: ['infra'] } });
  await ext.setHue({ danger: ['infra', 'app'] });
  assert.deepEqual(ext.fileWrites, []);
  assert.equal(ext.readonly(), undefined);
});

test('extension.js: the last Hue glob out removes the setting', async () => {
  const ext = await loadExtension({ hue: { dangerReadonly: true, danger: ['infra'] } });
  assert.deepEqual(ext.readonly(), { '/w/infra/**': true });
  await ext.setHue({ danger: [] });
  assert.equal(ext.readonly(), undefined);
});

test('extension.js: Hue: Open shows the page, which changes the settings and follows them', async () => {
  const ext = await loadExtension({ hue: { folders: { infra: 'blue' }, danger: ['app/prod'] } });
  ext.commands['hue.open']();
  assert.equal(ext.panels, 1);
  ext.commands['hue.open']();
  assert.equal(ext.panels, 1, 'one page: the second open reveals it');
  assert.equal(ext.revealed, 1);
  await ext.fromPage({ type: 'ready' });
  assert.deepEqual(ext.state().folders.map((f) => [f.name, f.color, f.prodPaths]), [['infra', 'blue', []], ['app', null, ['prod']]]);

  await ext.fromPage({ type: 'setColor', folder: 'app', value: 'pink' });
  assert.deepEqual(ext.hueWrites.at(-1), ['folders', { infra: 'blue', app: 'pink' }]);
  assert.equal(ext.state().folders[1].color, 'pink', 'the page is redrawn after the write');

  await ext.fromPage({ type: 'setSetting', key: 'dangerReadonly', value: true });
  assert.deepEqual(ext.readonly(), { '/w/app/prod': true, '/w/app/prod/**': true }, 'and the setting takes effect');

  const writes = ext.hueWrites.length;
  await ext.fromPage({ type: 'setSetting', key: 'danger', value: [] });
  await ext.fromPage({ type: 'addProdPath', folder: 'app', rel: 'x' }); // only from the folder picker
  assert.equal(ext.hueWrites.length, writes, 'nothing written for what the page may not send');

  ext.closePanel();
  ext.commands['hue.open']();
  assert.equal(ext.panels, 2, 'closed: the next open makes a new one');
});

test('extension.js: Add path… marks what the folder picker gives, inside the folder only', async () => {
  const ext = await loadExtension({ hue: { danger: [] } });
  ext.commands['hue.open']();
  ext.picked = [{ scheme: 'file', fsPath: '/w/app/k8s/prod' }];
  await ext.fromPage({ type: 'pickProdPath', folder: 'app' });
  assert.deepEqual(ext.hueWrites.at(-1), ['danger', ['app/k8s/prod']]);
  ext.picked = [{ scheme: 'file', fsPath: '/w/infra/x' }]; // another folder
  await ext.fromPage({ type: 'pickProdPath', folder: 'app' });
  ext.picked = [{ scheme: 'file', fsPath: '/w/app' }]; // the folder itself
  await ext.fromPage({ type: 'pickProdPath', folder: 'app' });
  assert.equal(ext.hueWrites.length, 1);
  assert.equal(ext.warned.length, 2);
  ext.picked = undefined; // cancelled
  await ext.fromPage({ type: 'pickProdPath', folder: 'app' });
  assert.equal(ext.hueWrites.length, 1);
  assert.equal(ext.warned.length, 2);
});

test('extension.js: Hue: Search outside a .code-workspace only says so', async () => {
  const ext = await loadExtension();
  ext.workspace.workspaceFile = undefined;
  await ext.commands['hue.search']();
  assert.equal(ext.quickPicks.length, 0);
  assert.equal(ext.informed.length, 1);
});

test('extension.js: Hue: Search, the top menu: folders with colored dots, Enter goes there', async () => {
  const ext = await loadExtension({ hue: { folders: { infra: 'blue' }, danger: ['app'] } });
  await ext.search();
  assert.deepEqual(ext.qp.items.filter((i) => i.kind === -1).map((i) => i.label), ['Folders', 'Hue']);
  assert.match(fs.readFileSync(ext.item('infra').iconPath.fsPath, 'utf8'), /fill="#[0-9a-f]{6}"/, 'an svg dot in the folder\'s color');
  assert.deepEqual([ext.item('app').iconPath.id, ext.item('app').iconPath.color.id], ['warning', 'errorForeground']);
  assert.deepEqual(ext.item('infra').buttons.map((b) => b.iconPath.id), ['terminal', 'symbol-color', 'warning']);
  assert.ok(ext.qp.visible);
  await ext.accept('app');
  assert.ok(!ext.qp.visible && ext.qp.disposed, 'closed after going');
  assert.ok(ext.executed.some(([c, uri]) => c === 'revealInExplorer' && uri.fsPath === '/w/app'));
});

test('extension.js: Hue: Search, the buttons of a folder line', async () => {
  const ext = await loadExtension({ hue: { folders: { infra: 'blue' } } });
  await ext.search();
  await ext.press('app', 'prod');
  assert.deepEqual(ext.hueWrites.at(-1), ['danger', ['app']]);
  assert.ok(ext.qp.visible, 'PROD flips in place: the box stays');
  assert.equal(ext.item('app').description, 'PROD');
  await ext.press('app', 'terminal');
  assert.equal(ext.terminals.at(-1).name, '⚠ app');
  assert.ok(!ext.qp.visible);
  await ext.search();
  await ext.press('infra', 'color');
  assert.equal(ext.qp.title, 'Hue: color of infra');
  assert.equal(ext.qp.buttons[0].id, 'back');
  await ext.accept('teal');
  assert.deepEqual(ext.hueWrites.at(-1), ['folders', { infra: 'teal' }]);
  assert.ok(!ext.qp.visible);
});

test('extension.js: Hue: Search, Color… with Back and Custom…', async () => {
  const ext = await loadExtension();
  await ext.search();
  await ext.accept('Color of infra…');
  await ext.back();
  assert.equal(ext.qp.title, undefined, 'Back: the top menu');
  assert.deepEqual(ext.qp.buttons, []);
  await ext.accept('Color of infra…');
  ext.inputs.push('#123ABC ');
  await ext.accept('Custom…');
  assert.deepEqual(ext.hueWrites.at(-1), ['folders', { infra: '#123ABC' }]);
  await ext.search();
  await ext.accept('Color of infra…');
  ext.inputs.push(undefined); // Escape in the input box
  await ext.accept('Custom #123ABC');
  assert.equal(ext.hueWrites.length, 1);
});

test('extension.js: Hue: Search, Settings… stays open to flip several', async () => {
  const ext = await loadExtension({ hue: { danger: ['infra'] } });
  await ext.search();
  await ext.accept('Settings…');
  assert.equal(ext.qp.title, 'Hue: settings');
  await ext.accept('PROD opens read-only');
  assert.deepEqual(ext.hueWrites.at(-1), ['dangerReadonly', true]);
  assert.deepEqual(ext.readonly(), { '/w/infra/**': true }, 'and it takes effect');
  assert.ok(ext.qp.visible);
  assert.equal(ext.item('PROD opens read-only').iconPath.id, 'check', 'redrawn with the check');
  await ext.accept('strong');
  assert.deepEqual(ext.hueWrites.at(-1), ['strength', 'strong']);
  ext.inputs.push('main, release');
  await ext.accept('none'); // the fake starts with no protected branches
  assert.deepEqual(ext.hueWrites.at(-1), ['protectedBranches', ['main', 'release']]);
  await ext.search();
  await ext.accept('Settings…');
  await ext.back();
  assert.equal(ext.qp.title, undefined);
});
