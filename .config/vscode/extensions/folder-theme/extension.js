// Folder Theme: tints the window (title bar, activity bar, sidebar, status bar) with the
// color of the workspace folder of the active editor. A folder with no color of its own
// gets a stable one from its name (folderTheme.auto). Window colors cannot be set per
// folder, so the colors are written to the workspace's workbench.colorCustomizations
// whenever the active folder changes. Only keys of its own are touched.
//
// Settings: folderTheme.folders (folder name -> color), folderTheme.colors (the named
// colors: name -> hue 0-360 or #rrggbb), folderTheme.strength, folderTheme.parts,
// folderTheme.danger (folders, or paths inside them, that get a red title bar and a PROD marker),
// folderTheme.dangerContexts (the same while the active Kubernetes context is one of them) and
// folderTheme.protectedBranches (the status bar turns amber on those branches).
const vscode = require('vscode');
const fs = require('fs');
const os = require('os');
const path = require('path');

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

// Glob for a path inside a folder: ** crosses directories, * and ? do not
function globRegex(glob) {
  const re = glob.replace(/[.+^${}()|[\]\\]/g, '\\$&').replace(/\*\*\/?/g, '\u0000').replace(/\*/g, '[^/]*').replace(/\?/g, '[^/]')
    .replace(/\u0000/g, '(?:.*/)?');
  return new RegExp(`^${re}$`);
}

// danger entries: "folder" (all of it) or "folder/some/path" (that path, globs allowed).
// rel = the active file's path inside the folder, undefined when there is none
function isDanger(entries, folderName, rel) {
  for (const entry of entries) {
    const slash = entry.indexOf('/');
    if (slash === -1) { if (entry === folderName) return entry; continue; }
    if (entry.slice(0, slash) !== folderName || rel === undefined) continue;
    const sub = entry.slice(slash + 1).replace(/\/+$/, '');
    const parts = rel.split('/');
    // The path itself or anything under it: test every leading part of the file's path
    for (let i = 1; i <= parts.length; i++) if (globRegex(sub).test(parts.slice(0, i).join('/'))) return entry;
  }
  return undefined;
}

// A plain name, or a pattern where * stands for any text and ? for one character
function nameMatches(pattern, value) {
  const re = pattern.replace(/[.+^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*').replace(/\?/g, '.');
  return new RegExp(`^${re}$`).test(value);
}

// Only ~/.kube/config: the file kubectl and kontext change. A KUBECONFIG exported in a
// terminal is not visible to VS Code, so it is not supported
function kubeconfigPath() { return path.join(os.homedir(), '.kube', 'config'); }

// Only the current-context line is read, nothing else
function currentContext() {
  try {
    const m = /^current-context:\s*(.+?)\s*$/m.exec(fs.readFileSync(kubeconfigPath(), 'utf8'));
    return m ? m[1].replace(/^["']|["']$/g, '') : undefined;
  } catch (e) { return undefined; }
}

const AMBER = { dark: '#8a5a00', light: '#ffd58a' };
const DANGER = { dark: ['#8a1c1c', '#4a1212'], light: ['#f5b5b5', '#f9d5d5'] };

const PARTS = Object.keys(palette(0, false, 1));
const OWNED = Object.values(palette(0, false, 1)).flatMap((p) => Object.keys(p));

function activate(context) {
  function colorsSetting() {
    return { ...DEFAULT_COLORS, ...(vscode.workspace.getConfiguration('folderTheme').get('colors') || {}) };
  }

  // The active file's path inside `folder` (posix), or undefined
  function activeRelPath(folder) {
    const doc = vscode.window.activeTextEditor && vscode.window.activeTextEditor.document;
    if (!doc || !folder || doc.uri.scheme !== 'file') return undefined;
    const rel = path.relative(folder.uri.fsPath, doc.uri.fsPath);
    return rel.startsWith('..') || path.isAbsolute(rel) ? undefined : rel.split(path.sep).join('/');
  }

  // Set by Go to Folder when the folder has no file to open; cleared by the next editor change
  let forced;
  const lastFiles = () => context.workspaceState.get('lastFiles') || {};
  function rememberFile() {
    const doc = vscode.window.activeTextEditor && vscode.window.activeTextEditor.document;
    const folder = doc && doc.uri.scheme === 'file' && vscode.workspace.getWorkspaceFolder(doc.uri);
    if (!folder) return;
    const files = lastFiles();
    if (files[folder.name] !== doc.uri.toString()) context.workspaceState.update('lastFiles', { ...files, [folder.name]: doc.uri.toString() });
  }

  function activeFolderName() {
    if (forced) return forced;
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
    const enabled = settings.get('enabled') !== false;
    let hue = map[name] === undefined ? undefined : hueOf(map[name], colors);
    if (hue === undefined && map[name] === undefined && settings.get('auto')) hue = autoHue(name, colors);
    if (!enabled) hue = undefined;
    const kind = vscode.window.activeColorTheme.kind;
    const light = kind === vscode.ColorThemeKind.Light || kind === vscode.ColorThemeKind.HighContrastLight;
    const k = STRENGTH[settings.get('strength')] || 1;
    const parts = new Set(settings.get('parts') || PARTS);
    const dangerEntry = enabled ? isDanger(settings.get('danger') || [], name, activeRelPath(folder)) : undefined;
    const kube = enabled ? currentContext() : undefined;
    const kubeDanger = !!kube && (settings.get('dangerContexts') || []).some((p) => nameMatches(p, kube));
    const danger = dangerEntry !== undefined || kubeDanger;
    const protectedBranches = settings.get('protectedBranches') || [];
    const branch = enabled && folder && protectedBranches.length ? await branchOf(folder) : undefined;
    const onProtected = !!branch && protectedBranches.includes(branch);
    const theme = light ? 'light' : 'dark';

    if (danger) {
      dangerItem.text = kubeDanger && dangerEntry === undefined ? `$(warning) PROD · ${kube}` : '$(warning) PROD';
      dangerItem.tooltip = [dangerEntry !== undefined && `Folder Theme: dangerous place (${dangerEntry})`,
        kubeDanger && `Folder Theme: dangerous Kubernetes context (${kube})`].filter(Boolean).join('\n');
      dangerItem.show();
    } else dangerItem.hide();

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

  // Mark or unmark the whole folder, the active file's directory, or an existing entry
  async function toggleDanger() {
    const folder = await commandFolder();
    if (!folder) return;
    const settings = vscode.workspace.getConfiguration('folderTheme');
    const list = settings.get('danger') || [];
    const rel = activeRelPath(folder);
    const dir = rel && path.posix.dirname(rel) !== '.' ? `${folder.name}/${path.posix.dirname(rel)}` : undefined;
    const candidates = [folder.name, dir, ...list.filter((e) => e === folder.name || e.startsWith(`${folder.name}/`))];
    const items = [...new Set(candidates.filter(Boolean))].map((entry) => ({
      label: `${list.includes(entry) ? '$(check)' : '$(circle-outline)'} ${entry}`,
      description: entry === folder.name ? 'the whole folder' : undefined,
      entry,
    }));
    const pick = await vscode.window.showQuickPick(items, { placeHolder: 'Toggle danger (PROD)' });
    if (!pick) return;
    const next = list.includes(pick.entry) ? list.filter((e) => e !== pick.entry) : [...list, pick.entry];
    await settings.update('danger', next, vscode.ConfigurationTarget.Workspace);
  }

  // Add or remove the current Kubernetes context in the list (a user setting: contexts are yours, not a workspace's)
  async function toggleContextDanger() {
    const kube = currentContext();
    if (!kube) {
      vscode.window.showInformationMessage('Folder Theme: no current Kubernetes context found in your kubeconfig.');
      return;
    }
    const settings = vscode.workspace.getConfiguration('folderTheme');
    const inspected = settings.inspect('dangerContexts') || {};
    const list = inspected.globalValue || [];
    const next = list.includes(kube) ? list.filter((c) => c !== kube) : [...list, kube];
    await settings.update('dangerContexts', next, vscode.ConfigurationTarget.Global);
    vscode.window.showInformationMessage(`Folder Theme: context ${kube} is ${next.includes(kube) ? 'now' : 'no longer'} marked as dangerous.`);
  }

  // A terminal in the folder, its tab in the folder's color (red with a warning in a danger place)
  async function openTerminal() {
    const folder = await commandFolder();
    if (!folder) return;
    const settings = vscode.workspace.getConfiguration('folderTheme');
    const map = settings.get('folders') || {};
    const colors = colorsSetting();
    let hue = map[folder.name] === undefined ? undefined : hueOf(map[folder.name], colors);
    if (hue === undefined && map[folder.name] === undefined) hue = autoHue(folder.name, colors);
    const kube = currentContext();
    const danger = isDanger(settings.get('danger') || [], folder.name, activeRelPath(folder)) !== undefined
      || (!!kube && (settings.get('dangerContexts') || []).some((p) => nameMatches(p, kube)));
    const red = new vscode.ThemeColor('terminal.ansiRed');
    vscode.window.createTerminal({
      name: danger ? `⚠ ${folder.name}` : folder.name,
      cwd: folder.uri,
      color: danger ? red : hue === undefined ? undefined : new vscode.ThemeColor(ansiFor(hue)),
      iconPath: danger ? new vscode.ThemeIcon('warning', red) : new vscode.ThemeIcon('terminal'),
    }).show();
  }

  // Quick pick of the workspace folders: opens the last file you had there, or reveals the folder
  async function goToFolder() {
    const folders = vscode.workspace.workspaceFolders || [];
    if (!vscode.workspace.workspaceFile || !folders.length) {
      vscode.window.showInformationMessage('Folder Theme: open a .code-workspace first.');
      return;
    }
    const files = lastFiles();
    const current = activeFolderName();
    const items = folders.map((f) => ({
      label: `${f.name === current ? '$(arrow-right)' : '$(folder)'} ${f.name}`,
      detail: files[f.name] ? vscode.workspace.asRelativePath(vscode.Uri.parse(files[f.name]), false) : undefined,
      folder: f,
    }));
    const pick = await vscode.window.showQuickPick(items, { placeHolder: 'Go to folder', matchOnDetail: true });
    if (!pick) return;
    if (files[pick.folder.name]) {
      try {
        const doc = await vscode.workspace.openTextDocument(vscode.Uri.parse(files[pick.folder.name]));
        await vscode.window.showTextDocument(doc);
        return;
      } catch (e) { /* the file is gone: reveal the folder instead */ }
    }
    forced = pick.folder.name;
    await vscode.commands.executeCommand('revealInExplorer', pick.folder.uri);
    apply();
  }

  // Take the colors out of the workspace and keep the extension off here
  async function clear() {
    await vscode.workspace.getConfiguration('folderTheme').update('enabled', false, vscode.ConfigurationTarget.Workspace);
    vscode.window.showInformationMessage('Folder Theme: colors removed. Set folderTheme.enabled to true to turn it back on.');
  }

  context.subscriptions.push(
    vscode.window.onDidChangeActiveTextEditor(() => { forced = undefined; rememberFile(); apply(); }),
    vscode.window.onDidChangeActiveColorTheme(apply),
    vscode.workspace.onDidChangeWorkspaceFolders(apply),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration('folderTheme')) apply();
    }),
    vscode.commands.registerCommand('folderTheme.setColor', setColor),
    vscode.commands.registerCommand('folderTheme.toggleDanger', toggleDanger),
    vscode.commands.registerCommand('folderTheme.openTerminal', openTerminal),
    vscode.commands.registerCommand('folderTheme.toggleContextDanger', toggleContextDanger),
    vscode.commands.registerCommand('folderTheme.goToFolder', goToFolder),
    vscode.commands.registerCommand('folderTheme.clear', clear),
    vscode.commands.registerCommand('folderTheme.refresh', apply),
  );
  // kubectl / kontext rewrite ~/.kube/config when the context changes: poll its stat, and
  // look again when the window gets focus
  const kubeconfig = kubeconfigPath();
  fs.watchFile(kubeconfig, { interval: 2000 }, apply);
  context.subscriptions.push({ dispose: () => fs.unwatchFile(kubeconfig, apply) });
  context.subscriptions.push(vscode.window.onDidChangeWindowState((state) => { if (state.focused) apply(); }));
  rememberFile();
  apply();
}

module.exports = { activate, deactivate() {} };
