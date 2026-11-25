<script>
  import MoodEyes from '../lib/MoodEyes.svelte';
  import { t, i18n, fmtBytes, fmtDate } from '../lib/i18n.svelte.js';
  import { get, post, put } from '../lib/api.js';
  import { app, isAdmin } from '../lib/session.svelte.js';

  let settings = $state(null);
  let apiKey = $state('');
  let saved = $state(false);
  let ledger = $state([]);

  let history = $state([]); // {role, text, pending?}
  let message = $state('');
  let thinking = $state(false);
  let chatError = $state('');
  let chatEl = $state();

  const localFeatures = ['diskrisk', 'sentinel', 'fan', 'forecast'];
  const remoteFeatures = ['assistant', 'explain'];
  const suggestions = ['ai.suggest.health', 'ai.suggest.disks', 'ai.suggest.share', 'ai.suggest.fan'];

  async function load() {
    if (isAdmin()) settings = await get('/ai/settings');
    ledger = (await get('/ai/ledger')) ?? [];
  }
  load();

  async function saveSettings(e) {
    e?.preventDefault();
    const body = { ...settings, apiKey: apiKey ? apiKey : undefined };
    delete body.hasKey;
    settings = await put('/ai/settings', body);
    apiKey = '';
    saved = true;
    setTimeout(() => (saved = false), 2000);
  }
  async function removeKey() {
    const body = { ...settings, apiKey: '' };
    delete body.hasKey;
    settings = await put('/ai/settings', body);
  }

  async function send(text) {
    const msg = (text ?? message).trim();
    if (!msg || thinking) return;
    chatError = '';
    message = '';
    const prior = history.map(({ role, text }) => ({ role, text }));
    history.push({ role: 'user', text: msg });
    thinking = true;
    try {
      const r = await post('/ai/chat', { history: prior, message: msg, lang: i18n.lang });
      history.push({ role: 'assistant', text: r.reply, pending: r.pending ?? [], tools: r.tools ?? [] });
      ledger = (await get('/ai/ledger')) ?? [];
    } catch (err) {
      chatError = t(err.key, err.params);
    } finally {
      thinking = false;
      queueMicrotask(() => chatEl?.scrollTo({ top: chatEl.scrollHeight, behavior: 'smooth' }));
    }
  }

  async function decide(pa, confirm) {
    try {
      await post('/ai/actions/' + pa.id, { confirm });
      pa.done = confirm ? 'confirmed' : 'discarded';
    } catch (err) {
      pa.done = 'error';
      pa.error = t(err.key, err.params);
    }
  }

  let s = $derived(app.status);
  let assistantOn = $derived(settings ? settings.features.assistant && settings.provider !== 'none' : true);
</script>

<h1>{t('nav.ai')}</h1>
<p class="muted">{t('ai.intro')}</p>

<section class="grid cols-2">
  <!-- Assistant -->
  <div class="card chat">
    <div class="row">
      <MoodEyes mood={thinking ? 'focused' : chatError ? 'worried' : 'calm'} size={64} follow={false} />
      <div>
        <h2 style="margin:0">{t('ai.assistant')}</h2>
        <div class="muted small">{t('ai.assistantSub')}</div>
      </div>
    </div>
    <div class="log" bind:this={chatEl} aria-live="polite">
      {#if history.length === 0}
        <p class="muted small">{t('ai.empty')}</p>
        <div class="row">
          {#each suggestions as k (k)}<button class="small" onclick={() => send(t(k))}>{t(k)}</button>{/each}
        </div>
      {/if}
      {#each history as m, i (i)}
        <div class="msg {m.role}">
          <div class="bubble">{m.text}</div>
          {#if m.tools?.length}<div class="tools small muted">✦ {t('ai.usedTools')}: {m.tools.join(', ')}</div>{/if}
          {#each m.pending ?? [] as pa (pa.id)}
            <div class="pending notice">
              <div class="small muted">{t('ai.pendingTitle')}</div>
              <strong>{pa.summary}</strong>
              {#if pa.done}
                <div class="small">{t('ai.action.' + pa.done)} {pa.error ?? ''}</div>
              {:else}
                <div class="row" style="margin-top:.5rem">
                  <button class="primary" onclick={() => decide(pa, true)}>{t('ai.confirm')}</button>
                  <button class="ghost" onclick={() => decide(pa, false)}>{t('ai.discard')}</button>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/each}
      {#if thinking}<div class="msg assistant"><div class="bubble muted">{t('ai.thinking')}</div></div>{/if}
    </div>
    {#if chatError}<p class="error" role="alert">{chatError}</p>{/if}
    {#if !assistantOn}
      <p class="notice small">{t('ai.assistantOff')}</p>
    {/if}
    <form
      class="row composer"
      onsubmit={(e) => {
        e.preventDefault();
        send();
      }}
    >
      <label class="sr-only" for="msg">{t('ai.placeholder')}</label>
      <input id="msg" bind:value={message} placeholder={t('ai.placeholder')} maxlength="4000" disabled={!assistantOn} />
      <button class="primary" disabled={thinking || !assistantOn}>{t('ai.send')}</button>
    </form>
  </div>

  <!-- Local insights -->
  <div class="stack">
    <div class="card stack">
      <h2>{t('ai.local')}</h2>
      <p class="muted small">{t('ai.localSub')}</p>
      {#if s}
        <div class="insight">
          <strong>⛁ {t('feature.diskrisk')}</strong>
          <div class="small">
            {#each s.risks ?? [] as r (r.device)}
              <span class="badge {r.level === 'high' ? 'bad' : r.level === 'medium' ? 'warn' : 'ok'}">{t('storage.bay', { n: r.bay })}: {t('risk.' + r.level)}</span>
            {/each}
          </div>
        </div>
        <div class="insight">
          <strong>⛨ {t('feature.sentinel')}</strong>
          <div class="small">{s.incidents?.length ? t('ai.sentinelAlert', { n: s.incidents.length }) : t('ai.sentinelOk')}</div>
        </div>
        <div class="insight">
          <strong>❄ {t('feature.fan')}</strong>
          <div class="small">{t('ai.fanInfo', { pct: s.fan.pct, c: s.fan.predictedCpuC, trend: s.fan.trendPerMin })}</div>
        </div>
        <div class="insight">
          <strong>◔ {t('feature.forecast')}</strong>
          <div class="small">
            {#if s.fill.learning}{t('fill.learning')}{:else if s.fill.daysToFull < 0}{t('fill.stable')}{:else}{t('fill.days', { days: s.fill.daysToFull, rate: fmtBytes(s.fill.bytesPerDay) })}{/if}
          </div>
        </div>
      {/if}
    </div>

    <div class="card stack">
      <h2>{t('ai.privacy')}</h2>
      <p class="muted small">{t('ai.privacySub')}</p>
      {#if ledger.length === 0}
        <p class="small">✓ {t('ai.privacyNone')}</p>
      {:else}
        <table class="small">
          <thead><tr><th>{t('ai.when')}</th><th>{t('ai.provider')}</th><th>{t('ai.sent')}</th></tr></thead>
          <tbody>
            {#each ledger.slice(0, 8) as e, i (i)}
              <tr>
                <td>{fmtDate(e.at)}</td>
                <td>{e.provider} · <span class="mono">{e.model}</span></td>
                <td>{t('ai.sentDetail', { bytes: fmtBytes(e.bytes), tools: e.toolsSent?.join(', ') || '–' })}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  </div>
</section>

{#if isAdmin() && settings}
  <form class="card stack settings" onsubmit={saveSettings}>
    <h2>{t('ai.settings')}</h2>
    <div class="grid cols-2">
      <div class="stack">
        <div class="field">
          <label for="prov">{t('ai.provider')}</label>
          <select id="prov" bind:value={settings.provider}>
            <option value="none">{t('ai.provider.none')}</option>
            <option value="anthropic">{t('ai.provider.anthropic')}</option>
            <option value="ollama">{t('ai.provider.ollama')}</option>
          </select>
        </div>
        {#if settings.provider !== 'none'}
          <div class="field">
            <label for="model">{t('ai.model')}</label>
            <input id="model" bind:value={settings.model} placeholder={settings.provider === 'anthropic' ? 'claude-opus-5-5' : 'qwen3:8b'} />
          </div>
        {/if}
        {#if settings.provider === 'anthropic'}
          <div class="field">
            <label for="key">{t('ai.apiKey')}</label>
            <input id="key" type="password" bind:value={apiKey} autocomplete="off" placeholder={settings.hasKey ? '••••••••  ' + t('ai.keySaved') : 'sk-ant-…'} />
            <div class="muted small">{t('ai.keyHelp')}</div>
            {#if settings.hasKey}<button type="button" class="ghost small" onclick={removeKey}>{t('ai.removeKey')}</button>{/if}
          </div>
        {/if}
        {#if settings.provider === 'ollama'}
          <div class="field">
            <label for="ep">{t('ai.endpoint')}</label>
            <input id="ep" bind:value={settings.endpoint} placeholder="http://192.168.1.20:11434" />
            <div class="muted small">{t('ai.ollamaHelp')}</div>
          </div>
        {/if}
      </div>
      <div class="stack">
        <fieldset>
          <legend class="small">{t('ai.localFeatures')}</legend>
          {#each localFeatures as f (f)}
            <label class="check"><input type="checkbox" bind:checked={settings.features[f]} /><span><strong>{t('feature.' + f)}</strong><br /><span class="muted small">{t('feature.' + f + '.help')}</span></span></label>
          {/each}
        </fieldset>
        <fieldset>
          <legend class="small">{t('ai.remoteFeatures')}</legend>
          {#each remoteFeatures as f (f)}
            <label class="check"><input type="checkbox" bind:checked={settings.features[f]} /><span><strong>{t('feature.' + f)}</strong><br /><span class="muted small">{t('feature.' + f + '.help')}</span></span></label>
          {/each}
        </fieldset>
      </div>
    </div>
    <div class="row"><button class="primary">{t('common.save')}</button>{#if saved}<span class="badge ok">✓ {t('common.saved')}</span>{/if}</div>
  </form>
{/if}

<style>
  .chat {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
    min-height: 520px;
  }
  .chat :global(svg) {
    border-radius: 14px;
  }
  .log {
    flex: 1;
    overflow-y: auto;
    max-height: 480px;
    display: flex;
    flex-direction: column;
    gap: 0.7rem;
    padding-right: 0.3rem;
  }
  .msg {
    display: flex;
    flex-direction: column;
    max-width: 88%;
  }
  .msg.user {
    align-self: flex-end;
    align-items: flex-end;
  }
  .bubble {
    padding: 0.6rem 0.9rem;
    border-radius: 16px;
    background: var(--surface-2);
    white-space: pre-wrap;
  }
  .msg.user .bubble {
    background: var(--accent);
    color: #fff;
  }
  .tools {
    margin-top: 0.2rem;
  }
  .pending {
    margin-top: 0.5rem;
  }
  .composer input {
    flex: 1;
    width: auto;
  }
  .insight {
    border-top: 1px solid var(--line);
    padding-top: 0.6rem;
  }
  .insight .badge {
    margin: 0.2rem 0.2rem 0 0;
  }
  .settings {
    margin-top: 1rem;
  }
  fieldset {
    border: 1px solid var(--line);
    border-radius: var(--radius-s);
    padding: 0.6rem 0.9rem;
    display: grid;
    gap: 0.6rem;
  }
</style>
