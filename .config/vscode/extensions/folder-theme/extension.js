// Folder Theme: tints the window (title bar, activity bar, sidebar, status bar) with the
// color of the workspace folder of the active editor. A folder with no color of its own
// gets a stable one from its name (folderTheme.auto). Window colors cannot be set per
// folder, so the colors are written to the workspace's workbench.colorCustomizations
// whenever the active folder changes. Only keys of its own are touched.
//
// Settings: folderTheme.folders (folder name -> color), folderTheme.colors (the named
// colors: name -> hue 0-360 or #rrggbb), folderTheme.strength, folderTheme.parts,
// folderTheme.danger (folders that get a red title bar and a PROD marker) and
// folderTheme.protectedBranches (the status bar turns amber on those branches).
const vscode = require('vscode');

const DEFAULT_COLORS = { red: 4, orange: 24, yellow: 46, green: 140, teal: 176, blue: 212, purple: 268, pink: 328 };
const STRENGTH = { subtle: 0.55, normal: 1, strong: 1.35 };

function hueOfHex(value) {
  const m = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(value);
  if (!m) return undefined;
  const [r, g, b] = [m[1], m[2], m[3]].map((x) => parseInt(x, 16) / 255);
  const max = Math.max(r, g, b), d = max - Math.min(r, g, b);
  if (d === 0) return 0;
  const h = max === r ? ((g - b) / d) % 6 : max === g ? (b - r) / d + 2 : (r - g) / d + 4;
  return Math.round((h * 60 + 360) % 360);
}

// A color value is a name from `colors`, a hue number or a #rrggbb
function hueOf(value, colors) {
  if (typeof value === 'string' && value in colors) value = colors[value];
  if (typeof value === 'number') return ((value % 360) + 360) % 360;
  return typeof value === 'string' ? hueOfHex(value.trim()) : undefined;
}

// Same name, same color: djb2 hash of the folder name picks one of the named colors
function autoHue(name, colors) {
  const names = Object.keys(colors);
  if (!names.length) return undefined;
  let h = 5381;
  for (const c of name) h = ((h * 33) ^ c.charCodeAt(0)) >>> 0;
  return hueOf(names[h % names.length], colors);
}

function hex(h, s, l) {
  s = Math.min(s, 100) / 100; l /= 100;
  const a = s * Math.min(l, 1 - l);
  const f = (n) => {
    const k = (n + h / 30) % 12;
    const c = l - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
    return Math.round(255 * c).toString(16).padStart(2, '0');
  };
  return `#${f(0)}${f(8)}${f(4)}`;
}

// Keys by part. Dark: a strong bar color, a barely tinted sidebar so the files stay the
// focus. Light: the same idea, pale. k scales the saturation (folderTheme.strength).
function palette(h, light, k) {
  const t = light
    ? { bar: hex(h, 55 * k, 84), barOff: hex(h, 35 * k, 90), act: hex(h, 45 * k, 88), side: hex(h, 40 * k, 96),
        head: hex(h, 40 * k, 92), fg: '#1f2328', fgOff: '#57606a', accent: hex(h, 70, 40), sel: hex(h, 55 * k, 86),
        hover: hex(h, 45 * k, 92) }
    : { bar: hex(h, 42 * k, 24), barOff: hex(h, 25 * k, 15), act: hex(h, 30 * k, 15), side: hex(h, 16 * k, 11),
        head: hex(h, 22 * k, 15), fg: '#ffffffee', fgOff: '#ffffff88', accent: hex(h, 75, 62), sel: hex(h, 35 * k, 25),
        hover: hex(h, 20 * k, 17) };
  return {
    titleBar: {
      'titleBar.activeBackground': t.bar,
      'titleBar.inactiveBackground': t.barOff,
      'titleBar.activeForeground': t.fg,
      'titleBar.inactiveForeground': t.fgOff,
    },
    activityBar: {
      'activityBar.background': t.act,
      'activityBar.foreground': t.fg,
      'activityBar.inactiveForeground': t.fgOff,
      'activityBar.activeBorder': t.accent,
    },
    sideBar: {
      'sideBar.background': t.side,
      'sideBarSectionHeader.background': t.head,
      'list.activeSelectionBackground': t.sel,
      'list.inactiveSelectionBackground': t.head,
      'list.hoverBackground': t.hover,
    },
    statusBar: {
      'statusBar.background': t.bar,
      'statusBar.foreground': t.fg,
    },
    tabs: { 'tab.activeBorderTop': t.accent },
  };
}

// Terminal tabs only take theme colors: the nearest ANSI one to the hue
function ansiFor(h) {
  if (h < 15 || h >= 345) return 'terminal.ansiRed';
  if (h < 70) return 'terminal.ansiYellow';
  if (h < 165) return 'terminal.ansiGreen';
  if (h < 200) return 'terminal.ansiCyan';
  if (h < 250) return 'terminal.ansiBlue';
  if (h < 330) return 'terminal.ansiMagenta';
  return 'terminal.ansiBrightMagenta';
}

const AMBER = { dark: '#8a5a00', light: '#ffd58a' };
const DANGER = { dark: ['#8a1c1c', '#4a1212'], light: ['#f5b5b5', '#f9d5d5'] };

const PARTS = Object.keys(palette(0, false, 1));
const OWNED = Object.values(palette(0, false, 1)).flatMap((p) => Object.keys(p));

function activate(context) {
  function colorsSetting() {
    return { ...DEFAULT_COLORS, ...(vscode.workspace.getConfiguration('folderTheme').get('colors') || {}) };
  }

  function activeFolderName() {
    const doc = vscode.window.activeTextEditor && vscode.window.activeTextEditor.document;
    const active = doc && vscode.workspace.getWorkspaceFolder(doc.uri);
    if (active) return active.name;
    return context.workspaceState.get('folder');
  }

  // The vscode.git extension API, to read the branch of a folder's repository
  let gitApi;
  const watched = new Set();
  async function git() {
    if (gitApi !== undefined) return gitApi;
    gitApi = null;
    try {
      const ext = vscode.extensions.getExtension('vscode.git');
      const exports = ext && (ext.isActive ? ext.exports : await ext.activate());
      gitApi = exports ? exports.getAPI(1) : null;
    } catch (e) { /* no git: no branch warning */ }
    if (gitApi) {
      const watch = (repo) => {
        if (watched.has(repo)) return;
        watched.add(repo);
        context.subscriptions.push(repo.state.onDidChange(apply));
      };
      gitApi.repositories.forEach(watch);
      context.subscriptions.push(gitApi.onDidOpenRepository((repo) => { watch(repo); apply(); }));
    }
    return gitApi;
  }

  async function branchOf(folder) {
    const api = await git();
    const repo = api && api.getRepository(folder.uri);
    return repo && repo.state.HEAD && repo.state.HEAD.name;
  }

  const dangerItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 10000);
  dangerItem.text = '$(warning) PROD';
  dangerItem.tooltip = 'Folder Theme: this folder is marked as dangerous';
  dangerItem.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
  context.subscriptions.push(dangerItem);

  async function apply() {
    // Colors land in the .code-workspace file; never write a repo's .vscode/settings.json
    if (!vscode.workspace.workspaceFile) { dangerItem.hide(); return; }
    const folders = vscode.workspace.workspaceFolders || [];
    let name = activeFolderName();
    if (!name && folders.length) name = folders[0].name;
    if (!name) return;
    context.workspaceState.update('folder', name);
    const folder = folders.find((f) => f.name === name);

    const settings = vscode.workspace.getConfiguration('folderTheme');
    const colors = colorsSetting();
    const map = settings.get('folders') || {};
    let hue = map[name] === undefined ? undefined : hueOf(map[name], colors);
    if (hue === undefined && map[name] === undefined && settings.get('auto')) hue = autoHue(name, colors);
    const kind = vscode.window.activeColorTheme.kind;
    const light = kind === vscode.ColorThemeKind.Light || kind === vscode.ColorThemeKind.HighContrastLight;
    const k = STRENGTH[settings.get('strength')] || 1;
    const parts = new Set(settings.get('parts') || PARTS);
    const danger = (settings.get('danger') || []).includes(name);
    const protectedBranches = settings.get('protectedBranches') || [];
    const branch = folder && protectedBranches.length ? await branchOf(folder) : undefined;
    const onProtected = !!branch && protectedBranches.includes(branch);
    const theme = light ? 'light' : 'dark';

    if (danger) dangerItem.show(); else dangerItem.hide();

    const cfg = vscode.workspace.getConfiguration('workbench');
    const current = { ...((cfg.inspect('colorCustomizations') || {}).workspaceValue || {}) };
    const next = { ...current };
    for (const key of OWNED) delete next[key];
    if (hue !== undefined) {
      const groups = palette(hue, light, k);
      for (const part of PARTS) if (part === 'tabs' || parts.has(part)) Object.assign(next, groups[part]);
    }
    // Warnings win over the folder color
    if (onProtected) {
      next['statusBar.background'] = AMBER[theme];
      next['statusBar.foreground'] = light ? '#1f2328' : '#ffffffee';
    }
    if (danger) {
      next['titleBar.activeBackground'] = DANGER[theme][0];
      next['titleBar.inactiveBackground'] = DANGER[theme][1];
      next['titleBar.activeForeground'] = light ? '#1f2328' : '#ffffffee';
      next['titleBar.inactiveForeground'] = light ? '#57606a' : '#ffffff88';
    }
    if (JSON.stringify(next) === JSON.stringify(current)) return;
    await cfg.update('colorCustomizations', Object.keys(next).length ? next : undefined, vscode.ConfigurationTarget.Workspace);
  }

  // The folder a command acts on: the active editor's, else ask when there are several
  async function commandFolder() {
    const folders = vscode.workspace.workspaceFolders || [];
    if (!vscode.workspace.workspaceFile || !folders.length) {
      vscode.window.showInformationMessage('Folder Theme: open a .code-workspace first.');
      return undefined;
    }
    const doc = vscode.window.activeTextEditor && vscode.window.activeTextEditor.document;
    const active = doc && vscode.workspace.getWorkspaceFolder(doc.uri);
    if (active) return active;
    if (folders.length === 1) return folders[0];
    const name = await vscode.window.showQuickPick(folders.map((f) => f.name), { placeHolder: 'Folder' });
    return folders.find((f) => f.name === name);
  }

  // Quick pick: set (or clear) the color of the active folder, no JSON editing
  async function setColor() {
    const folder = await commandFolder();
    if (!folder) return;
    const name = folder.name;

    const items = [
      ...Object.keys(colorsSetting()).map((c) => ({ label: `$(circle-filled) ${c}`, value: c })),
      { label: '$(symbol-color) Custom #rrggbb…', value: 'custom' },
      { label: '$(sync) Automatic (from the name)', value: 'auto' },
    ];
    const pick = await vscode.window.showQuickPick(items, { placeHolder: `Color for ${name}` });
    if (!pick) return;

    let value = pick.value;
    if (value === 'custom') {
      value = await vscode.window.showInputBox({
        prompt: `Color for ${name}`, placeHolder: '#3366ff',
        validateInput: (v) => (hueOfHex(v.trim()) === undefined ? 'Use #rrggbb' : undefined),
      });
      if (!value) return;
      value = value.trim();
    }
    const next = { ...(vscode.workspace.getConfiguration('folderTheme').get('folders') || {}) };
    if (value === 'auto') delete next[name]; else next[name] = value;
    // The configuration change re-applies the colors
    await vscode.workspace.getConfiguration('folderTheme').update('folders', next, vscode.ConfigurationTarget.Workspace);
  }

  async function toggleDanger() {
    const folder = await commandFolder();
    if (!folder) return;
    const settings = vscode.workspace.getConfiguration('folderTheme');
    const list = settings.get('danger') || [];
    const next = list.includes(folder.name) ? list.filter((n) => n !== folder.name) : [...list, folder.name];
    await settings.update('danger', next, vscode.ConfigurationTarget.Workspace);
    vscode.window.showInformationMessage(`Folder Theme: ${folder.name} is ${next.includes(folder.name) ? 'now' : 'no longer'} marked as dangerous.`);
  }

  // A terminal in the folder, its tab in the folder's color
  async function openTerminal() {
    const folder = await commandFolder();
    if (!folder) return;
    const map = vscode.workspace.getConfiguration('folderTheme').get('folders') || {};
    const colors = colorsSetting();
    let hue = map[folder.name] === undefined ? undefined : hueOf(map[folder.name], colors);
    if (hue === undefined && map[folder.name] === undefined) hue = autoHue(folder.name, colors);
    vscode.window.createTerminal({
      name: folder.name,
      cwd: folder.uri,
      color: hue === undefined ? undefined : new vscode.ThemeColor(ansiFor(hue)),
      iconPath: new vscode.ThemeIcon('terminal'),
    }).show();
  }

  context.subscriptions.push(
    vscode.window.onDidChangeActiveTextEditor(apply),
    vscode.window.onDidChangeActiveColorTheme(apply),
    vscode.workspace.onDidChangeWorkspaceFolders(apply),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration('folderTheme')) apply();
    }),
    vscode.commands.registerCommand('folderTheme.setColor', setColor),
    vscode.commands.registerCommand('folderTheme.toggleDanger', toggleDanger),
    vscode.commands.registerCommand('folderTheme.openTerminal', openTerminal),
    vscode.commands.registerCommand('folderTheme.refresh', apply),
  );
  apply();
}

module.exports = { activate, deactivate() {} };
