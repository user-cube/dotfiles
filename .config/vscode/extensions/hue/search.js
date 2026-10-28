// Hue: Search (⌘K H): a menu you type to filter, like the command palette.
// The top menu has one line per folder (its color as the icon, ⚠ for PROD;
// Enter goes there, its buttons open a terminal, pick its color or flip PROD)
// and Hue's own entries; Color… and Settings… are menus of their own, with a
// way back. Without VS Code: the items of each menu (mainItems, colorItems,
// settingsItems) and the icon of a color (swatchSvg). An item changes a
// setting (message: what the page would post, checked by applyMessage) or
// runs something (run). icon: a codicon name, { swatch: '#rrggbb' } or
// { prod: true }. extension.js shows the menus; colors.test.js tests this.
const { STRENGTH, hueOf, folderHue, palette } = require('./colors');
const { WINDOW_PARTS } = require('./panel');

// The bright color of a hue, the one the swatches show
const swatchOf = (hue, light) => (hue === undefined ? undefined : palette(hue, light, 1).activityBar['activityBar.activeBorder']);

// A 16px dot in that color, for a quick pick icon. Only a #rrggbb gets in
function swatchSvg(hex) {
  if (!/^#[0-9a-f]{6}$/i.test(hex)) throw new Error(`not a #rrggbb: ${hex}`);
  return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="6" fill="${hex}"/></svg>`;
}

// folders: the workspace folders' names; active: the folder you are in (or
// undefined); settings: Hue's (hue.*); colors: the named colors in use
function mainItems({ folders, active, settings, colors, light = false }) {
  const own = settings.folders || {};
  const danger = settings.danger || [];
  const auto = settings.auto !== false;
  const items = [{ separator: 'Folders' }];
  for (const name of folders) {
    const prod = danger.includes(name);
    const paths = danger.filter((e) => e.startsWith(`${name}/`)).length;
    const hue = folderHue(name, { folders: own, colors, auto });
    const color = own[name] !== undefined ? own[name] : hue === undefined ? 'no color' : 'auto';
    items.push({
      label: name,
      description: [prod ? 'PROD' : color, paths ? `${paths} PROD path${paths > 1 ? 's' : ''}` : '', name === active ? 'you are here' : '']
        .filter(Boolean).join(' · '),
      icon: prod ? { prod: true } : hue === undefined ? 'circle-large-outline' : { swatch: swatchOf(hue, light) },
      run: 'goTo',
      folder: name,
      buttons: [
        { id: 'terminal', icon: 'terminal', tooltip: `Terminal in ${name}` },
        { id: 'color', icon: 'symbol-color', tooltip: `Color of ${name}…` },
        { id: 'prod', icon: prod ? 'shield' : 'warning', tooltip: prod ? `Unmark ${name} as PROD` : `Mark ${name} as PROD` },
      ],
    });
  }
  items.push({ separator: 'Hue' });
  if (active !== undefined && folders.includes(active)) {
    items.push({ label: `Color of ${active}…`, icon: 'symbol-color', run: 'colorMenu', folder: active });
  }
  items.push({ label: 'PROD for this file\'s place…', description: 'the folder, the file\'s directory, or an entry', icon: 'warning', run: 'toggleDanger' });
  items.push({ label: 'Settings…', description: settingsSummary(settings), icon: 'settings-gear', run: 'settingsMenu' });
  items.push({ label: 'Open the Hue page', description: 'folders, colors, PROD and settings on one page', icon: 'window', run: 'open' });
  return items;
}

// "on · normal · read-only": the settings at a glance
function settingsSummary(settings) {
  const strength = STRENGTH[settings.strength] ? settings.strength : 'normal';
  return [settings.enabled === false ? 'off' : 'on', strength, settings.dangerReadonly === true ? 'PROD read-only' : '']
    .filter(Boolean).join(' · ');
}

// Color of one folder: the named colors as dots, Automatic, a custom #rrggbb
function colorItems({ folder, settings, colors, light = false }) {
  const current = (settings.folders || {})[folder];
  const items = [{ separator: `Color of ${folder}` }];
  for (const name of Object.keys(colors)) {
    items.push({ label: name, description: current === name ? 'current' : undefined, icon: { swatch: swatchOf(hueOf(name, colors), light) },
      message: { type: 'setColor', folder, value: name } });
  }
  const custom = typeof current === 'string' && current.startsWith('#');
  items.push({ label: custom ? `Custom ${current}` : 'Custom…', description: custom ? 'current' : '#rrggbb',
    icon: custom && hueOf(current, colors) !== undefined ? { swatch: current.toLowerCase() } : 'symbol-color', run: 'customColor', folder });
  items.push({ label: 'Automatic', description: current === undefined ? 'current' : 'from the name', icon: 'sync',
    message: { type: 'setColor', folder, value: null } });
  return items;
}

// The settings as switches (a check when on), the strength and the parts
function settingsItems({ settings }) {
  const check = (on) => (on ? 'check' : 'circle-large-outline');
  const enabled = settings.enabled !== false;
  const auto = settings.auto !== false;
  const readonly = settings.dangerReadonly === true;
  const strength = STRENGTH[settings.strength] ? settings.strength : 'normal';
  const parts = settings.parts || WINDOW_PARTS;
  const branches = settings.protectedBranches || ['main', 'master'];
  const items = [
    { separator: 'Settings' },
    { label: 'Hue in this workspace', description: enabled ? 'on' : 'off: no colors', icon: check(enabled),
      message: { type: 'setSetting', key: 'enabled', value: !enabled } },
    { label: 'Automatic colors', description: 'a folder with no color gets one from its name', icon: check(auto),
      message: { type: 'setSetting', key: 'auto', value: !auto } },
    { label: 'PROD opens read-only', description: 'also stops saves from the menu and auto save', icon: check(readonly),
      message: { type: 'setSetting', key: 'dangerReadonly', value: !readonly } },
    { separator: 'Strength' },
    ...Object.keys(STRENGTH).map((value) => ({ label: value, description: value === strength ? 'current' : undefined,
      icon: value === strength ? 'circle-filled' : 'circle-outline', message: { type: 'setSetting', key: 'strength', value } })),
    { separator: 'Tinted parts' },
    ...WINDOW_PARTS.map((part) => ({ label: part, icon: check(parts.includes(part)),
      message: { type: 'setSetting', key: 'parts', value: parts.includes(part) ? parts.filter((p) => p !== part) : [...parts, part] } })),
    { separator: 'Protected branches' },
    { label: branches.length ? branches.join(', ') : 'none', description: 'the status bar turns amber on these: edit…', icon: 'git-branch',
      run: 'branches' },
  ];
  return items;
}

module.exports = { mainItems, colorItems, settingsItems, swatchSvg };
