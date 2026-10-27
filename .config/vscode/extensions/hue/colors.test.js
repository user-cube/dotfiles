// Tests of colors.js, Hue without VS Code: node --test (CI runs them).
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const {
  DEFAULT_COLORS, OWNED, PARTS,
  hueOfHex, hueOf, autoHue, hex, palette, ansiFor, globRegex, isDanger, nextCustomizations,
} = require('./colors');

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

test('install.sh packages every local module extension.js and colors.js require', () => {
  const install = fs.readFileSync(path.join(__dirname, 'install.sh'), 'utf8');
  for (const file of ['extension.js', 'colors.js']) {
    const src = fs.readFileSync(path.join(__dirname, file), 'utf8');
    for (const [, mod] of src.matchAll(/require\('\.\/([^']+)'\)/g)) {
      const name = mod.endsWith('.js') ? mod : `${mod}.js`;
      assert.ok(install.includes(`"$dir/${name}"`), `${file} requires ./${mod}, which install.sh doesn't put in the .vsix`);
    }
  }
});

// extension.js with a fake `vscode`: activate registers the commands, and apply
// writes the active folder's colors (and the danger red) to the workspace
test('extension.js: activate, then the active folder\'s colors land in the workspace', async () => {
  const Module = require('node:module');
  const updates = [];
  let prodShown = false;
  const folder = { name: 'infra', uri: { fsPath: '/w/infra', scheme: 'file' } };
  const hue = { folders: { infra: 'blue' }, danger: ['infra/prod'], auto: true, protectedBranches: [], parts: ['statusBar'] };
  const commands = [];
  const event = () => () => ({ dispose() {} });
  const fake = {
    window: {
      activeTextEditor: { document: { uri: { scheme: 'file', fsPath: '/w/infra/prod/main.tf' } } },
      activeColorTheme: { kind: 2 },
      createStatusBarItem: () => ({ show() { prodShown = true; }, hide() { prodShown = false; }, dispose() {} }),
      onDidChangeActiveTextEditor: event(), onDidChangeActiveColorTheme: event(),
    },
    workspace: {
      workspaceFile: { fsPath: '/w/x.code-workspace' },
      workspaceFolders: [folder],
      getWorkspaceFolder: () => folder,
      getConfiguration: (section) => section === 'hue'
        ? { get: (key) => hue[key] }
        : { inspect: () => ({ workspaceValue: { 'editor.background': '#000000' } }), update: async (key, value) => updates.push([key, value]) },
      onDidChangeWorkspaceFolders: event(), onDidChangeConfiguration: event(),
    },
    commands: { registerCommand: (name) => { commands.push(name); return { dispose() {} }; } },
    extensions: { getExtension: () => undefined },
    StatusBarAlignment: { Left: 1 }, ColorThemeKind: { Light: 1, Dark: 2, HighContrast: 3, HighContrastLight: 4 },
    ConfigurationTarget: { Workspace: 2 }, ThemeColor: class {}, ThemeIcon: class {},
  };
  const load = Module._load;
  Module._load = function (request, ...rest) { return request === 'vscode' ? fake : load.call(this, request, ...rest); };
  try {
    delete require.cache[require.resolve('./extension')];
    const state = new Map();
    require('./extension').activate({ subscriptions: [], workspaceState: { get: (k) => state.get(k), update: (k, v) => state.set(k, v) } });
    await new Promise((resolve) => setImmediate(resolve));
  } finally {
    Module._load = load;
  }
  assert.deepEqual(commands.sort(), ['hue.clear', 'hue.goToFolder', 'hue.openTerminal', 'hue.refresh', 'hue.setColor', 'hue.toggleDanger']);
  assert.equal(updates.length, 1);
  const [key, colors] = updates[0];
  assert.equal(key, 'colorCustomizations');
  assert.deepEqual(colors, nextCustomizations({ 'editor.background': '#000000' }, { hue: DEFAULT_COLORS.blue, light: false, parts: ['statusBar'], danger: true }));
  assert.ok(prodShown, 'the PROD marker is shown');
});
