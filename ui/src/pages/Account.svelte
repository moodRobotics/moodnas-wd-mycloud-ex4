<script>
  import { t } from '../lib/i18n.svelte.js';
  import { post } from '../lib/api.js';
  import { app, signedIn } from '../lib/session.svelte.js';

  let pw = $state({ current: '', new: '' });
  let pwMsg = $state('');
  let pwErr = $state('');

  let enrol = $state(null); // {secret, url}
  let code = $state('');
  let totpErr = $state('');
  let disablePw = $state('');

  async function changePw(e) {
    e.preventDefault();
    pwMsg = pwErr = '';
    try {
      signedIn(await post('/me/password', pw));
      pw = { current: '', new: '' };
      pwMsg = t('account.pwChanged');
    } catch (err) {
      pwErr = t(err.key, err.params);
    }
  }
  async function begin() {
    totpErr = '';
    enrol = await post('/me/totp/begin');
  }
  async function confirmTotp(e) {
    e.preventDefault();
    try {
      await post('/me/totp/confirm', { secret: enrol.secret, code });
      enrol = null;
      code = '';
      app.me = { ...app.me, totp: true };
    } catch (err) {
      totpErr = t(err.key, err.params);
    }
  }
  async function disable(e) {
    e.preventDefault();
    try {
      await post('/me/totp/disable', { password: disablePw });
      disablePw = '';
      app.me = { ...app.me, totp: false };
    } catch (err) {
      totpErr = t(err.key, err.params);
    }
  }
</script>

<h1>{t('nav.account')}</h1>
<section class="grid cols-2">
  <form class="card stack" onsubmit={changePw}>
    <h2>{t('account.password')}</h2>
    <div class="field"><label for="cp">{t('account.current')}</label><input id="cp" type="password" bind:value={pw.current} autocomplete="current-password" required /></div>
    <div class="field"><label for="np">{t('account.new')}</label><input id="np" type="password" bind:value={pw.new} minlength="10" autocomplete="new-password" required /><div class="muted small">{t('user.passwordRules')}</div></div>
    {#if pwErr}<p class="error">{pwErr}</p>{/if}
    {#if pwMsg}<p class="badge ok">✓ {pwMsg}</p>{/if}
    <button class="primary">{t('common.save')}</button>
  </form>

  <div class="card stack">
    <h2>{t('account.totp')}</h2>
    <p class="muted small">{t('account.totpHelp')}</p>
    {#if app.me.totp}
      <p class="badge ok">✓ {t('account.totpOn')}</p>
      <form class="row" onsubmit={disable}>
        <input type="password" bind:value={disablePw} placeholder={t('account.current')} aria-label={t('account.current')} required style="flex:1;width:auto" />
        <button class="danger">{t('account.totpDisable')}</button>
      </form>
    {:else if !enrol}
      <button class="primary" onclick={begin}>{t('account.totpEnable')}</button>
    {:else}
      <ol class="small">
        <li>{t('account.totpStep1')}</li>
        <li>{t('account.totpStep2')}</li>
      </ol>
      <div class="secret mono">{enrol.secret.match(/.{1,4}/g).join(' ')}</div>
      <a class="small" href={enrol.url}>{t('account.totpLink')}</a>
      <form class="row" onsubmit={confirmTotp}>
        <input class="mono" bind:value={code} inputmode="numeric" maxlength="6" placeholder="123456" aria-label={t('login.totp')} required style="flex:1;width:auto" />
        <button class="primary">{t('account.totpConfirm')}</button>
      </form>
    {/if}
    {#if totpErr}<p class="error">{totpErr}</p>{/if}
  </div>
</section>

<style>
  .secret {
    font-size: 1.1rem;
    background: var(--surface-2);
    border-radius: var(--radius-s);
    padding: 0.6rem 0.8rem;
    word-break: break-all;
  }
</style>
