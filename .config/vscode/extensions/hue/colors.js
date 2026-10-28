// Hue's colors, without VS Code: the color of a folder, the palette of the window
// parts, the danger places, and the colorCustomizations they make. extension.js
// reads the settings and writes the result; colors.test.js tests this
// (node --test).
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

// A folder's hue: its own color (hue.folders), else one from its name when
// hue.auto is on; none with Hue off (hue.enabled false)
function folderHue(name, { folders = {}, colors, auto = true, enabled = true }) {
  if (!enabled) return undefined;
  if (folders[name] !== undefined) return hueOf(folders[name], colors);
  return auto ? autoHue(name, colors) : undefined;
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
// focus. Light: the same idea, pale. k scales the saturation (hue.strength).
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

const AMBER = { dark: '#8a5a00', light: '#ffd58a' };
const DANGER = { dark: ['#8a1c1c', '#4a1212'], light: ['#f5b5b5', '#f9d5d5'] };

const PARTS = Object.keys(palette(0, false, 1));
const OWNED = Object.values(palette(0, false, 1)).flatMap((p) => Object.keys(p));

// The workspace's colorCustomizations with Hue's keys for this state: every key
// Hue owns comes out first (others are left alone), then the folder's palette
// (the parts chosen, tabs always), then the warnings, which win over it.
// hue undefined: no folder color. Returns a new object.
function nextCustomizations(current, { hue, light, k = 1, parts = PARTS, onProtected = false, danger = false }) {
  const next = { ...current };
  for (const key of OWNED) delete next[key];
  const chosen = new Set(parts);
  if (hue !== undefined) {
    const groups = palette(hue, light, k);
    for (const part of PARTS) if (part === 'tabs' || chosen.has(part)) Object.assign(next, groups[part]);
  }
  const theme = light ? 'light' : 'dark';
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
  return next;
}

module.exports = {
  DEFAULT_COLORS, STRENGTH, PARTS, OWNED,
  hueOfHex, hueOf, autoHue, folderHue, hex, palette, ansiFor, globRegex, isDanger, nextCustomizations,
};
