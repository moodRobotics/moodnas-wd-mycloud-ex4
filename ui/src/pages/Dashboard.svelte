<script>
  import MoodEyes from '../lib/MoodEyes.svelte';
  import FrontPanel from '../lib/FrontPanel.svelte';
  import { t, fmtBytes, fmtDuration, fmtNum } from '../lib/i18n.svelte.js';
  import { post } from '../lib/api.js';
  import { app, isAdmin, refreshStatus } from '../lib/session.svelte.js';

  let s = $derived(app.status);
  let mood = $derived(s?.mood?.state ?? 'calm');
  let array = $derived(s?.arrays?.[0]);
  let usedPct = $derived(array?.sizeBytes ? Math.round((array.usedBytes / array.sizeBytes) * 100) : 0);
  let netUp = $derived(s?.system?.net?.find((n) => n.up));

  const scenarios = ['normal', 'busy', 'hot', 'disk-failing', 'degraded', 'ransomware'];
  async function scenario(name) {
    await post('/sim/scenario', { scenario: name });
    refreshStatus();
  }
  const sevClass = (sev) => (sev >= 3 ? 'bad' : sev === 2 ? 'warn' : '');
</script>

{#if !s}
  <p class="muted">{t('common.loading')}</p>
{:else}
  <section class="hero card">
    <MoodEyes {mood} size={300} label={t('mood.state.' + mood)} />
    <div class="stack">
      <div class="muted small upper">{t('dash.moodLabel')}</div>
      <h1 class="moodname">{t('mood.state.' + mood)}</h1>
      <ul class="reasons">
        {#each s.mood.reasons as r, i (i)}
          <li class="notice {sevClass(r.severity)}">{t(r.key, r.params)}</li>
        {/each}
      </ul>
      {#if s.fan?.reason}
        <p class="muted small">✦ {t(s.fan.reason)}</p>
      {/if}
    </div>
  </section>

  <section class="grid cols-4 stats">
    <div class="card">
      <div class="muted small">{t('dash.cpuTemp')}</div>
      <div class="big">{fmtNum(s.sensors.cpuTempC)}<small>°C</small></div>
      {#if s.fan.trendPerMin > 0.2}<div class="small warn-t">↗ {t('dash.predicted', { c: s.fan.predictedCpuC })}</div>{/if}
    </div>
    <div class="card">
      <div class="muted small">{t('dash.fan')}</div>
      <div class="big">{s.sensors.fanPct >= 0 ? s.sensors.fanPct : '–'}<small>%</small></div>
      <div class="small muted">{s.fan.quiet ? '☾ ' + t('dash.quiet') : t('dash.fanAi')}</div>
    </div>
    <div class="card">
      <div class="muted small">{t('dash.cpu')}</div>
      <div class="big">{Math.round(s.system.cpuPct)}<small>%</small></div>
      <div class="small muted">{t('dash.load', { l: s.system.load1 })}</div>
    </div>
    <div class="card">
      <div class="muted small">{t('dash.memory')}</div>
      <div class="big">{fmtBytes(s.system.memUsed, true)}</div>
      <div class="small muted">/ {fmtBytes(s.system.memTotal, true)}</div>
    </div>
  </section>

  <section class="grid cols-2">
    <div class="card stack">
      <h2>{t('dash.storage')}</h2>
      {#if array}
        <div class="row"><strong>{fmtBytes(array.usedBytes)}</strong><span class="muted">/ {fmtBytes(array.sizeBytes)}</span><span class="spacer"></span><span class="badge {array.state === 'clean' ? 'ok' : 'bad'}">{array.level.toUpperCase()} · {t('array.' + array.state)}</span></div>
        <div class="bar {usedPct > 90 ? 'bad' : usedPct > 75 ? 'warn' : ''}"><span style="width:{usedPct}%"></span></div>
        <p class="small">
          ✦
          {#if s.fill.learning}{t('fill.learning')}
          {:else if s.fill.daysToFull < 0}{t('fill.stable')}
          {:else}{t('fill.days', { days: s.fill.daysToFull, rate: fmtBytes(s.fill.bytesPerDay) })}{/if}
        </p>
      {/if}
      <div class="bays">
        {#each s.disks as d (d.device)}
          {@const risk = s.risks?.find((r) => r?.device === d.device)}
          <div class="bay {risk?.level}" title={d.model}>
            <div class="small muted">{t('storage.bay', { n: d.bay })}</div>
            <div class="small">{d.smart?.tempC ?? '–'}°C</div>
            {#if risk}<div class="small risk">{t('risk.' + risk.level)}</div>{/if}
          </div>
        {/each}
      </div>
      <a href="#/storage" class="small">{t('dash.storageMore')} →</a>
    </div>

    <div class="card stack">
      <h2>{t('dash.device')}</h2>
      <FrontPanel panel={s.panel} />
      <table class="small">
        <tbody>
          <tr><th>{t('dash.model')}</th><td>{s.system.model}</td></tr>
          <tr><th>{t('dash.uptime')}</th><td>{fmtDuration(s.system.uptime)}</td></tr>
          <tr><th>{t('dash.network')}</th><td>{netUp ? `${netUp.name} · ${netUp.speedMbps} Mb/s · ${netUp.address || ''}` : '–'}</td></tr>
          <tr><th>{t('dash.kernel')}</th><td class="mono">{s.system.kernel}</td></tr>
        </tbody>
      </table>
    </div>
  </section>

  {#if app.boot?.simulator && isAdmin()}
    <section class="card stack">
      <h2>{t('sim.title')}</h2>
      <p class="muted small">{t('sim.help')}</p>
      <div class="row">
        {#each scenarios as sc (sc)}
          <button class:primary={app.scenario === sc} onclick={() => scenario(sc)}>{t('sim.' + sc)}</button>
        {/each}
      </div>
    </section>
  {/if}
{/if}

<style>
  section + section {
    margin-top: 1rem;
  }
  .hero {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 2rem;
    align-items: center;
  }
  .upper {
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 700;
  }
  .moodname {
    font-size: 2.6rem;
    margin: 0;
  }
  .reasons {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.5rem;
  }
  .big {
    font-size: 2rem;
    font-weight: 900;
  }
  .big small {
    font-size: 1rem;
    color: var(--muted);
    margin-left: 2px;
  }
  .warn-t {
    color: var(--warn);
  }
  .bays {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.5rem;
  }
  .bay {
    border: 1px solid var(--line);
    border-radius: 12px;
    padding: 0.6rem;
    background: var(--surface-2);
    border-top: 4px solid var(--ok);
  }
  .bay.medium {
    border-top-color: var(--warn);
  }
  .bay.high {
    border-top-color: var(--bad);
  }
  .risk {
    font-weight: 700;
  }
  th {
    width: 35%;
  }
  @media (max-width: 760px) {
    .hero {
      grid-template-columns: 1fr;
      justify-items: center;
      text-align: center;
    }
  }
</style>
