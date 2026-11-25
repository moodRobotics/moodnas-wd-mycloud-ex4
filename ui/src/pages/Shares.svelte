<script>
  import { t } from '../lib/i18n.svelte.js';
  import { get, post, put, del } from '../lib/api.js';
  import { isAdmin } from '../lib/session.svelte.js';

  let shares = $state([]);
  let users = $state([]);
  let error = $state('');
  let form = $state({ name: '', description: '', users: ['*'], readOnly: false, sentinel: true });

  async function load() {
    shares = await get('/shares');
    if (isAdmin()) users = await get('/users');
  }
  load();

  const toggle = (list, v) => (list.includes(v) ? list.filter((x) => x !== v) : [...list, v]);

  async function create(e) {
    e.preventDefault();
    error = '';
    try {
      await post('/shares', form);
      form = { name: '', description: '', users: ['*'], readOnly: false, sentinel: true };
      load();
    } catch (err) {
      error = t(err.key, err.params);
    }
  }
  async function save(sh) {
    try {
      await put('/shares/' + encodeURIComponent(sh.name), { ...sh, name: undefined });
      load();
    } catch (err) {
      error = t(err.key, err.params);
    }
  }
  async function remove(sh) {
    if (!confirm(t('shares.confirmDelete', { name: sh.name }))) return;
    await del('/shares/' + encodeURIComponent(sh.name));
    load();
  }
</script>

<h1>{t('nav.shares')}</h1>
<p class="muted">{t('shares.help')}</p>
{#if error}<p class="error" role="alert">{error}</p>{/if}

<section class="grid cols-2">
  {#each shares as sh (sh.name)}
    <article class="card stack">
      <div class="row">
        <h2 style="margin:0">⌂ {sh.name}</h2>
        <span class="spacer"></span>
        {#if sh.readOnly}<span class="badge warn">{t('shares.readOnly')}</span>{/if}
        {#if sh.sentinel}<span class="badge accent">✦ {t('shares.sentinel')}</span>{/if}
      </div>
      {#if sh.description}<p class="muted">{sh.description}</p>{/if}
      <div class="small">
        <span class="muted">{t('shares.access')}:</span>
        {sh.users.includes('*') ? t('shares.everyone') : sh.users.length ? sh.users.join(', ') : t('shares.adminsOnly')}
      </div>
      <div class="small muted mono">\\{location.hostname}\{sh.name}</div>
      {#if isAdmin()}
        <details>
          <summary class="small">{t('common.edit')}</summary>
          <div class="stack" style="margin-top:.6rem">
            <input bind:value={sh.description} placeholder={t('shares.description')} aria-label={t('shares.description')} />
            <fieldset class="users">
              <legend class="small">{t('shares.access')}</legend>
              <label class="check"><input type="checkbox" checked={sh.users.includes('*')} onchange={() => (sh.users = toggle(sh.users, '*'))} />{t('shares.everyone')}</label>
              {#each users as u (u.name)}
                <label class="check"><input type="checkbox" checked={sh.users.includes(u.name)} onchange={() => (sh.users = toggle(sh.users, u.name))} />{u.name}</label>
              {/each}
            </fieldset>
            <label class="check"><input type="checkbox" bind:checked={sh.readOnly} />{t('shares.readOnly')}</label>
            <label class="check"><input type="checkbox" bind:checked={sh.sentinel} />{t('shares.sentinelHelp')}</label>
            <div class="row"><button class="primary" onclick={() => save(sh)}>{t('common.save')}</button><span class="spacer"></span><button class="danger" onclick={() => remove(sh)}>{t('common.delete')}</button></div>
          </div>
        </details>
      {/if}
    </article>
  {/each}
</section>

{#if isAdmin()}
  <form class="card stack" style="margin-top:1rem;max-width:560px" onsubmit={create}>
    <h2>{t('shares.new')}</h2>
    <div class="field"><label for="sn">{t('shares.name')}</label><input id="sn" bind:value={form.name} maxlength="32" required /></div>
    <div class="field"><label for="sd">{t('shares.description')}</label><input id="sd" bind:value={form.description} /></div>
    <fieldset class="users">
      <legend class="small">{t('shares.access')}</legend>
      <label class="check"><input type="checkbox" checked={form.users.includes('*')} onchange={() => (form.users = toggle(form.users, '*'))} />{t('shares.everyone')}</label>
      {#each users as u (u.name)}
        <label class="check"><input type="checkbox" checked={form.users.includes(u.name)} onchange={() => (form.users = toggle(form.users, u.name))} />{u.name}</label>
      {/each}
    </fieldset>
    <label class="check"><input type="checkbox" bind:checked={form.sentinel} />{t('shares.sentinelHelp')}</label>
    <button class="primary">{t('shares.create')}</button>
  </form>
{/if}

<style>
  .users {
    border: 1px solid var(--line);
    border-radius: var(--radius-s);
    padding: 0.5rem 0.8rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem 1.2rem;
  }
  summary {
    cursor: pointer;
    font-weight: 700;
  }
</style>
