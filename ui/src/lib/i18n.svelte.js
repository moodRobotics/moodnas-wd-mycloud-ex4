import en from '../i18n/en.json';
import es from '../i18n/es.json';
import pt from '../i18n/pt.json';
import fr from '../i18n/fr.json';
import de from '../i18n/de.json';
import it from '../i18n/it.json';
import zh from '../i18n/zh.json';
import ja from '../i18n/ja.json';

const dicts = { en, es, pt, fr, de, it, zh, ja };

export const languages = [
  { code: 'es', name: 'Español' },
  { code: 'en', name: 'English' },
  { code: 'pt', name: 'Português' },
  { code: 'fr', name: 'Français' },
  { code: 'de', name: 'Deutsch' },
  { code: 'it', name: 'Italiano' },
  { code: 'zh', name: '简体中文' },
  { code: 'ja', name: '日本語' },
];

function detect() {
  const saved = localStorage.getItem('moodnas.lang');
  if (saved && dicts[saved]) return saved;
  for (const l of navigator.languages || [navigator.language]) {
    const c = (l || '').slice(0, 2).toLowerCase();
    if (dicts[c]) return c;
  }
  return 'en';
}

export const i18n = $state({ lang: detect() });

export function setLang(code) {
  if (!dicts[code]) return;
  i18n.lang = code;
  localStorage.setItem('moodnas.lang', code);
  document.documentElement.lang = code;
}

document.documentElement.lang = i18n.lang;

// t(key, {name: 'x'}) -> translated string with {name} placeholders filled.
// Falls back to English, then to the key itself.
export function t(key, params) {
  const s = dicts[i18n.lang]?.[key] ?? en[key] ?? key;
  if (!params) return s;
  return s.replace(/\{(\w+)\}/g, (m, k) => (params[k] !== undefined ? fmtParam(params[k]) : m));
}

function fmtParam(v) {
  if (typeof v === 'number') return new Intl.NumberFormat(i18n.lang).format(Math.round(v * 10) / 10);
  return String(v);
}

export function fmtNum(n, digits = 1) {
  return new Intl.NumberFormat(i18n.lang, { minimumFractionDigits: digits, maximumFractionDigits: digits }).format(n);
}

// Disk sizes use decimal units (as printed on drives); RAM uses binary units.
export function fmtBytes(n, binary = false) {
  if (n == null) return '–';
  const units = binary ? ['B', 'KiB', 'MiB', 'GiB', 'TiB'] : ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const step = binary ? 1024 : 1000;
  let i = 0;
  let v = n;
  while (v >= step && i < units.length - 1) {
    v /= step;
    i++;
  }
  return `${new Intl.NumberFormat(i18n.lang, { maximumFractionDigits: v < 10 ? 1 : 0 }).format(v)} ${units[i]}`;
}

export function fmtDuration(ns) {
  const s = Math.floor(ns / 1e9);
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return t('time.dh', { d, h });
  if (h > 0) return t('time.hm', { h, m });
  return t('time.m', { m });
}

export function fmtDate(iso) {
  return new Intl.DateTimeFormat(i18n.lang, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso));
}
