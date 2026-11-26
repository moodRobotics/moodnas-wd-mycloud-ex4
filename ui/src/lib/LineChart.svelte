<script>
  // Time-series line chart (one y axis, 1-2 series of the same unit).
  // Dataviz rules: 2px lines, 10% area wash for a single series, hairline
  // grid, crosshair + tooltip that snaps to the nearest point, legend and
  // direct labels for 2 series, gaps where data is missing, data table view.
  import { t, i18n } from './i18n.svelte.js';

  let {
    title,
    points = [],
    series = [], // [{ key, label, color: 'var(--series-1)' }]
    unit = '',
    decimals = 0,
    yMax = null, // fixed top (e.g. 100 for %), else auto
    zeroBased = true,
    from,
    to,
    note = '',
  } = $props();

  const H = 150;
  const M = { top: 10, right: 12, bottom: 22, left: 38 };
  let width = $state(400);
  let hover = $state(-1);

  let multi = $derived(series.length > 1);
  // room for direct labels at the line ends (≈6.5px per character at 11px)
  let right = $derived(multi ? Math.min(130, 14 + 6.5 * Math.max(...series.map((s) => s.label.length))) : M.right);
  let plotW = $derived(Math.max(50, width - M.left - right));
  let plotH = H - M.top - M.bottom;

  let values = $derived(series.flatMap((s) => points.map((p) => p[s.key]).filter((v) => v != null)));
  let hasData = $derived(values.length > 0);

  function niceStep(span) {
    const raw = span / 3;
    const mag = 10 ** Math.floor(Math.log10(raw || 1));
    return [1, 2, 2.5, 5, 10].map((m) => m * mag).find((s) => s >= raw) ?? raw;
  }
  let domain = $derived.by(() => {
    if (!hasData) return [0, yMax ?? 1];
    const hi = Math.max(...values);
    const lo = Math.min(...values);
    if (yMax != null) return [0, yMax];
    if (zeroBased) {
      const step = niceStep(Math.max(hi, 0.01) * 1.15);
      return [0, Math.ceil((hi * 1.15) / step) * step || step];
    }
    const step = niceStep(Math.max(hi - lo, 4) * 1.3);
    return [Math.floor((lo - step * 0.3) / step) * step, Math.ceil((hi + step * 0.3) / step) * step];
  });
  let ticks = $derived.by(() => {
    const [a, b] = domain;
    const step = niceStep(b - a);
    const out = [];
    for (let v = a; v <= b + 1e-9; v += step) out.push(Math.round(v * 100) / 100);
    return out;
  });

  let axisDecimals = $derived(ticks.length > 1 && ticks[1] - ticks[0] < 1 ? (ticks[1] - ticks[0] < 0.1 ? 2 : 1) : 0);
  const fmtAxis = (v) => new Intl.NumberFormat(i18n.lang, { maximumFractionDigits: axisDecimals, minimumFractionDigits: axisDecimals }).format(v);

  // direct labels: at each series' last point, pushed apart so they never overlap
  let directLabels = $derived.by(() => {
    if (!multi) return [];
    const labs = series
      .map((s) => ({ s, p: lastOf(s.key) }))
      .filter((l) => l.p)
      .map((l) => ({ label: l.s.label, key: l.s.key, x: x(l.p.t) + 6, y: y(l.p[l.s.key]) + 4 }))
      .sort((a, b) => a.y - b.y);
    for (let i = 1; i < labs.length; i++) if (labs[i].y - labs[i - 1].y < 13) labs[i].y = labs[i - 1].y + 13;
    return labs;
  });

  const x = (ts) => M.left + ((ts - from) / Math.max(1, to - from)) * plotW;
  const y = (v) => M.top + plotH - ((v - domain[0]) / (domain[1] - domain[0] || 1)) * plotH;

  // one path per series, broken at missing values
  function segments(key) {
    const segs = [];
    let cur = [];
    for (const p of points) {
      if (p[key] == null) {
        if (cur.length) segs.push(cur);
        cur = [];
      } else cur.push(p);
    }
    if (cur.length) segs.push(cur);
    return segs;
  }
  const linePath = (seg, key) => seg.map((p, i) => `${i ? 'L' : 'M'}${x(p.t).toFixed(1)},${y(p[key]).toFixed(1)}`).join('');
  const areaPath = (seg, key) =>
    `${linePath(seg, key)}L${x(seg.at(-1).t).toFixed(1)},${y(domain[0])}L${x(seg[0].t).toFixed(1)},${y(domain[0])}Z`;

  const lastOf = (key) => [...points].reverse().find((p) => p[key] != null);

  const fmt = (v) =>
    v == null ? '–' : new Intl.NumberFormat(i18n.lang, { minimumFractionDigits: decimals, maximumFractionDigits: decimals }).format(v);
  const fmtTime = (ts, withDate = false) =>
    new Intl.DateTimeFormat(i18n.lang, withDate ? { dateStyle: 'short', timeStyle: 'short' } : { timeStyle: 'short' }).format(new Date(ts * 1000));

  function nearest(px) {
    let best = -1;
    let bestD = Infinity;
    points.forEach((p, i) => {
      if (!series.some((s) => p[s.key] != null)) return;
      const d = Math.abs(x(p.t) - px);
      if (d < bestD) {
        bestD = d;
        best = i;
      }
    });
    return best;
  }
  function onMove(ev) {
    const r = ev.currentTarget.getBoundingClientRect();
    hover = nearest(((ev.clientX - r.left) / r.width) * width);
  }
  function onKey(ev) {
    const valid = points.map((p, i) => (series.some((s) => p[s.key] != null) ? i : -1)).filter((i) => i >= 0);
    if (!valid.length) return;
    let pos = valid.indexOf(hover);
    if (ev.key === 'ArrowLeft') pos = pos <= 0 ? 0 : pos - 1;
    else if (ev.key === 'ArrowRight') pos = pos < 0 ? valid.length - 1 : Math.min(valid.length - 1, pos + 1);
    else if (ev.key === 'Escape') return (hover = -1);
    else return;
    ev.preventDefault();
    hover = valid[pos];
  }
  let hp = $derived(hover >= 0 ? points[hover] : null);
  let tipLeft = $derived(hp ? x(hp.t) : 0);
</script>

<figure class="chart">
  <figcaption class="head">
    <span class="title">{title}</span>
    {#if !multi && hasData}
      <span class="now">{fmt(lastOf(series[0].key)?.[series[0].key])}<small>{unit}</small></span>
    {/if}
  </figcaption>
  {#if multi}
    <ul class="legend">
      {#each series as s (s.key)}
        <li><span class="key" style="background:{s.color}"></span>{s.label} <strong>{fmt(lastOf(s.key)?.[s.key])}{unit}</strong></li>
      {/each}
    </ul>
  {/if}

  <div class="plot" bind:clientWidth={width}>
    {#if !hasData}
      <p class="empty muted small">{note || t('perf.noData')}</p>
    {:else}
      <svg
        {width}
        height={H}
        role="img"
        aria-label={title}
        tabindex="0"
        onpointermove={onMove}
        onpointerleave={() => (hover = -1)}
        onkeydown={onKey}
        onblur={() => (hover = -1)}
      >
        {#each ticks as v (v)}
          <line class="grid" x1={M.left} x2={M.left + plotW} y1={y(v)} y2={y(v)} />
          <text class="axis" x={M.left - 6} y={y(v) + 4} text-anchor="end">{fmtAxis(v)}</text>
        {/each}
        {#each [from, (from + to) / 2, to] as ts, i (i)}
          <text class="axis" x={x(ts)} y={H - 6} text-anchor={i === 0 ? 'start' : i === 2 ? 'end' : 'middle'}>{fmtTime(ts)}</text>
        {/each}

        {#each series as s (s.key)}
          {#each segments(s.key) as seg, i (i)}
            {#if !multi}<path d={areaPath(seg, s.key)} fill={s.color} opacity="0.1" />{/if}
            {#if seg.length === 1}
              <circle cx={x(seg[0].t)} cy={y(seg[0][s.key])} r="2.5" fill={s.color} />
            {:else}
              <path d={linePath(seg, s.key)} fill="none" stroke={s.color} stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
            {/if}
          {/each}
        {/each}

        {#each directLabels as l (l.key)}
          <text class="direct" x={l.x} y={l.y}>{l.label}</text>
        {/each}

        {#if hp}
          <line class="cross" x1={x(hp.t)} x2={x(hp.t)} y1={M.top} y2={M.top + plotH} />
          {#each series as s (s.key)}
            {#if hp[s.key] != null}
              <circle cx={x(hp.t)} cy={y(hp[s.key])} r="4" fill={s.color} class="dot" />
            {/if}
          {/each}
        {/if}
      </svg>

      {#if hp}
        <div class="tip" style="left:{tipLeft}px" class:flip={tipLeft > width * 0.6} role="status">
          <div class="tip-time">{fmtTime(hp.t, true)}</div>
          {#each series as s (s.key)}
            <div class="tip-row">
              <span class="lkey" style="background:{s.color}"></span>
              <strong>{fmt(hp[s.key])}{unit}</strong>
              {#if multi}<span class="muted">{s.label}</span>{/if}
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  </div>

  {#if hasData}
    <details class="table small">
      <summary>{t('perf.table')}</summary>
      <div class="scroll">
        <table>
          <thead><tr><th>{t('perf.time')}</th>{#each series as s (s.key)}<th>{s.label} ({unit.trim() || '–'})</th>{/each}</tr></thead>
          <tbody>
            {#each [...points].reverse().filter((p) => series.some((s) => p[s.key] != null)) as p (p.t)}
              <tr><td>{fmtTime(p.t, true)}</td>{#each series as s (s.key)}<td>{fmt(p[s.key])}</td>{/each}</tr>
            {/each}
          </tbody>
        </table>
      </div>
    </details>
  {/if}
</figure>

<style>
  .chart {
    margin: 0;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: var(--radius);
    padding: 1rem 1rem 0.6rem;
    min-width: 0;
  }
  .head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.5rem;
  }
  .title {
    font-weight: 800;
  }
  .now {
    font-size: 1.4rem;
    font-weight: 900;
  }
  .now small {
    font-size: 0.85rem;
    color: var(--muted);
    margin-left: 2px;
  }
  .legend {
    list-style: none;
    display: flex;
    gap: 1rem;
    padding: 0;
    margin: 0.3rem 0 0;
    font-size: 0.85rem;
    color: var(--muted);
  }
  .legend strong {
    color: var(--text);
  }
  .key {
    display: inline-block;
    width: 14px;
    height: 2px;
    border-radius: 1px;
    vertical-align: middle;
    margin-right: 6px;
  }
  .plot {
    position: relative;
    margin-top: 0.4rem;
  }
  svg {
    display: block;
    overflow: visible;
    touch-action: pan-y;
  }
  svg:focus-visible {
    outline: 2px solid var(--accent-2);
    outline-offset: 2px;
    border-radius: 6px;
  }
  .grid {
    stroke: var(--line);
    stroke-width: 1;
  }
  .axis {
    fill: var(--muted);
    font-size: 11px;
    font-variant-numeric: tabular-nums;
  }
  .direct {
    fill: var(--muted);
    font-size: 11px;
    font-weight: 700;
  }
  .cross {
    stroke: var(--muted);
    stroke-width: 1;
  }
  .dot {
    stroke: var(--surface);
    stroke-width: 2;
  }
  .empty {
    height: 150px;
    display: grid;
    place-items: center;
    margin: 0;
    text-align: center;
  }
  .tip {
    position: absolute;
    top: 0;
    transform: translateX(12px);
    background: var(--surface-2);
    border: 1px solid var(--line);
    border-radius: 10px;
    padding: 0.45rem 0.65rem;
    font-size: 0.82rem;
    pointer-events: none;
    white-space: nowrap;
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.35);
  }
  .tip.flip {
    transform: translateX(calc(-100% - 12px));
  }
  .tip-time {
    color: var(--muted);
    margin-bottom: 0.15rem;
  }
  .tip-row {
    display: flex;
    align-items: center;
    gap: 0.45rem;
  }
  .lkey {
    width: 12px;
    height: 2px;
    border-radius: 1px;
  }
  .table summary {
    cursor: pointer;
    color: var(--muted);
    margin-top: 0.3rem;
  }
  .scroll {
    max-height: 220px;
    overflow-y: auto;
    margin-top: 0.4rem;
  }
  .table td,
  .table th {
    padding: 0.25rem 0.4rem;
    font-variant-numeric: tabular-nums;
  }
</style>
