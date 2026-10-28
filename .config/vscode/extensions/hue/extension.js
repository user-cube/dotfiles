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
const crypto = require('crypto');
const fs = require('fs');

const {
  DEFAULT_COLORS, STRENGTH, PARTS, hueOfHex, folderHue, ansiFor, isDanger, decorationOf, nextCustomizations, readonlyGlobs, nextReadonly,
} = require('./colors');
const { panelState, applyMessage, panelHtml } = require('./panel');
const { mainItems, colorItems, settingsItems, sidebarItems, settingsTree, swatchOf, swatchSvg } = require('./search');

function activate(context) {
  function isLight() {
    const kind = vscode.window.activeColorTheme.kind;
    return kind === vscode.ColorThemeKind.Light || kind === vscode.ColorThemeKind.HighContrastLight;
  }

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
  // The folder you are in, a dot in its color; a click opens Hue: Search
  const folderItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 9999);
  folderItem.command = 'hue.search';
  context.subscriptions.push(folderItem);
  const viewChanged = new vscode.EventEmitter();
  // created here: apply sets its badge
  let folderView;
  context.subscriptions.push(viewChanged);

  // Each apply waits for git (the branch); one started later wins, so the colors
  // of a folder you just left never land after the ones of the folder you are in
  let latest = 0;
  async function apply() {
    const run = ++latest;
    viewChanged.fire(); // whatever moved the window may have changed the view's lines
    // Colors land in the .code-workspace file; never write a repo's .vscode/settings.json
    if (!vscode.workspace.workspaceFile) { dangerItem.hide(); folderItem.hide(); folderView.badge = undefined; return; }
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
    const light = isLight();
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
    if (enabled) {
      folderItem.text = `$(${hue === undefined ? 'circle-large-outline' : 'circle-filled'}) ${name}`;
      folderItem.color = swatchOf(hue, light);
      folderItem.tooltip = `Hue: you are in ${name}. Click for Hue: Search`;
      folderItem.show();
    } else folderItem.hide();
    // the same warning on Hue's activity bar icon, seen with the sidebar closed
    folderView.badge = danger ? { value: 1, tooltip: `PROD: ${dangerEntry}` } : undefined;

    const cfg = vscode.workspace.getConfiguration('workbench');
    const current = { ...((cfg.inspect('colorCustomizations') || {}).workspaceValue || {}) };
    const next = nextCustomizations(current, { hue, light, k, parts: settings.get('parts') || PARTS, onProtected, danger });
    if (JSON.stringify(next) === JSON.stringify(current)) return;
    await cfg.update('colorCustomizations', Object.keys(next).length ? next : undefined, vscode.ConfigurationTarget.Workspace);
  }

  // hue.dangerReadonly: the danger places open read-only (files.readonlyInclude,
  // in the workspace), which also stops the saves ⌘S can't: the menu, auto save.
  // "File: Set Active Editor Writeable in Session" opens one up. Hue keeps the
  // globs it added in workspaceState, so it takes out only those
  async function applyReadonly() {
    if (!vscode.workspace.workspaceFile) return;
    const settings = vscode.workspace.getConfiguration('hue');
    const on = settings.get('enabled') !== false && settings.get('dangerReadonly') === true;
    const folders = (vscode.workspace.workspaceFolders || []).map((f) => ({ name: f.name, fsPath: f.uri.fsPath }));
    const globs = on ? readonlyGlobs(settings.get('danger') || [], folders) : [];
    const owned = context.workspaceState.get('readonlyGlobs') || [];
    const files = vscode.workspace.getConfiguration('files');
    const current = (files.inspect('readonlyInclude') || {}).workspaceValue || {};
    const { next, owned: nowOwned } = nextReadonly(current, owned, globs);
    if (JSON.stringify(nowOwned) !== JSON.stringify(owned)) context.workspaceState.update('readonlyGlobs', nowOwned);
    if (JSON.stringify(next) === JSON.stringify(current)) return;
    await files.update('readonlyInclude', Object.keys(next).length ? next : undefined, vscode.ConfigurationTarget.Workspace);
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

  // Hue: Open, the page (panel.js): one at a time, redrawn when the settings or
  // the theme change. A message from it goes through applyMessage, which checks
  // it; a PROD path comes from the folder picker, never from the page itself
  let panel;
  const SETTING_KEYS = ['folders', 'danger', 'auto', 'enabled', 'dangerReadonly', 'reuseTerminal', 'strength', 'parts', 'protectedBranches'];
  function hueSettings() {
    const settings = vscode.workspace.getConfiguration('hue');
    return Object.fromEntries(SETTING_KEYS.map((key) => [key, settings.get(key)]));
  }
  function postState() {
    if (!panel) return;
    if (!vscode.workspace.workspaceFile) {
      panel.webview.postMessage({ type: 'state', state: { noWorkspace: true } });
      return;
    }
    const light = isLight();
    const names = (vscode.workspace.workspaceFolders || []).map((f) => f.name);
    panel.webview.postMessage({ type: 'state', state: panelState(names, hueSettings(), colorsSetting(), light) });
  }
  async function onPanelMessage(message) {
    if (!message || typeof message !== 'object') return;
    if (message.type === 'ready') { postState(); return; }
    if (!vscode.workspace.workspaceFile || message.type === 'addProdPath') return;
    const folders = vscode.workspace.workspaceFolders || [];
    if (message.type === 'pickProdPath') {
      const folder = folders.find((f) => f.name === message.folder);
      const rel = folder && await pickProdPath(folder);
      if (!rel) return;
      message = { type: 'addProdPath', folder: folder.name, rel };
    }
    const change = applyMessage(message, hueSettings(), folders.map((f) => f.name), colorsSetting());
    if (change) await vscode.workspace.getConfiguration('hue').update(change.key, change.value, vscode.ConfigurationTarget.Workspace);
  }
  // Add path…: a path inside the folder from the folder picker, or undefined
  async function pickProdPath(folder) {
    const picked = await vscode.window.showOpenDialog({
      defaultUri: folder.uri, canSelectFolders: true, canSelectFiles: true, canSelectMany: false, openLabel: 'Mark as PROD',
    });
    if (!picked || !picked.length) return undefined;
    const rel = relPathIn(folder, picked[0]);
    if (!rel) vscode.window.showWarningMessage(`Hue: pick a path inside ${folder.name} (for all of it, tick PROD).`);
    return rel;
  }

  function openPanel() {
    if (panel) { panel.reveal(); return; }
    panel = vscode.window.createWebviewPanel('hue', 'Hue', vscode.ViewColumn.Active, { enableScripts: true, localResourceRoots: [] });
    panel.webview.html = panelHtml(crypto.randomBytes(16).toString('hex'));
    const messages = panel.webview.onDidReceiveMessage(onPanelMessage);
    panel.onDidDispose(() => { messages.dispose(); panel = undefined; });
  }

  // The folder a command acts on: the one of the Hue view's line it came from,
  // else the active editor's, else ask when there are several
  async function commandFolder(line) {
    const own = line && line.folder && (vscode.workspace.workspaceFolders || []).find((f) => f.name === line.folder);
    if (own) return own;
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
  async function setColor(line) {
    const folder = await commandFolder(line);
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

  // Mark or unmark the whole folder, the active file's directory, or an existing
  // entry. From a line of the Hue view: flips the whole folder, no questions
  async function toggleDanger(line) {
    const folder = await commandFolder(line);
    if (!folder) return;
    const settings = vscode.workspace.getConfiguration('hue');
    const list = settings.get('danger') || [];
    if (line && line.folder) {
      await writeChange({ type: 'setProd', folder: folder.name, on: !list.includes(folder.name) });
      return;
    }
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
  async function openTerminal(line) {
    const folder = await commandFolder(line);
    if (folder) openTerminalIn(folder);
  }
  // hue.reuseTerminal: the one Hue opened in the folder shows again while it is
  // open, unless the folder became PROD (or stopped being) since: its tab says so
  const folderTerminals = new Map(); // folder name -> { terminal, danger }
  function openTerminalIn(folder) {
    const settings = vscode.workspace.getConfiguration('hue');
    const enabled = settings.get('enabled') !== false;
    const hue = folderHue(folder.name, { folders: settings.get('folders') || {}, colors: colorsSetting(), auto: settings.get('auto') !== false, enabled });
    const danger = enabled && isDanger(settings.get('danger') || [], folder.name, activeRelPath(folder)) !== undefined;
    const open = folderTerminals.get(folder.name);
    if (settings.get('reuseTerminal') !== false && open && open.danger === danger && open.terminal.exitStatus === undefined) {
      open.terminal.show();
      return;
    }
    const red = new vscode.ThemeColor('terminal.ansiRed');
    const terminal = vscode.window.createTerminal({
      name: danger ? `⚠ ${folder.name}` : folder.name,
      cwd: folder.uri,
      color: danger ? red : hue === undefined ? undefined : new vscode.ThemeColor(ansiFor(hue)),
      iconPath: danger ? new vscode.ThemeIcon('warning', red) : new vscode.ThemeIcon('terminal'),
    });
    folderTerminals.set(folder.name, { terminal, danger });
    terminal.show();
  }

  // Quick pick of the workspace folders: opens the last file you had there, or
  // reveals the folder. A line of the Hue view goes straight to its folder
  async function goToFolder(line) {
    const folders = vscode.workspace.workspaceFolders || [];
    const own = line && line.folder && folders.find((f) => f.name === line.folder);
    if (own) { await goTo(own); return; }
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
    if (pick) await goTo(pick.folder);
  }

  // Opens the last file you had in the folder, or reveals the folder
  async function goTo(folder) {
    const files = lastFiles();
    if (files[folder.name]) {
      try {
        const doc = await vscode.workspace.openTextDocument(vscode.Uri.parse(files[folder.name]));
        await vscode.window.showTextDocument(doc);
        return;
      } catch (e) { /* the file is gone: reveal the folder instead */ }
    }
    forced = { name: folder.name };
    await vscode.commands.executeCommand('revealInExplorer', folder.uri);
    apply();
  }

  // Hue: Search (⌘K H): a menu you type to filter (search.js). The top one
  // has a line per folder, with buttons; Color… and Settings… open in the same
  // box, with Back. A setting changes through applyMessage, as from the page;
  // the settings menu stays open so you can flip several
  function iconOf(icon) {
    if (!icon) return undefined;
    if (typeof icon === 'string') return new vscode.ThemeIcon(icon);
    if (icon.prod) return new vscode.ThemeIcon('warning', new vscode.ThemeColor('errorForeground'));
    // a dot in the color: an svg in the extension's storage, written once
    const file = path.join(context.globalStorageUri.fsPath, 'swatches', `${icon.swatch.slice(1).toLowerCase()}.svg`);
    if (!fs.existsSync(file)) {
      fs.mkdirSync(path.dirname(file), { recursive: true });
      fs.writeFileSync(file, swatchSvg(icon.swatch));
    }
    return vscode.Uri.file(file);
  }
  function toQuickPick(items) {
    return items.map((item) => (item.separator
      ? { label: item.separator, kind: vscode.QuickPickItemKind.Separator }
      : { ...item, iconPath: iconOf(item.icon),
          buttons: (item.buttons || []).map((b) => ({ ...b, iconPath: new vscode.ThemeIcon(b.icon) })) }));
  }
  async function writeChange(message) {
    const folders = vscode.workspace.workspaceFolders || [];
    const change = applyMessage(message, hueSettings(), folders.map((f) => f.name), colorsSetting());
    if (change) await vscode.workspace.getConfiguration('hue').update(change.key, change.value, vscode.ConfigurationTarget.Workspace);
  }

  async function editBranches() {
    const value = await vscode.window.showInputBox({ title: 'Hue: protected branches', prompt: 'Branch names, separated by commas',
      value: (hueSettings().protectedBranches || ['main', 'master']).join(', ') });
    if (value !== undefined) await writeChange({ type: 'setSetting', key: 'protectedBranches', value: value.split(',') });
  }

  async function search() {
    const folders = vscode.workspace.workspaceFolders || [];
    if (!vscode.workspace.workspaceFile || !folders.length) {
      vscode.window.showInformationMessage('Hue: open a .code-workspace first.');
      return;
    }
    const light = isLight();
    const folderOf = (name) => folders.find((f) => f.name === name);
    const qp = vscode.window.createQuickPick();
    qp.matchOnDescription = true;
    let menu = { name: 'main' };
    function show(next = menu) {
      menu = next;
      const state = { folders: folders.map((f) => f.name), active: activeFolderName(), settings: hueSettings(), colors: colorsSetting(), light };
      if (menu.name === 'color') {
        qp.title = `Hue: color of ${menu.folder}`;
        qp.placeholder = 'A color, Automatic, or Custom…';
        qp.items = toQuickPick(colorItems({ ...state, folder: menu.folder }));
      } else if (menu.name === 'settings') {
        qp.title = 'Hue: settings';
        qp.placeholder = 'Enter flips a switch; Back or Escape when done';
        qp.items = toQuickPick(settingsItems(state));
      } else {
        qp.title = undefined;
        qp.placeholder = 'Hue: a folder, a color, PROD or a setting';
        qp.items = toQuickPick(mainItems(state));
      }
      qp.buttons = menu.name === 'main' ? [] : [vscode.QuickInputButtons.Back];
      qp.value = '';
    }
    async function act(item) {
      const folder = item.folder && folderOf(item.folder);
      if (item.run === 'colorMenu') return show({ name: 'color', folder: item.folder });
      if (item.run === 'settingsMenu') return show({ name: 'settings' });
      if (item.message) {
        await writeChange(item.message);
        if (menu.name === 'settings') return show(); // keep it open: flip another
        return qp.hide();
      }
      qp.hide();
      if (item.run === 'open') openPanel();
      else if (item.run === 'goTo' && folder) await goTo(folder);
      else if (item.run === 'terminal' && folder) openTerminalIn(folder);
      else if (item.run === 'toggleDanger') await toggleDanger();
      else if (item.run === 'customColor') {
        const value = await vscode.window.showInputBox({ title: `Hue: color of ${item.folder}`, prompt: 'A color as #rrggbb', placeHolder: '#3366ff',
          validateInput: (v) => (hueOfHex(v.trim()) === undefined ? 'Use #rrggbb' : undefined) });
        if (value) await writeChange({ type: 'setColor', folder: item.folder, value: value.trim() });
      } else if (item.run === 'branches') await editBranches();
    }
    qp.onDidAccept(() => { const [item] = qp.selectedItems; if (item) act(item); });
    qp.onDidTriggerButton(() => show({ name: 'main' })); // Back: the only title button
    qp.onDidTriggerItemButton(async ({ item, button }) => {
      if (button.id === 'terminal') return act({ run: 'terminal', folder: item.folder });
      if (button.id === 'color') return show({ name: 'color', folder: item.folder });
      if (button.id === 'prod') {
        await writeChange({ type: 'setProd', folder: item.folder, on: !(hueSettings().danger || []).includes(item.folder) });
        return show(); // stays open, the line now says PROD (or not)
      }
      return undefined;
    });
    qp.onDidHide(() => qp.dispose());
    show();
    qp.show();
  }

  // The Hue views in the activity bar. Folders (sidebarItems): a line per
  // folder, a click goes there and its buttons (package.json,
  // view/item/context) run the commands above with the line; under it its PROD
  // paths (a click reveals one, ✕ takes it out) and Add PROD path…. Settings
  // (settingsTree): a click flips a switch, as in Search. Both are redrawn by
  // every apply, and are empty outside a .code-workspace, where package.json's
  // viewsWelcome says so
  const hasGlob = (rel) => /[*?[]/.test(rel);
  function treeItem(line, id) {
    const item = new vscode.TreeItem(line.label);
    item.id = id;
    item.description = line.description;
    item.iconPath = iconOf(line.icon);
    return item;
  }
  const folderTree = {
    onDidChangeTreeData: viewChanged.event,
    async getChildren(parent) {
      if (parent) return parent.children || [];
      if (!vscode.workspace.workspaceFile) return [];
      const folders = vscode.workspace.workspaceFolders || [];
      const branches = {};
      for (const folder of folders) branches[folder.name] = await branchOf(folder);
      return sidebarItems({ folders: folders.map((f) => f.name), active: activeFolderName(), settings: hueSettings(), colors: colorsSetting(), light: isLight(), branches });
    },
    getTreeItem(line) {
      if (!line.folder) {
        const item = treeItem(line, 'page');
        item.command = { command: 'hue.open', title: 'Open' };
        return item;
      }
      if (line.run === 'addProdPath') {
        const item = treeItem(line, `${line.folder}/+`);
        item.command = { command: 'hue.addProdPath', title: 'Add PROD path', arguments: [line] };
        return item;
      }
      if (line.rel !== undefined) {
        const item = treeItem(line, `${line.folder}/${line.rel}`);
        item.contextValue = 'prodPath';
        item.tooltip = `PROD: ${line.folder}/${line.rel}`;
        if (!hasGlob(line.rel)) {
          const folder = (vscode.workspace.workspaceFolders || []).find((f) => f.name === line.folder);
          if (folder) item.command = { command: 'revealInExplorer', title: 'Reveal', arguments: [vscode.Uri.file(path.join(folder.uri.fsPath, line.rel))] };
        }
        return item;
      }
      const item = treeItem(line, line.folder);
      item.contextValue = 'folder';
      item.tooltip = `Go to ${line.folder}${line.protectedBranch ? ` (on ${line.branch}, a protected branch)` : ''}`;
      // the PROD paths open, the rest closed; VS Code keeps what you change (by id)
      item.collapsibleState = line.children.length > 1 ? vscode.TreeItemCollapsibleState.Expanded : vscode.TreeItemCollapsibleState.Collapsed;
      item.command = { command: 'hue.goToFolder', title: 'Go to Folder', arguments: [line] };
      // on a protected branch: the label in amber (branchColors below)
      if (line.protectedBranch) item.resourceUri = vscode.Uri.from({ scheme: 'hue', path: `/${line.folder}` });
      return item;
    },
  };
  // The decorations: a folder line on a protected branch (above), and in the
  // Explorer and the tabs (decorationOf) P in red inside a PROD place and each
  // workspace folder in its color. Asked again when Hue's settings change
  const decorationsChanged = new vscode.EventEmitter();
  context.subscriptions.push(decorationsChanged);
  const decorations = {
    onDidChangeFileDecorations: decorationsChanged.event,
    provideFileDecoration(uri) {
      if (uri.scheme === 'hue') return { color: new vscode.ThemeColor('list.warningForeground'), tooltip: 'protected branch' };
      const folder = uri.scheme === 'file' && vscode.workspace.workspaceFile && vscode.workspace.getWorkspaceFolder(uri);
      const settings = vscode.workspace.getConfiguration('hue');
      if (!folder || settings.get('enabled') === false) return undefined;
      const hue = folderHue(folder.name, { folders: settings.get('folders') || {}, colors: colorsSetting(), auto: settings.get('auto') !== false });
      const found = decorationOf({ danger: settings.get('danger') || [], folderName: folder.name, rel: relPathIn(folder, uri), hue });
      return found && { ...found, color: new vscode.ThemeColor(found.color) };
    },
  };
  const settingsView = {
    onDidChangeTreeData: viewChanged.event,
    getChildren(parent) {
      if (parent) return parent.children || [];
      return vscode.workspace.workspaceFile ? settingsTree({ settings: hueSettings() }) : [];
    },
    getTreeItem(line) {
      const item = treeItem(line, line.children ? line.label : undefined);
      if (line.children) item.collapsibleState = vscode.TreeItemCollapsibleState.Expanded;
      else item.command = { command: 'hue.setting', title: 'Change', arguments: [line] };
      return item;
    },
  };
  async function setting(line) {
    if (line && line.message) await writeChange(line.message);
    else if (line && line.run === 'branches') await editBranches();
  }
  async function addProdPath(line) {
    const folder = line && (vscode.workspace.workspaceFolders || []).find((f) => f.name === line.folder);
    const rel = folder && await pickProdPath(folder);
    if (rel) await writeChange({ type: 'addProdPath', folder: folder.name, rel });
  }
  async function removeProdPath(line) {
    if (line && line.rel !== undefined) await writeChange({ type: 'removeProdPath', folder: line.folder, rel: line.rel });
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
    vscode.window.onDidChangeActiveColorTheme(() => { apply(); postState(); }),
    vscode.workspace.onDidChangeWorkspaceFolders(() => { applyReadonly(); apply(); postState(); decorationsChanged.fire(); }),
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration('hue')) { updateSaveGuard(); applyReadonly(); apply(); postState(); decorationsChanged.fire(); }
    }),
    vscode.commands.registerCommand('hue.setColor', setColor),
    vscode.commands.registerCommand('hue.toggleDanger', toggleDanger),
    vscode.commands.registerCommand('hue.openTerminal', openTerminal),
    vscode.commands.registerCommand('hue.goToFolder', goToFolder),
    vscode.commands.registerCommand('hue.clear', clear),
    vscode.commands.registerCommand('hue.refresh', apply),
    vscode.commands.registerCommand('hue.save', save),
    vscode.commands.registerCommand('hue.open', openPanel),
    vscode.commands.registerCommand('hue.search', search),
    vscode.commands.registerCommand('hue.removeProdPath', removeProdPath),
    // a click on a line of the views, not in the command palette
    vscode.commands.registerCommand('hue.addProdPath', addProdPath),
    vscode.commands.registerCommand('hue.setting', setting),
    vscode.window.registerTreeDataProvider('hue.settings', settingsView),
    vscode.window.registerFileDecorationProvider(decorations),
    vscode.window.onDidCloseTerminal((terminal) => {
      for (const [name, open] of folderTerminals) if (open.terminal === terminal) folderTerminals.delete(name);
    }),
  );
  folderView = vscode.window.createTreeView('hue.folders', { treeDataProvider: folderTree });
  context.subscriptions.push(folderView);
  rememberFile();
  updateSaveGuard();
  applyReadonly();
  apply();
}

module.exports = { activate, deactivate() {} };
