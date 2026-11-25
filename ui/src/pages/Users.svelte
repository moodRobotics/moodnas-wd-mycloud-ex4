<script>
  import { t, fmtDate } from '../lib/i18n.svelte.js';
  import { get, post, del } from '../lib/api.js';
  import { app } from '../lib/session.svelte.js';
  import { USER_PATTERN } from '../lib/patterns.js';

  let users = $state([]);
  let form = $state({ name: '', password: '', role: 'user' });
  let error = $state('');
  const load = async () => (users = await get('/users'));
  load();

  async function create(e) {
    e.preventDefault();
    error = '';
    try {
      await post('/users', form);
      form = { name: '', password: '', role: 'user' };
      load();
    } catch (err) {
      error = t(err.key, err.params);
    }
  }
  async function remove(u) {
    if (!confirm(t('users.confirmDelete', { name: u.name }))) return;
    await del('/users/' + encodeURIComponent(u.name));
    load();
  }
</script>

<h1>{t('nav.users')}</h1>
<section class="grid cols-2">
  <div class="card">
    <table>
      <thead><tr><th>{t('users.name')}</th><th>{t('users.role')}</th><th>2FA</th><th>{t('users.created')}</th><th></th></tr></thead>
      <tbody>
        {#each users as u (u.name)}
          <tr>
            <td><strong>{u.name}</strong></td>
            <td>{t('role.' + u.role)}</td>
            <td>{u.totp ? '✓' : '–'}</td>
            <td class="small">{fmtDate(u.created)}</td>
            <td>{#if u.name !== app.me.user}<button class="danger small" onclick={() => remove(u)}>{t('common.delete')}</button>{/if}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
  <form class="card stack" onsubmit={create}>
    <h2>{t('users.new')}</h2>
    <div class="field"><label for="un">{t('users.name')}</label><input id="un" bind:value={form.name} pattern={USER_PATTERN} maxlength="32" required /><div class="muted small">{t('user.nameRules')}</div></div>
    <div class="field"><label for="upw">{t('login.password')}</label><input id="upw" type="password" bind:value={form.password} minlength="10" autocomplete="new-password" required /><div class="muted small">{t('user.passwordRules')}</div></div>
    <div class="field">
      <label for="ur">{t('users.role')}</label>
      <select id="ur" bind:value={form.role}><option value="user">{t('role.user')}</option><option value="admin">{t('role.admin')}</option></select>
    </div>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <button class="primary">{t('users.create')}</button>
  </form>
</section>
