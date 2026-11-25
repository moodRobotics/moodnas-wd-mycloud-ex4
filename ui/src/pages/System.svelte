<script>
  import { t, languages } from '../lib/i18n.svelte.js';
  import { get, put } from '../lib/api.js';
  import { app } from '../lib/session.svelte.js';
  import { HOST_PATTERN } from '../lib/patterns.js';

  let settings = $state(null);
  let error = $state('');
  let saved = $state(false);
  get('/settings').then((s) => (settings = s));

  async function save(e) {
    e.preventDefault();
    error = '';
    try {
      settings = await put('/settings', settings);
      app.boot = { ...app.boot, hostname: settings.hostname };
      saved = true;
      setTimeout(() => (saved = false), 2000);
    } catch (err) {
      error = t(err.key, err.params);
    }
  }
  const hours = Array.from({ length: 24 }, (_, i) => i);
</script>

<h1>{t('nav.system')}</h1>
{#if settings}
  <form class="grid cols-2" onsubmit={save}>
    <div class="card stack">
      <h2>{t('system.general')}</h2>
      <div class="field"><label for="hn">{t('setup.hostname')}</label><input id="hn" bind:value={settings.hostname} pattern={HOST_PATTERN} maxlength="32" required /></div>
      <div class="field">
        <label for="dl">{t('system.defaultLang')}</label>
        <select id="dl" bind:value={settings.language}>{#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}</select>
        <div class="muted small">{t('system.defaultLangHelp')}</div>
      </div>
    </div>
    <div class="card stack">
      <h2>❄ {t('system.fan')}</h2>
      <label class="check"><input type="radio" bind:group={settings.fan.mode} value="ai" /><span><strong>{t('system.fanAi')}</strong><br /><span class="muted small">{t('system.fanAiHelp')}</span></span></label>
      <label class="check"><input type="radio" bind:group={settings.fan.mode} value="manual" /><span><strong>{t('system.fanManual')}</strong></span></label>
      {#if settings.fan.mode === 'manual'}
        <div class="field"><label for="fp">{settings.fan.manualPct}%</label><input id="fp" type="range" min="20" max="100" step="5" bind:value={settings.fan.manualPct} /></div>
      {/if}
      <div class="row">
        <div class="field"><label for="qs">{t('system.quietFrom')}</label><select id="qs" bind:value={settings.fan.quietStart}>{#each hours as h (h)}<option value={h}>{h}:00</option>{/each}</select></div>
        <div class="field"><label for="qe">{t('system.quietTo')}</label><select id="qe" bind:value={settings.fan.quietEnd}>{#each hours as h (h)}<option value={h}>{h}:00</option>{/each}</select></div>
      </div>
      <p class="muted small">{t('system.safety')}</p>
    </div>
    <div class="row" style="grid-column:1/-1">
      <button class="primary">{t('common.save')}</button>
      {#if saved}<span class="badge ok">✓ {t('common.saved')}</span>{/if}
      {#if error}<span class="error">{error}</span>{/if}
    </div>
  </form>
  <section class="card stack" style="margin-top:1rem">
    <h2>{t('system.about')}</h2>
    <p>{t('system.aboutText')}</p>
    <p class="small muted">MoodNAS v{app.boot?.version} · {app.status?.platform} · GPL-3.0 · Mood Robotics</p>
  </section>
{/if}
