<script>
  import MoodEyes from '../lib/MoodEyes.svelte';
  import { t, i18n, setLang, languages } from '../lib/i18n.svelte.js';
  import { post } from '../lib/api.js';
  import { app, signedIn } from '../lib/session.svelte.js';
  import { HOST_PATTERN, USER_PATTERN } from '../lib/patterns.js';

  let step = $state(0);
  let code = $state('');
  let user = $state('');
  let password = $state('');
  let password2 = $state('');
  let hostname = $state(app.boot?.hostname || 'moodnas');
  let error = $state('');
  let busy = $state(false);

  let mismatch = $derived(password2 !== '' && password !== password2);

  async function finish(e) {
    e.preventDefault();
    error = '';
    if (mismatch) return;
    busy = true;
    try {
      const me = await post('/setup', { setupCode: code, user, password, hostname, language: i18n.lang });
      app.boot = { ...app.boot, setupRequired: false, hostname };
      signedIn(me);
      location.hash = '#/dashboard';
    } catch (err) {
      error = t(err.key, err.params);
      if (err.key === 'setup.badCode') step = 1;
    } finally {
      busy = false;
    }
  }
</script>

<div class="wrap">
  <div class="card box stack">
    <MoodEyes mood={step === 0 ? 'calm' : step === 1 ? 'focused' : 'calm'} size={200} />

    {#if step === 0}
      <h1>{t('setup.welcome')}</h1>
      <p class="muted">{t('setup.intro')}</p>
      <div class="field">
        <label for="lang">{t('common.language')}</label>
        <select id="lang" value={i18n.lang} onchange={(e) => setLang(e.currentTarget.value)}>
          {#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
        </select>
      </div>
      <button class="primary" onclick={() => (step = 1)}>{t('setup.start')}</button>
    {:else if step === 1}
      <h2>{t('setup.codeTitle')}</h2>
      <p class="muted">{t(app.boot?.codeOnPanel ? 'setup.codeHelp' : 'setup.codeHelpLog')}</p>
      {#if !app.boot?.codeOnPanel}<pre class="mono small cmd">sh moodnas-sidecar.sh log</pre>{/if}
      <form
        onsubmit={(e) => {
          e.preventDefault();
          step = 2;
        }}
      >
        <div class="field">
          <label for="code">{t('setup.code')}</label>
          <input
            id="code"
            class="mono code"
            bind:value={code}
            placeholder="ABCD-EFGH"
            autocomplete="one-time-code"
            required
            maxlength="9"
          />
        </div>
        {#if error}<p class="error" role="alert">{error}</p>{/if}
        <div class="row"><button type="button" class="ghost" onclick={() => (step = 0)}>{t('common.back')}</button><span class="spacer"></span><button class="primary">{t('common.next')}</button></div>
      </form>
    {:else}
      <h2>{t('setup.adminTitle')}</h2>
      <form onsubmit={finish}>
        <div class="field">
          <label for="host">{t('setup.hostname')}</label>
          <input id="host" bind:value={hostname} pattern={HOST_PATTERN} maxlength="32" required />
        </div>
        <div class="field">
          <label for="user">{t('setup.user')}</label>
          <input id="user" bind:value={user} pattern={USER_PATTERN} maxlength="32" autocomplete="username" required />
          <div class="muted small">{t('user.nameRules')}</div>
        </div>
        <div class="field">
          <label for="pw">{t('setup.password')}</label>
          <input id="pw" type="password" bind:value={password} minlength="10" autocomplete="new-password" required />
          <div class="muted small">{t('user.passwordRules')}</div>
        </div>
        <div class="field">
          <label for="pw2">{t('setup.password2')}</label>
          <input id="pw2" type="password" bind:value={password2} autocomplete="new-password" required />
          {#if mismatch}<div class="error small">{t('setup.mismatch')}</div>{/if}
        </div>
        {#if error}<p class="error" role="alert">{error}</p>{/if}
        <div class="row" style="margin-top:1rem"><button type="button" class="ghost" onclick={() => (step = 1)}>{t('common.back')}</button><span class="spacer"></span><button class="primary" disabled={busy || mismatch}>{t('setup.finish')}</button></div>
      </form>
    {/if}
  </div>
  <p class="muted small foot">MoodNAS · Mood Robotics · {t('setup.footer')}</p>
</div>

<style>
  .wrap {
    min-height: 100vh;
    display: grid;
    place-items: center;
    align-content: center;
    padding: 2rem 1rem;
  }
  .box {
    width: min(460px, 100%);
    text-align: left;
  }
  .box :global(svg) {
    margin: 0 auto;
  }
  .code {
    font-size: 1.6rem;
    letter-spacing: 0.2em;
    text-transform: uppercase;
    text-align: center;
  }
  .cmd {
    background: var(--surface-2);
    border-radius: var(--radius-s);
    padding: 0.5rem 0.8rem;
    overflow-x: auto;
  }
  .foot {
    margin-top: 1rem;
  }
</style>
