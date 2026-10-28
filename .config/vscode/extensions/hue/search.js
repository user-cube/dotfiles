// Hue: Search (⌘K H): everything Hue does in one list you type to filter, like
// the command palette. Without VS Code: the items for this state (searchItems).
// An item either changes a setting (message: what the page would post, checked
// by applyMessage like the page's) or runs something (run: open, goTo,
// terminal, toggleDanger). { separator } starts a group. extension.js shows
// the list and acts on the pick; colors.test.js tests this.
const { STRENGTH } = require('./colors');

// folders: the workspace folders' names; active: the folder you are in (or
// undefined); settings: Hue's (hue.*); colors: the named colors in use
function searchItems({ folders, active, settings, colors }) {
  const own = settings.folders || {};
  const danger = settings.danger || [];
  const on = (b) => (b ? 'on' : 'off');
  const items = [{ label: '$(window) Open Hue', description: 'the page: folders, colors, PROD, settings', run: 'open' }];

  items.push({ separator: 'Folders' });
  for (const name of folders) {
    items.push({ label: `$(arrow-right) Go to ${name}`, description: name === active ? 'you are here' : undefined, run: 'goTo', folder: name });
  }
  for (const name of folders) {
    items.push({ label: `$(terminal) Terminal in ${name}`, run: 'terminal', folder: name });
  }

  if (active !== undefined && folders.includes(active)) {
    items.push({ separator: `Color of ${active}` });
    for (const color of Object.keys(colors)) {
      items.push({ label: `$(circle-filled) ${color}`, description: own[active] === color ? 'current' : `color of ${active}`,
        message: { type: 'setColor', folder: active, value: color } });
    }
    items.push({ label: '$(sync) Automatic', description: own[active] === undefined ? 'current' : `color of ${active}, from its name`,
      message: { type: 'setColor', folder: active, value: null } });
  }

  items.push({ separator: 'PROD' });
  items.push({ label: '$(warning) PROD for the current file\'s place…', description: 'the folder, the file\'s directory, or an entry', run: 'toggleDanger' });
  for (const name of folders) {
    const prod = danger.includes(name);
    items.push({ label: `$(warning) ${prod ? 'Unmark' : 'Mark'} ${name} as PROD`, description: prod ? 'all of it is PROD now' : undefined,
      message: { type: 'setProd', folder: name, on: !prod } });
  }

  items.push({ separator: 'Settings' });
  const enabled = settings.enabled !== false;
  const auto = settings.auto !== false;
  const readonly = settings.dangerReadonly === true;
  const strength = STRENGTH[settings.strength] ? settings.strength : 'normal';
  items.push({ label: `$(eye${enabled ? '-closed' : ''}) Turn Hue ${on(!enabled)} in this workspace`, description: `now ${on(enabled)}`,
    message: { type: 'setSetting', key: 'enabled', value: !enabled } });
  items.push({ label: `$(lock) Turn PROD read-only ${on(!readonly)}`, description: `now ${on(readonly)}`,
    message: { type: 'setSetting', key: 'dangerReadonly', value: !readonly } });
  items.push({ label: `$(sync) Turn automatic colors ${on(!auto)}`, description: `now ${on(auto)}`,
    message: { type: 'setSetting', key: 'auto', value: !auto } });
  for (const value of Object.keys(STRENGTH)) {
    items.push({ label: `$(color-mode) Strength: ${value}`, description: value === strength ? 'current' : undefined,
      message: { type: 'setSetting', key: 'strength', value } });
  }
  return items;
}

module.exports = { searchItems };
