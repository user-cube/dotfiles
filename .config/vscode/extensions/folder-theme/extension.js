// Folder Theme: tints the window (title bar, activity bar, sidebar, status bar) with the
// color of the workspace folder of the active editor. Window colors cannot be set per
// folder, so the colors are written to the workspace's workbench.colorCustomizations
// whenever the active folder changes. Only keys of its own are touched.
const vscode = require('vscode');

const HUES = { red: 4, orange: 24, yellow: 46, green: 140, teal: 176, blue: 212, purple: 268, pink: 328 };

function hueOf(value) {
  if (value in HUES) return HUES[value];
  const m = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(value);
  if (!m) return undefined;
  const [r, g, b] = [m[1], m[2], m[3]].map((x) => parseInt(x, 16) / 255);
  const max = Math.max(r, g, b), d = max - Math.min(r, g, b);
  if (d === 0) return 0;
  const h = max === r ? ((g - b) / d) % 6 : max === g ? (b - r) / d + 2 : (r - g) / d + 4;
  return Math.round((h * 60 + 360) % 360);
}

function hex(h, s, l) {
  s /= 100; l /= 100;
  const a = s * Math.min(l, 1 - l);
  const f = (n) => {
    const k = (n + h / 30) % 12;
    const c = l - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
    return Math.round(255 * c).toString(16).padStart(2, '0');
  };
  return `#${f(0)}${f(8)}${f(4)}`;
}

// Dark: a strong bar color, a barely tinted sidebar so the files stay the focus.
// Light: the same idea, pale.
function palette(h, light) {
  const t = light
    ? { bar: hex(h, 55, 84), barOff: hex(h, 35, 90), act: hex(h, 45, 88), side: hex(h, 40, 96), head: hex(h, 40, 92),
        fg: '#1f2328', fgOff: '#57606a', accent: hex(h, 70, 40), sel: hex(h, 55, 86), hover: hex(h, 45, 92) }
    : { bar: hex(h, 42, 24), barOff: hex(h, 25, 15), act: hex(h, 30, 15), side: hex(h, 16, 11), head: hex(h, 22, 15),
        fg: '#ffffffee', fgOff: '#ffffff88', accent: hex(h, 75, 62), sel: hex(h, 35, 25), hover: hex(h, 20, 17) };
  return {
    'titleBar.activeBackground': t.bar,
    'titleBar.inactiveBackground': t.barOff,
    'titleBar.activeForeground': t.fg,
    'titleBar.inactiveForeground': t.fgOff,
    'activityBar.background': t.act,
    'activityBar.foreground': t.fg,
    'activityBar.inactiveForeground': t.fgOff,
    'activityBar.activeBorder': t.accent,
    'sideBar.background': t.side,
    'sideBarSectionHeader.background': t.head,
    'statusBar.background': t.bar,
    'statusBar.foreground': t.fg,
    'tab.activeBorderTop': t.accent,
    'list.activeSelectionBackground': t.sel,
    'list.inactiveSelectionBackground': t.head,
    'list.hoverBackground': t.hover,
  };
}

const OWNED = Object.keys(palette(0, false));

function activate(context) {
  let applied;

  async function apply() {
    // Colors land in the .code-workspace file; never write a repo's .vscode/settings.json
    if (!vscode.workspace.workspaceFile) return;
    const folders = vscode.workspace.workspaceFolders || [];
    const doc = vscode.window.activeTextEditor && vscode.window.activeTextEditor.document;
    const active = doc && vscode.workspace.getWorkspaceFolder(doc.uri);
    let name = active ? active.name : context.workspaceState.get('folder');
    if (!name && folders.length) name = folders[0].name;
    if (!name) return;
    context.workspaceState.update('folder', name);

    const map = vscode.workspace.getConfiguration('folderTheme').get('folders') || {};
    const hue = map[name] === undefined ? undefined : hueOf(map[name]);
    const kind = vscode.window.activeColorTheme.kind;
    const light = kind === vscode.ColorThemeKind.Light || kind === vscode.ColorThemeKind.HighContrastLight;
    const key = `${name}|${hue}|${light}`;
    if (key === applied) return;

    const cfg = vscode.workspace.getConfiguration('workbench');
    const current = { ...((cfg.inspect('colorCustomizations') || {}).workspaceValue || {}) };
    const next = { ...current };
    for (const k of OWNED) delete next[k];
    if (hue !== undefined) Object.assign(next, palette(hue, light));
    applied = key;
    if (JSON.stringify(next) === JSON.stringify(current)) return;
    await cfg.update('colorCustomizations', Object.keys(next).length ? next : undefined, vscode.ConfigurationTarget.Workspace);
  }

  context.subscriptions.push(
    vscode.window.onDidChangeActiveTextEditor(apply),
    vscode.window.onDidChangeActiveColorTheme(() => { applied = undefined; apply(); }),
    vscode.workspace.onDidChangeWorkspaceFolders(apply),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration('folderTheme')) { applied = undefined; apply(); }
    }),
    vscode.commands.registerCommand('folderTheme.refresh', () => { applied = undefined; return apply(); }),
  );
  apply();
}

module.exports = { activate, deactivate() {} };
