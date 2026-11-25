// Checks that every language has exactly the keys of en.json, that
// placeholders match, and that every static t('key') in the code exists.
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', 'src');
const langs = ['en', 'es', 'pt', 'fr', 'de', 'it', 'zh', 'ja'];
const dicts = Object.fromEntries(langs.map((l) => [l, JSON.parse(readFileSync(join(root, 'i18n', l + '.json'), 'utf8'))]));
const ref = dicts.en;
const placeholders = (s) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',');
let errors = 0;
const fail = (msg) => {
  errors++;
  console.error('✗ ' + msg);
};

for (const l of langs.slice(1)) {
  const d = dicts[l];
  for (const k of Object.keys(ref)) {
    if (!(k in d)) fail(`${l}: missing "${k}"`);
    else if (placeholders(d[k]) !== placeholders(ref[k])) fail(`${l}: placeholders differ in "${k}"`);
  }
  for (const k of Object.keys(d)) if (!(k in ref)) fail(`${l}: extra key "${k}"`);
}

function walk(dir) {
  return readdirSync(dir).flatMap((f) => {
    const p = join(dir, f);
    return statSync(p).isDirectory() ? walk(p) : /\.(svelte|js)$/.test(f) ? [p] : [];
  });
}
for (const file of walk(root)) {
  const src = readFileSync(file, 'utf8');
  // only complete literal keys: t('a.b') or t('a.b', …); t('prefix.' + x) is dynamic
  for (const m of src.matchAll(/\bt\('([\w.-]+)'\s*[,)]/g)) {
    if (!(m[1] in ref)) fail(`${file}: unknown key "${m[1]}"`);
  }
}

// dynamic keys built in code: prefix + every value they can take
const dynamic = {
  'nav.': ['dashboard', 'storage', 'shares', 'ai', 'security', 'users', 'system', 'account'],
  'mood.state.': ['calm', 'focused', 'sleepy', 'worried', 'alarmed'],
  'role.': ['admin', 'user'],
  'array.': ['clean', 'degraded', 'resyncing', 'recovering', 'inactive'],
  'risk.': ['low', 'medium', 'high'],
  'risk.advice.': ['medium', 'high'],
  'sim.': ['normal', 'busy', 'hot', 'disk-failing', 'degraded', 'ransomware'],
  'feature.': ['diskrisk', 'sentinel', 'fan', 'forecast', 'assistant', 'explain'],
  'ai.provider.': ['none', 'anthropic', 'ollama'],
  'ai.action.': ['confirmed', 'discarded', 'error'],
  'factor.': ['reallocated', 'reported_uncorrectable', 'command_timeout', 'pending', 'offline_uncorrectable', 'age_years', 'temp_over_45', 'smart_failed'],
};
for (const [prefix, values] of Object.entries(dynamic)) {
  for (const v of values) if (!(prefix + v in ref)) fail(`dynamic key missing: ${prefix + v}`);
}
for (const f of dynamic['feature.']) if (!(`feature.${f}.help` in ref)) fail(`missing feature.${f}.help`);

if (errors) {
  console.error(`\n${errors} i18n problem(s)`);
  process.exit(1);
}
console.log(`i18n OK: ${Object.keys(ref).length} keys × ${langs.length} languages`);
