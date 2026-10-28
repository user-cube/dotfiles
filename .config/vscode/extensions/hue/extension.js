// Hue: tints the window (title bar, activity bar, sidebar, status bar) with the
// color of the workspace folder of the active editor. A folder with no color of its own
// gets a stable one from its name (hue.auto). Window colors cannot be set per
// folder, so the colors are written to the workspace's workbench.colorCustomizations
// whenever the active folder changes. Only keys of its own are touched.
//
// Settings: hue.folders (folder name -> color), hue.colors (the named
// colors: name -> hue 0-360 or #rrggbb), hue.strength, hue.parts,
// hue.danger (folders, or paths inside them, that get a red title bar and a PROD marker) and
// hue.protectedBranches (the status bar turns amber on those branches).
const vscode = require('vscode');
const path = require('path');

const {
  DEFAULT_COLORS, STRENGTH, PARTS, hueOfHex, folderHue, ansiFor, isDanger, nextCustomizations,
} = require('./colors');

function activate(context) {
  function colorsSetting() {
    return { ...DEFAULT_COLORS, ...(vscode.workspace.getConfiguration('hue').get('colors') || {}) };
  }

  // A file uri's path inside `folder` (posix), or undefined (outside it, or the folder itself)
  function relPathIn(folder, uri) {
    if (!uri || !folder || uri.scheme !== 'file') return undefined;
    const rel = path.relative(folder.uri.fsPath, uri.fsPath);
    return !rel || rel.startsWith('..') || path.isAbsolute(rel) ? undefined : rel.split(path.sep).join('/');
  }

  // The active file's path inside `folder` (posix), or undefined
  function activeRelPath(folder) {
    const doc = vscode.window.activeTextEditor && vscode.window.activeTextEditor.document;
    return doc ? relPathIn(folder, doc.uri) : undefined;
  }

  // Where a terminal is: its folder and the path inside it, from the shell's
  // current directory (shell integration) or the one it was opened in. Inside
  // tmux there is no shell integration: the directory it was opened in
  function terminalPlace(terminal) {
    let cwd = (terminal.shellIntegration && terminal.shellIntegration.cwd) || (terminal.creationOptions && terminal.creationOptions.cwd);
    if (typeof cwd === 'string') cwd = vscode.Uri.file(cwd);
    const folder = cwd && vscode.workspace.getWorkspaceFolder(cwd);
    return folder ? { name: folder.name, rel: relPathIn(folder, cwd) } : undefined;
  }

  // The place the window follows instead of the active editor's file: a folder
  // picked in Go to Folder (no file to open), or the active terminal's. Cleared
  // by the next editor change. { name, rel }
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
    if (forced) return forced.name;
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
  dangerItem.tooltip = 'Hue: this folder is marked as dangerous';
  dangerItem.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
  context.subscriptions.push(dangerItem);

  // Each apply waits for git (the branch); one started later wins, so the colors
  // of a folder you just left never land after the ones of the folder you are in
  let latest = 0;
  async function apply() {
    const run = ++latest;
    // Colors land in the .code-workspace file; never write a repo's .vscode/settings.json
    if (!vscode.workspace.workspaceFile) { dangerItem.hide(); return; }
    const folders = vscode.workspace.workspaceFolders || [];
    let name = activeFolderName();
    if (!name && folders.length) name = folders[0].name;
    if (!name) return;
    // apply runs on every git state change: write only what changed
    if (context.workspaceState.get('folder') !== name) context.workspaceState.update('folder', name);
    const folder = folders.find((f) => f.name === name);

    const settings = vscode.workspace.getConfiguration('hue');
    const enabled = settings.get('enabled') !== false;
    const hue = folderHue(name, { folders: settings.get('folders') || {}, colors: colorsSetting(), auto: settings.get('auto') !== false, enabled });
    const kind = vscode.window.activeColorTheme.kind;
    const light = kind === vscode.ColorThemeKind.Light || kind === vscode.ColorThemeKind.HighContrastLight;
    const k = STRENGTH[settings.get('strength')] || 1;
    const rel = forced ? forced.rel : activeRelPath(folder);
    const dangerEntry = enabled ? isDanger(settings.get('danger') || [], name, rel) : undefined;
    const danger = dangerEntry !== undefined;
    const protectedBranches = settings.get('protectedBranches') || [];
    const branch = enabled && folder && protectedBranches.length ? await branchOf(folder) : undefined;
    const onProtected = !!branch && protectedBranches.includes(branch);
    if (run !== latest) return;

    if (danger) {
      dangerItem.tooltip = `Hue: dangerous place (${dangerEntry})`;
      dangerItem.show();
    } else dangerItem.hide();

    const cfg = vscode.workspace.getConfiguration('workbench');
    const current = { ...((cfg.inspect('colorCustomizations') || {}).workspaceValue || {}) };
    const next = nextCustomizations(current, { hue, light, k, parts: settings.get('parts') || PARTS, onProtected, danger });
    if (JSON.stringify(next) === JSON.stringify(current)) return;
    await cfg.update('colorCustomizations', Object.keys(next).length ? next : undefined, vscode.ConfigurationTarget.Workspace);
  }

  // ⌘S in a danger place asks first, once per place in a session. Only ⌘S (the
  // hue.dangerFile context turns its keybinding on): VS Code can't stop a save
  // that already started, from the menu or auto save
  function activeFileDanger() {
    const settings = vscode.workspace.getConfiguration('hue');
    const doc = vscode.window.activeTextEditor && vscode.window.activeTextEditor.document;
    const folder = doc && doc.uri.scheme === 'file' && vscode.workspace.getWorkspaceFolder(doc.uri);
    if (!folder || settings.get('enabled') === false || !vscode.workspace.workspaceFile) return undefined;
    return isDanger(settings.get('danger') || [], folder.name, activeRelPath(folder));
  }
  let dangerFile;
  function updateSaveGuard() {
    const now = activeFileDanger() !== undefined;
    if (now !== dangerFile) {
      dangerFile = now;
      vscode.commands.executeCommand('setContext', 'hue.dangerFile', now);
    }
  }
  const confirmed = new Set();
  async function save() {
    const entry = activeFileDanger();
    if (entry !== undefined && !confirmed.has(entry)) {
      const answer = await vscode.window.showWarningMessage(`You are editing PROD: ${entry}. Save anyway?`,
        { modal: true, detail: 'Marked in hue.danger. Hue asks once per place until VS Code restarts.' }, 'Save');
      if (answer !== 'Save') return;
      confirmed.add(entry);
    }
    await vscode.commands.executeCommand('workbench.action.files.save');
  }

  // The folder a command acts on: the active editor's, else ask when there are several
  async function commandFolder() {
    const folders = vscode.workspace.workspaceFolders || [];
    if (!vscode.workspace.workspaceFile || !folders.length) {
      vscode.window.showInformationMessage('Hue: open a .code-workspace first.');
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
    const next = { ...(vscode.workspace.getConfiguration('hue').get('folders') || {}) };
    if (value === 'auto') delete next[name]; else next[name] = value;
    // The configuration change re-applies the colors
    await vscode.workspace.getConfiguration('hue').update('folders', next, vscode.ConfigurationTarget.Workspace);
  }

  // Mark or unmark the whole folder, the active file's directory, or an existing entry
  async function toggleDanger() {
    const folder = await commandFolder();
    if (!folder) return;
    const settings = vscode.workspace.getConfiguration('hue');
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

  // A terminal in the folder, its tab in the folder's color (red with a warning in a danger place)
  async function openTerminal() {
    const folder = await commandFolder();
    if (!folder) return;
    const settings = vscode.workspace.getConfiguration('hue');
    const enabled = settings.get('enabled') !== false;
    const hue = folderHue(folder.name, { folders: settings.get('folders') || {}, colors: colorsSetting(), auto: settings.get('auto') !== false, enabled });
    const danger = enabled && isDanger(settings.get('danger') || [], folder.name, activeRelPath(folder)) !== undefined;
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
      vscode.window.showInformationMessage('Hue: open a .code-workspace first.');
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
    forced = { name: pick.folder.name };
    await vscode.commands.executeCommand('revealInExplorer', pick.folder.uri);
    apply();
  }

  // Take the colors out of the workspace and keep the extension off here
  async function clear() {
    await vscode.workspace.getConfiguration('hue').update('enabled', false, vscode.ConfigurationTarget.Workspace);
    vscode.window.showInformationMessage('Hue: colors removed. Set hue.enabled to true to turn it back on.');
  }

  context.subscriptions.push(
    vscode.window.onDidChangeActiveTextEditor(() => { forced = undefined; rememberFile(); updateSaveGuard(); apply(); }),
    // A terminal in another folder takes the window there; one with no folder leaves it
    vscode.window.onDidChangeActiveTerminal((terminal) => {
      const place = terminal && terminalPlace(terminal);
      if (place) { forced = place; apply(); }
    }),
    vscode.window.onDidChangeActiveColorTheme(apply),
    vscode.workspace.onDidChangeWorkspaceFolders(apply),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration('hue')) { updateSaveGuard(); apply(); }
    }),
    vscode.commands.registerCommand('hue.setColor', setColor),
    vscode.commands.registerCommand('hue.toggleDanger', toggleDanger),
    vscode.commands.registerCommand('hue.openTerminal', openTerminal),
    vscode.commands.registerCommand('hue.goToFolder', goToFolder),
    vscode.commands.registerCommand('hue.clear', clear),
    vscode.commands.registerCommand('hue.refresh', apply),
    vscode.commands.registerCommand('hue.save', save),
  );
  rememberFile();
  updateSaveGuard();
  apply();
}

module.exports = { activate, deactivate() {} };
