// Hue's page (Hue: Open): a webview to pick the folders' colors, mark the PROD
// places and turn the settings on and off. Without VS Code: what the page shows
// (panelState), the setting a message from it changes (applyMessage, which
// trusts nothing the page sends) and the page itself (panelHtml).
// extension.js opens it and writes the settings; colors.test.js tests this.
const { STRENGTH, PARTS, hueOfHex, hueOf, folderHue, palette } = require('./colors');

const SWITCHES = ['enabled', 'auto', 'dangerReadonly', 'reuseTerminal'];
const WINDOW_PARTS = PARTS.filter((p) => p !== 'tabs');

// What the page shows. folders: the workspace folders' names; settings: Hue's
// (hue.*, as VS Code returns them); colors: the named colors in use
function panelState(folders, settings, colors, light = false) {
  const own = settings.folders || {};
  const danger = settings.danger || [];
  const auto = settings.auto !== false;
  // preview: the title bar in that color; swatch: the color itself, bright (the accent)
  const bar = (h) => (h === undefined ? undefined : palette(h, light, STRENGTH[settings.strength] || 1).titleBar['titleBar.activeBackground']);
  const swatch = (h) => (h === undefined ? undefined : palette(h, light, 1).activityBar['activityBar.activeBorder']);
  return {
    folders: folders.map((name) => {
      const hue = folderHue(name, { folders: own, colors, auto });
      return {
        name,
        color: own[name] === undefined ? null : own[name],
        preview: bar(hue),
        swatch: swatch(hue),
        prod: danger.includes(name),
        prodPaths: danger.filter((e) => e.startsWith(`${name}/`)).map((e) => e.slice(name.length + 1)),
      };
    }),
    colors: Object.keys(colors).map((name) => ({ name, preview: bar(hueOf(name, colors)), swatch: swatch(hueOf(name, colors)) })),
    light,
    // the title bar of a PROD place (colors.js DANGER)
    prodBar: light ? '#f5b5b5' : '#8a1c1c',
    barText: light ? '#1f2328' : '#ffffffee',
    settings: {
      enabled: settings.enabled !== false,
      auto,
      dangerReadonly: settings.dangerReadonly === true,
      reuseTerminal: settings.reuseTerminal !== false,
      strength: STRENGTH[settings.strength] ? settings.strength : 'normal',
      parts: settings.parts || WINDOW_PARTS,
      protectedBranches: settings.protectedBranches || ['main', 'master'],
    },
    strengths: Object.keys(STRENGTH),
    parts: WINDOW_PARTS,
  };
}

// The hue.* setting a message from the page changes, as { key, value }, or
// undefined when the message is not one Hue knows, or names a folder that
// isn't in the workspace, or carries a value the setting can't take
function applyMessage(message, settings, folders, colors) {
  if (!message || typeof message !== 'object') return undefined;
  const isFolder = (f) => typeof f === 'string' && folders.includes(f);
  const danger = settings.danger || [];
  switch (message.type) {
    case 'setColor': {
      if (!isFolder(message.folder)) return undefined;
      const value = message.value;
      const next = { ...(settings.folders || {}) };
      if (value === null) delete next[message.folder]; // Automatic
      else if (typeof value === 'string' && (value in colors || hueOfHex(value) !== undefined)) next[message.folder] = value;
      else return undefined;
      return { key: 'folders', value: next };
    }
    case 'setProd': {
      if (!isFolder(message.folder) || typeof message.on !== 'boolean') return undefined;
      const rest = danger.filter((e) => e !== message.folder);
      return { key: 'danger', value: message.on ? [...rest, message.folder] : rest };
    }
    case 'addProdPath': {
      // rel: a path inside the folder (from the folder picker), posix, no ..
      const rel = typeof message.rel === 'string' ? message.rel.replace(/^\/+|\/+$/g, '') : '';
      if (!isFolder(message.folder) || !rel || rel.split('/').some((p) => p === '..' || p === '.' || p === '')) return undefined;
      const entry = `${message.folder}/${rel}`;
      return { key: 'danger', value: danger.includes(entry) ? danger : [...danger, entry] };
    }
    case 'removeProdPath': {
      if (!isFolder(message.folder) || typeof message.rel !== 'string') return undefined;
      const entry = `${message.folder}/${message.rel}`;
      if (!danger.includes(entry)) return undefined;
      return { key: 'danger', value: danger.filter((e) => e !== entry) };
    }
    case 'setSetting': {
      const { key, value } = message;
      if (SWITCHES.includes(key)) return typeof value === 'boolean' ? { key, value } : undefined;
      if (key === 'strength') return value in STRENGTH ? { key, value } : undefined;
      if (key === 'parts') {
        if (!Array.isArray(value) || !value.every((p) => WINDOW_PARTS.includes(p))) return undefined;
        return { key, value: WINDOW_PARTS.filter((p) => value.includes(p)) };
      }
      if (key === 'protectedBranches') {
        if (!Array.isArray(value) || !value.every((b) => typeof b === 'string')) return undefined;
        return { key, value: [...new Set(value.map((b) => b.trim()).filter(Boolean))] };
      }
      return undefined;
    }
    default:
      return undefined;
  }
}

// The page: no remote content, the script and style by nonce only (CSP). It
// asks for the state (ready), draws it with DOM calls (a folder name is never
// HTML), and posts what you change; extension.js answers with the new state.
// Folders are cards (a title bar preview, the colors, PROD and its paths);
// the settings are switches, a segmented control and chips. VS Code's theme
// variables color it all
function panelHtml(nonce) {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'nonce-${nonce}'; script-src 'nonce-${nonce}';">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Hue</title>
<style nonce="${nonce}">
  :root {
    --fg: var(--vscode-foreground);
    --muted: var(--vscode-descriptionForeground);
    --card: var(--vscode-editorWidget-background, var(--vscode-sideBar-background));
    --line: var(--vscode-widget-border, var(--vscode-panel-border, #8883));
    --accent: var(--vscode-focusBorder);
    --danger: var(--vscode-errorForeground);
    --chip: var(--vscode-badge-background);
    --chip-fg: var(--vscode-badge-foreground);
    --radius: 10px;
  }
  * { box-sizing: border-box; }
  body { font-family: var(--vscode-font-family); font-size: var(--vscode-font-size); color: var(--fg); margin: 0; padding: 28px 32px 48px; }
  .page { max-width: 1100px; margin: 0 auto; }
  header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 28px; flex-wrap: wrap; }
  header h1 { font-size: 1.6em; font-weight: 600; margin: 0; letter-spacing: -0.01em; }
  header p { margin: 4px 0 0; color: var(--muted); }
  h2 { font-size: 0.8em; text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted); font-weight: 600; margin: 32px 0 12px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px; }
  .card { background: var(--card); border: 1px solid var(--line); border-radius: var(--radius); overflow: hidden; display: flex; flex-direction: column; }
  .card.prod { border-color: var(--danger); }
  .titlebar { height: 34px; display: flex; align-items: center; gap: 8px; padding: 0 12px; font-weight: 600; font-size: 0.95em; }
  .dots { display: flex; gap: 5px; margin-right: 4px; }
  .dots i { width: 9px; height: 9px; border-radius: 50%; background: currentColor; opacity: 0.35; }
  .titlebar .name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .badge { font-size: 0.75em; font-weight: 700; padding: 2px 7px; border-radius: 999px; background: #0004; letter-spacing: 0.04em; }
  .body { padding: 14px; display: flex; flex-direction: column; gap: 14px; }
  .label { font-size: 0.8em; color: var(--muted); margin-bottom: 6px; }
  .swatches { display: flex; flex-wrap: wrap; gap: 7px; align-items: center; }
  .sw { width: 24px; height: 24px; border-radius: 50%; border: 2px solid transparent; cursor: pointer; padding: 0; position: relative; outline-offset: 2px; }
  .sw:hover { transform: scale(1.12); }
  .sw.on { outline: 2px solid var(--fg); }
  .sw.auto, .sw.custom { background: transparent; border: 1px dashed var(--muted); color: var(--muted); font-size: 0.7em; font-weight: 700; display: grid; place-items: center; }
  .sw.custom.has { border: none; }
  .picker { position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none; }
  .row { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
  .row .text { display: flex; flex-direction: column; gap: 2px; }
  .row .hint { color: var(--muted); font-size: 0.85em; }
  .switch { position: relative; width: 34px; height: 20px; flex: none; cursor: pointer; }
  .switch input { opacity: 0; width: 0; height: 0; position: absolute; }
  .switch span { position: absolute; inset: 0; border-radius: 999px; background: var(--vscode-input-background); border: 1px solid var(--line); transition: background .15s; }
  .switch span::after { content: ''; position: absolute; width: 14px; height: 14px; left: 2px; top: 2px; border-radius: 50%; background: var(--muted); transition: transform .15s, background .15s; }
  .switch input:checked + span { background: var(--accent); border-color: var(--accent); }
  .switch input:checked + span::after { transform: translateX(14px); background: #fff; }
  .switch.danger input:checked + span { background: var(--danger); border-color: var(--danger); }
  .switch input:focus-visible + span { outline: 1px solid var(--accent); outline-offset: 2px; }
  .chips { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
  .chip { display: inline-flex; align-items: center; gap: 6px; padding: 3px 4px 3px 10px; border-radius: 999px; background: var(--chip); color: var(--chip-fg); font-size: 0.9em; }
  .chip.path { font-family: var(--vscode-editor-font-family); background: color-mix(in srgb, var(--danger) 18%, transparent); color: var(--fg); border: 1px solid color-mix(in srgb, var(--danger) 45%, transparent); }
  .chip button { background: none; border: none; color: inherit; cursor: pointer; padding: 0 5px; border-radius: 50%; font-size: 1.05em; line-height: 1; opacity: 0.7; }
  .chip button:hover { opacity: 1; background: #0003; }
  .add { background: none; border: 1px dashed var(--line); color: var(--muted); border-radius: 999px; padding: 3px 10px; cursor: pointer; font-size: 0.9em; }
  .add:hover { color: var(--fg); border-color: var(--fg); }
  .panel { background: var(--card); border: 1px solid var(--line); border-radius: var(--radius); }
  .panel > .row { padding: 14px 16px; border-bottom: 1px solid var(--line); }
  .panel > .row:last-child { border-bottom: none; }
  .seg { display: inline-flex; border: 1px solid var(--line); border-radius: 8px; overflow: hidden; }
  .seg button { background: none; border: none; color: var(--fg); padding: 5px 12px; cursor: pointer; }
  .seg button + button { border-left: 1px solid var(--line); }
  .seg button.on { background: var(--accent); color: #fff; }
  .toggle-chip { border: 1px solid var(--line); background: none; color: var(--muted); border-radius: 999px; padding: 4px 11px; cursor: pointer; }
  .toggle-chip.on { background: var(--chip); color: var(--chip-fg); border-color: transparent; }
  .tags input { background: transparent; border: none; color: var(--fg); outline: none; min-width: 110px; padding: 4px 2px; font: inherit; }
  .tags { border: 1px solid var(--line); border-radius: 8px; padding: 4px 6px; min-width: 260px; }
  .tags:focus-within { border-color: var(--accent); }
  .empty { color: var(--muted); padding: 40px 0; text-align: center; }
  .off .grid, .off .panel > .row:not(:first-child) { opacity: 0.45; }
</style>
</head>
<body>
<div class="page" id="app"><p class="empty">Loading…</p></div>
<script nonce="${nonce}">
(function () {
  const vscode = acquireVsCodeApi();
  const app = document.getElementById('app');
  const post = (message) => vscode.postMessage(message);
  function el(tag, props, ...children) {
    const node = document.createElement(tag);
    for (const [k, v] of Object.entries(props || {})) {
      if (v === undefined || v === null) continue;
      if (k === 'class') node.className = v;
      else if (k.startsWith('on')) node.addEventListener(k.slice(2), v);
      else if (k === 'style') node.style.cssText = v;
      else if (k === 'aria') for (const [a, b] of Object.entries(v)) node.setAttribute('aria-' + a, b);
      else node[k] = v;
    }
    for (const c of children) if (c != null && c !== false) node.append(c);
    return node;
  }
  const isHex = (c) => typeof c === 'string' && c.startsWith('#');

  function switchEl(checked, label, onChange, danger) {
    return el('label', { class: 'switch' + (danger ? ' danger' : ''), title: label },
      el('input', { type: 'checkbox', checked, aria: { label }, onchange: (e) => onChange(e.target.checked) }), el('span'));
  }

  function card(f, state) {
    const bar = f.prod ? state.prodBar : (f.preview || 'var(--vscode-titleBar-activeBackground)');
    const picker = el('input', { type: 'color', class: 'picker', value: isHex(f.color) ? f.color : '#3366ff', tabIndex: -1,
      onchange: (e) => post({ type: 'setColor', folder: f.name, value: e.target.value }) });
    const openPicker = () => { if (picker.showPicker) picker.showPicker(); else picker.click(); };
    const swatches = el('div', { class: 'swatches', role: 'radiogroup', aria: { label: 'Color of ' + f.name } },
      ...state.colors.map((c) => el('button', { class: 'sw' + (f.color === c.name ? ' on' : ''), title: c.name, style: 'background:' + c.swatch,
        aria: { label: c.name, checked: String(f.color === c.name) }, role: 'radio', onclick: () => post({ type: 'setColor', folder: f.name, value: c.name }) })),
      el('button', { class: 'sw custom' + (isHex(f.color) ? ' has on' : ''), title: isHex(f.color) ? 'Custom ' + f.color : 'Custom color…',
        style: isHex(f.color) ? 'background:' + f.color : '', textContent: isHex(f.color) ? '' : '+', role: 'radio',
        aria: { label: 'Custom color', checked: String(isHex(f.color)) }, onclick: openPicker }),
      el('button', { class: 'sw auto' + (f.color === null ? ' on' : ''), title: 'Automatic (from the name)', textContent: 'A', role: 'radio',
        aria: { label: 'Automatic', checked: String(f.color === null) }, onclick: () => post({ type: 'setColor', folder: f.name, value: null }) }),
      picker);
    const paths = el('div', { class: 'chips' },
      ...f.prodPaths.map((rel) => el('span', { class: 'chip path', title: f.name + '/' + rel }, '⚠ ' + rel,
        el('button', { textContent: '×', title: 'No longer PROD', aria: { label: 'Remove ' + rel }, onclick: () => post({ type: 'removeProdPath', folder: f.name, rel }) }))),
      el('button', { class: 'add', textContent: '+ Add path', title: 'Pick a directory or file inside ' + f.name, onclick: () => post({ type: 'pickProdPath', folder: f.name }) }));
    return el('div', { class: 'card' + (f.prod ? ' prod' : '') },
      el('div', { class: 'titlebar', style: 'background:' + bar + ';color:' + state.barText },
        el('span', { class: 'dots' }, el('i'), el('i'), el('i')),
        el('span', { class: 'name', textContent: f.name }),
        f.prod ? el('span', { class: 'badge', textContent: 'PROD' }) : null),
      el('div', { class: 'body' },
        el('div', {}, el('div', { class: 'label', textContent: 'Color' }), swatches),
        el('div', { class: 'row' },
          el('div', { class: 'text' }, el('span', { textContent: 'All of it is PROD' }), el('span', { class: 'hint', textContent: 'Red title bar, ⌘S asks first' })),
          switchEl(f.prod, 'All of ' + f.name + ' is PROD', (on) => post({ type: 'setProd', folder: f.name, on }), true)),
        el('div', {}, el('div', { class: 'label', textContent: 'PROD paths inside it' }), paths)));
  }

  function settingRow(title, hint, control) {
    return el('div', { class: 'row' }, el('div', { class: 'text' }, el('span', { textContent: title }), hint ? el('span', { class: 'hint', textContent: hint }) : null), control);
  }

  function branchesEl(branches) {
    const send = (list) => post({ type: 'setSetting', key: 'protectedBranches', value: list });
    const input = el('input', { placeholder: branches.length ? 'add…' : 'main', aria: { label: 'Add a protected branch' },
      onkeydown: (e) => {
        if ((e.key === 'Enter' || e.key === ',') && input.value.trim()) { e.preventDefault(); send([...branches, input.value]); }
        else if (e.key === 'Backspace' && !input.value && branches.length) send(branches.slice(0, -1));
      } });
    return el('div', { class: 'chips tags', onclick: () => input.focus() },
      ...branches.map((b) => el('span', { class: 'chip' }, b,
        el('button', { textContent: '×', aria: { label: 'Remove ' + b }, onclick: () => send(branches.filter((x) => x !== b)) }))),
      input);
  }

  function render(state) {
    if (state.noWorkspace) {
      app.replaceChildren(el('p', { class: 'empty', textContent: 'Hue works in a .code-workspace: open one to set its folders up here.' }));
      return;
    }
    const s = state.settings;
    const focused = document.activeElement && document.activeElement.matches('.tags input');
    app.className = 'page' + (s.enabled ? '' : ' off');
    app.replaceChildren(
      el('header', {},
        el('div', {}, el('h1', { textContent: 'Hue' }), el('p', { textContent: 'The window takes the color of the folder you work in. PROD places turn it red.' })),
        el('div', { class: 'row' }, el('span', { textContent: s.enabled ? 'On in this workspace' : 'Off in this workspace' }),
          switchEl(s.enabled, 'Hue on in this workspace', (v) => post({ type: 'setSetting', key: 'enabled', value: v })))),
      el('h2', { textContent: 'Folders' }),
      el('div', { class: 'grid' }, ...state.folders.map((f) => card(f, state))),
      el('h2', { textContent: 'Settings' }),
      el('div', { class: 'panel' },
        settingRow('Automatic colors', 'A folder with no color gets one from its name',
          switchEl(s.auto, 'Automatic colors', (v) => post({ type: 'setSetting', key: 'auto', value: v }))),
        settingRow('PROD opens read-only', 'Also stops saves from the menu and auto save',
          switchEl(s.dangerReadonly, 'PROD opens read-only', (v) => post({ type: 'setSetting', key: 'dangerReadonly', value: v }), true)),
        settingRow('One terminal per folder', 'The terminal button and ⌘⌥T show the one already open',
          switchEl(s.reuseTerminal, 'One terminal per folder', (v) => post({ type: 'setSetting', key: 'reuseTerminal', value: v }))),
        settingRow('Strength', 'How saturated the tint is',
          el('div', { class: 'seg', role: 'radiogroup', aria: { label: 'Strength' } }, ...state.strengths.map((v) => el('button', {
            class: v === s.strength ? 'on' : '', textContent: v, role: 'radio', aria: { checked: String(v === s.strength) },
            onclick: () => post({ type: 'setSetting', key: 'strength', value: v }) })))),
        settingRow('Tinted parts', 'Which parts of the window take the color',
          el('div', { class: 'chips' }, ...state.parts.map((p) => el('button', {
            class: 'toggle-chip' + (s.parts.includes(p) ? ' on' : ''), textContent: p, aria: { pressed: String(s.parts.includes(p)) },
            onclick: () => post({ type: 'setSetting', key: 'parts', value: s.parts.includes(p) ? s.parts.filter((x) => x !== p) : [...s.parts, p] }) })))),
        settingRow('Protected branches', 'The status bar turns amber on these', branchesEl(s.protectedBranches))));
    if (focused) { const input = app.querySelector('.tags input'); if (input) input.focus(); }
  }

  window.addEventListener('message', (event) => { if (event.data && event.data.type === 'state') render(event.data.state); });
  post({ type: 'ready' });
})();
</script>
</body>
</html>`;
}

module.exports = { panelState, applyMessage, panelHtml, WINDOW_PARTS };
