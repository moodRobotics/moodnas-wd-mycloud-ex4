<script>
  import { t, fmtBytes, fmtNum } from '../lib/i18n.svelte.js';
  import { app } from '../lib/session.svelte.js';

  let s = $derived(app.status);
  // a number, so t() formats it for the current language
  const pct = (p) => Math.round(p * 1000) / 10;
</script>

<h1>{t('nav.storage')}</h1>
{#if s}
  <section class="grid cols-2">
    {#each s.disks as d (d.device)}
      {@const risk = s.risks?.find((r) => r?.device === d.device)}
      <article class="card stack">
        <div class="row">
          <h2 style="margin:0">{t('storage.bay', { n: d.bay })}</h2>
          <span class="spacer"></span>
          {#if d.sleeping}<span class="badge">{t('storage.sleeping')}</span>{/if}
          <span class="badge {d.smart?.healthy ? 'ok' : 'bad'}">SMART {d.smart?.healthy ? t('storage.passed') : t('storage.failed')}</span>
        </div>
        <div class="muted small">{d.model} · {fmtBytes(d.sizeBytes)} · <span class="mono">{d.serial}</span> · /dev/{d.device}</div>
        {#if d.smart}
          <table class="small">
            <tbody>
              <tr><th>{t('smart.temp')}</th><td class="nw">{d.smart.tempC} °C</td><th>{t('smart.hours')}</th><td>{fmtNum(d.smart.powerOnHours / 8766)} {t('smart.years')}</td></tr>
              <tr><th>{t('smart.realloc')}</th><td>{d.smart.reallocated}</td><th>{t('smart.pending')}</th><td>{d.smart.pending}</td></tr>
              <tr><th>{t('smart.uncorr')}</th><td>{d.smart.offlineUncorr + d.smart.reportedUncorr}</td><th>{t('smart.crc')}</th><td>{d.smart.crcErrors}</td></tr>
            </tbody>
          </table>
        {/if}
        {#if risk}
          <div class="notice {risk.level === 'high' ? 'bad' : risk.level === 'medium' ? 'warn' : ''}">
            <strong>✦ {t('risk.title')}: {t('risk.' + risk.level)}</strong>
            <span class="muted"> · {t('risk.prob', { p: pct(risk.probability) })}</span>
            {#if risk.factors?.length}
              <div class="small">{t('risk.because')} {risk.factors.slice(0, 3).map((f) => t('factor.' + f.name)).join(', ')}</div>
            {/if}
            {#if risk.level !== 'low'}<div class="small">{t('risk.advice.' + risk.level)}</div>{/if}
          </div>
        {/if}
      </article>
    {/each}
  </section>

  <section class="card" style="margin-top:1rem">
    <h2>{t('storage.arrays')}</h2>
    <table>
      <thead><tr><th>{t('storage.name')}</th><th>{t('storage.level')}</th><th>{t('storage.state')}</th><th>{t('storage.members')}</th><th>{t('storage.capacity')}</th></tr></thead>
      <tbody>
        {#each s.arrays as a (a.name)}
          <tr>
            <td class="mono">{a.name}</td>
            <td>{a.level}</td>
            <td><span class="badge {a.state === 'clean' ? 'ok' : a.state === 'degraded' ? 'bad' : 'warn'}">{t('array.' + a.state)}</span>{#if a.progress} {a.progress}%{/if}</td>
            <td class="mono small">{a.members.join(' ')}{#if a.failed?.length} <span class="error">✗ {a.failed.join(' ')}</span>{/if}</td>
            <td>{fmtBytes(a.usedBytes)} / {fmtBytes(a.sizeBytes)}</td>
          </tr>
        {/each}
      </tbody>
    </table>
    <p class="muted small">{t('storage.compat')}</p>
  </section>
{/if}

<style>
  .nw {
    white-space: nowrap;
  }
</style>
