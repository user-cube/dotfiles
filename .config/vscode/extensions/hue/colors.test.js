// Tests of colors.js, Hue without VS Code: node --test (CI runs them).
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const {
  DEFAULT_COLORS, OWNED, PARTS,
  hueOfHex, hueOf, autoHue, folderHue, hex, palette, ansiFor, globRegex, isDanger, nextCustomizations, readonlyGlobs, nextReadonly,
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

// extension.js with a fake `vscode`: folders infra and app, an editor whose
// file can be switched, the events and commands it registers, what it writes.
// git: what vscode.extensions.getExtension('vscode.git') returns
const settle = () => new Promise((resolve) => setImmediate(resolve));

async function loadExtension({ hue = {}, file = '/w/infra/main.tf', git, readonly } = {}) {
  const Module = require('node:module');
  const folders = ['infra', 'app'].map((name) => ({ name, uri: { fsPath: `/w/${name}`, scheme: 'file' } }));
  const ext = { updates: [], fileWrites: [], commands: {}, events: {}, terminals: [], folderWrites: 0, prod: false, executed: [], asked: [], answer: undefined };
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
      showWarningMessage: async (message) => { ext.asked.push(message); return ext.answer; },
      onDidChangeActiveTextEditor: on('editor'), onDidChangeActiveColorTheme: on('theme'), onDidChangeActiveTerminal: on('terminal'),
    },
    workspace: {
      workspaceFile: { fsPath: '/w/x.code-workspace' },
      workspaceFolders: folders,
      getWorkspaceFolder: (uri) => folders.find((f) => uri.fsPath === f.uri.fsPath || uri.fsPath.startsWith(`${f.uri.fsPath}/`)),
      getConfiguration: (section) => section === 'hue'
        ? { get: (key) => settings[key] }
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
    StatusBarAlignment: { Left: 1 }, ColorThemeKind: { Light: 1, Dark: 2, HighContrast: 3, HighContrastLight: 4 },
    ConfigurationTarget: { Workspace: 2 },
    ThemeColor: class { constructor(id) { this.id = id; } },
    ThemeIcon: class { constructor(id) { this.id = id; } },
  };
  const state = new Map();
  const context = {
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
    ['hue.clear', 'hue.goToFolder', 'hue.openTerminal', 'hue.refresh', 'hue.save', 'hue.setColor', 'hue.toggleDanger']);
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
