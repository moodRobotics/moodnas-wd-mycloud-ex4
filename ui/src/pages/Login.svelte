<script>
  import MoodEyes from '../lib/MoodEyes.svelte';
  import { t, i18n, setLang, languages } from '../lib/i18n.svelte.js';
  import { post } from '../lib/api.js';
  import { app, signedIn } from '../lib/session.svelte.js';

  let user = $state('');
  let password = $state('');
  let totp = $state('');
  let needTotp = $state(false);
  let error = $state('');
  let busy = $state(false);
  let mood = $state('calm');

  async function submit(e) {
    e.preventDefault();
    error = '';
    busy = true;
    mood = 'focused';
    try {
      signedIn(await post('/login', { user, password, totp }));
    } catch (err) {
      if (err.key === 'auth.totpRequired') {
        needTotp = true;
        mood = 'calm';
      } else {
        error = t(err.key, err.params);
        mood = 'worried';
      }
    } finally {
      busy = false;
    }
  }
</script>

<div class="wrap">
  <form class="card box" onsubmit={submit}>
    <MoodEyes {mood} size={180} />
    <h1>{app.boot?.hostname}</h1>
    {#if !needTotp}
      <div class="field">
        <label for="u">{t('login.user')}</label>
        <input id="u" bind:value={user} autocomplete="username" required />
      </div>
      <div class="field">
        <label for="p">{t('login.password')}</label>
        <input id="p" type="password" bind:value={password} autocomplete="current-password" required />
      </div>
    {:else}
      <div class="field">
        <label for="t">{t('login.totp')}</label>
        <input id="t" class="mono" bind:value={totp} inputmode="numeric" autocomplete="one-time-code" maxlength="6" required />
      </div>
    {/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <button class="primary wide" disabled={busy}>{t('login.submit')}</button>
    <select aria-label={t('common.language')} value={i18n.lang} onchange={(e) => setLang(e.currentTarget.value)}>
      {#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
    </select>
  </form>
</div>

<style>
  .wrap {
    min-height: 100vh;
    display: grid;
    place-items: center;
    padding: 1rem;
  }
  .box {
    width: min(380px, 100%);
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
  }
  .box :global(svg) {
    margin: 0 auto;
  }
  h1 {
    text-align: center;
    margin: 0;
  }
  .wide {
    width: 100%;
  }
</style>
