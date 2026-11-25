<script>
  import { t, fmtDate } from '../lib/i18n.svelte.js';
  import { get, post } from '../lib/api.js';
  import { app, refreshStatus } from '../lib/session.svelte.js';

  let audit = $state([]);
  const load = async () => (audit = await get('/audit'));
  load();

  async function resolve(share) {
    if (!confirm(t('security.confirmResolve', { share }))) return;
    await post('/incidents/' + encodeURIComponent(share) + '/resolve');
    await refreshStatus();
    load();
  }
</script>

<h1>{t('nav.security')}</h1>

<section class="card stack">
  <h2>⛨ {t('feature.sentinel')}</h2>
  {#if app.status?.incidents?.length}
    {#each app.status.incidents as inc (inc.share)}
      <div class="notice bad stack">
        <strong>{t('security.incident', { share: inc.share })}</strong>
        <div class="small">{t('security.incidentDetail', { n: inc.events, score: inc.score })} · {fmtDate(inc.at)}</div>
        {#if inc.frozen}<div class="small">❄ {t('security.frozen')}</div>{/if}
        <ul class="small mono">{#each inc.sample as p, i (i)}<li>{p}</li>{/each}</ul>
        <div class="small">{t('security.advice')}</div>
        <div><button class="danger" onclick={() => resolve(inc.share)}>{t('security.resolve')}</button></div>
      </div>
    {/each}
  {:else}
    <p>✓ {t('ai.sentinelOk')}</p>
  {/if}
</section>

<section class="card" style="margin-top:1rem">
  <h2>{t('security.audit')}</h2>
  <table class="small">
    <thead><tr><th>{t('ai.when')}</th><th>{t('security.actor')}</th><th>{t('security.action')}</th><th>{t('security.details')}</th><th>IP</th></tr></thead>
    <tbody>
      {#each audit as e, i (i)}
        <tr>
          <td>{fmtDate(e.at)}</td>
          <td>{e.actor}</td>
          <td class="mono">{e.action}</td>
          <td class="mono">{e.detail ? Object.entries(e.detail).map(([k, v]) => `${k}=${v}`).join(' ') : ''}</td>
          <td class="mono">{e.ip ?? ''}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</section>
