<script>
  // Performance history: CPU, load, RAM and temperatures, one chart each
  // (never two scales on one axis), with one shared time-range control.
  import LineChart from './LineChart.svelte';
  import { t } from './i18n.svelte.js';
  import { get } from './api.js';

  const ranges = ['1h', '6h', '24h'];
  let range = $state(localStorage.getItem('moodnas.perfRange') || '1h');
  let data = $state(null);
  let loading = $state(false);

  async function load() {
    loading = true;
    try {
      data = await get('/history?range=' + range);
    } catch {
      /* keep the previous render */
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    localStorage.setItem('moodnas.perfRange', range);
    load();
    const id = setInterval(load, 30000);
    return () => clearInterval(id);
  });

  let pts = $derived(data?.points ?? []);
  let hasCpuTemp = $derived(pts.some((p) => p.cpuTemp != null));
</script>

<section class="perf stack">
  <div class="row">
    <h2 style="margin:0">{t('perf.title')}</h2>
    <span class="spacer"></span>
    <div class="seg" role="group" aria-label={t('perf.range')}>
      {#each ranges as r (r)}
        <button class:active={range === r} aria-pressed={range === r} onclick={() => (range = r)}>{t('perf.r' + r)}</button>
      {/each}
    </div>
  </div>

  {#if data}
    <div class="charts" class:loading>
      <LineChart title={t('perf.cpu')} points={pts} from={data.from} to={data.to} unit=" %" yMax={100}
        series={[{ key: 'cpu', label: t('perf.cpu'), color: 'var(--series-1)' }]} />
      <LineChart title={t('perf.load')} points={pts} from={data.from} to={data.to} decimals={2}
        series={[{ key: 'load', label: t('perf.load'), color: 'var(--series-1)' }]} />
      <LineChart title={t('perf.mem')} points={pts} from={data.from} to={data.to} unit=" %" yMax={100}
        series={[{ key: 'mem', label: t('perf.mem'), color: 'var(--series-1)' }]} />
      <LineChart title={t('perf.temp')} points={pts} from={data.from} to={data.to} unit=" °C" decimals={1} zeroBased={false}
        note={t('perf.tempMissing')}
        series={hasCpuTemp
          ? [
              { key: 'cpuTemp', label: t('perf.cpuTemp'), color: 'var(--series-1)' },
              { key: 'diskTemp', label: t('perf.diskTemp'), color: 'var(--series-2)' },
            ]
          : [{ key: 'diskTemp', label: t('perf.diskTemp'), color: 'var(--series-2)' }]} />
    </div>
    <p class="muted small">{t('perf.loadHelp')}</p>
  {/if}
</section>

<style>
  .perf {
    margin-top: 1rem;
  }
  .seg {
    display: inline-flex;
    border: 1px solid var(--line);
    border-radius: 999px;
    padding: 2px;
    background: var(--surface);
  }
  .seg button {
    border: 0;
    background: transparent;
    padding: 0.3rem 0.85rem;
  }
  .seg button.active {
    background: var(--accent);
    color: #fff;
  }
  .charts {
    display: grid;
    gap: 1rem;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  @media (max-width: 900px) {
    .charts {
      grid-template-columns: 1fr;
    }
  }
  .loading {
    opacity: 0.6;
    transition: opacity 0.2s;
  }
</style>
