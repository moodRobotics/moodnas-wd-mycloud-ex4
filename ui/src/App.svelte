<script>
  import { app, init, logout, refreshStatus, isAdmin } from './lib/session.svelte.js';
  import { t, i18n, setLang, languages } from './lib/i18n.svelte.js';
  import MoodEyes from './lib/MoodEyes.svelte';
  import Setup from './pages/Setup.svelte';
  import Login from './pages/Login.svelte';
  import Dashboard from './pages/Dashboard.svelte';
  import Storage from './pages/Storage.svelte';
  import Shares from './pages/Shares.svelte';
  import Users from './pages/Users.svelte';
  import AI from './pages/AI.svelte';
  import Security from './pages/Security.svelte';
  import System from './pages/System.svelte';
  import Account from './pages/Account.svelte';

  const routes = [
    { id: 'dashboard', icon: '◉', page: Dashboard },
    { id: 'storage', icon: '▤', page: Storage },
    { id: 'shares', icon: '⌂', page: Shares },
    { id: 'ai', icon: '✦', page: AI },
    { id: 'security', icon: '⛨', page: Security, admin: true },
    { id: 'users', icon: '☺', page: Users, admin: true },
    { id: 'system', icon: '⚙', page: System, admin: true },
    { id: 'account', icon: '◎', page: Account },
  ];

  let hash = $state(location.hash.slice(2) || 'dashboard');
  $effect(() => {
    const on = () => (hash = location.hash.slice(2) || 'dashboard');
    window.addEventListener('hashchange', on);
    return () => window.removeEventListener('hashchange', on);
  });

  let visible = $derived(routes.filter((r) => !r.admin || isAdmin()));
  let current = $derived(visible.find((r) => r.id === hash) ?? visible[0]);
  let menuOpen = $state(false);

  let theme = $state(localStorage.getItem('moodnas.theme') || 'dark');
  $effect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem('moodnas.theme', theme);
  });

  init();

  // live status: every 3 s while signed in
  $effect(() => {
    if (!app.me) return;
    refreshStatus();
    const id = setInterval(refreshStatus, 3000);
    return () => clearInterval(id);
  });

  let mood = $derived(app.status?.mood?.state ?? 'calm');
  $effect(() => {
    document.title = `MoodNAS · ${t('mood.state.' + mood)}`;
  });
</script>

{#if !app.ready}
  <div class="center"><MoodEyes mood="sleepy" size={180} /></div>
{:else if app.boot?.setupRequired && !app.me}
  <Setup />
{:else if !app.me}
  <Login />
{:else}
  <div class="shell">
    <aside class:open={menuOpen}>
      <a class="brand" href="#/dashboard" onclick={() => (menuOpen = false)}>
        <MoodEyes {mood} size={56} follow={false} label={t('mood.state.' + mood)} />
        <div>
          <div class="name">Mood<span>NAS</span></div>
          <div class="host muted small">{app.boot?.hostname}</div>
        </div>
      </a>
      <nav aria-label={t('nav.label')}>
        {#each visible as r (r.id)}
          <a
            href="#/{r.id}"
            class:active={current.id === r.id}
            aria-current={current.id === r.id ? 'page' : undefined}
            onclick={() => (menuOpen = false)}
          >
            <span class="icon" aria-hidden="true">{r.icon}</span>{t('nav.' + r.id)}
          </a>
        {/each}
      </nav>
      <div class="foot">
        <label class="sr-only" for="lang">{t('common.language')}</label>
        <select id="lang" value={i18n.lang} onchange={(e) => setLang(e.currentTarget.value)}>
          {#each languages as l (l.code)}<option value={l.code}>{l.name}</option>{/each}
        </select>
        <div class="row">
          <button class="ghost small" onclick={() => (theme = theme === 'dark' ? 'light' : 'dark')}>
            {theme === 'dark' ? '☀' : '☾'} {t('common.theme')}
          </button>
          <button class="ghost small" onclick={logout}>{t('common.logout')}</button>
        </div>
        <div class="muted small">{app.me.user} · {t('role.' + app.me.role)}</div>
        <div class="muted small">by Mood Robotics · v{app.boot?.version}</div>
      </div>
    </aside>
    <main>
      <button class="menu ghost" onclick={() => (menuOpen = !menuOpen)} aria-expanded={menuOpen}>
        ☰ {t('nav.menu')}
      </button>
      {#if app.boot?.simulator}
        <div class="sim small">{t('sim.banner')}</div>
      {/if}
      {#key current.id}
        <current.page />
      {/key}
    </main>
  </div>
{/if}

<style>
  .center {
    min-height: 100vh;
    display: grid;
    place-items: center;
  }
  .shell {
    display: grid;
    grid-template-columns: 250px 1fr;
    min-height: 100vh;
  }
  aside {
    background: var(--surface);
    border-right: 1px solid var(--line);
    padding: 1.2rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
    position: sticky;
    top: 0;
    height: 100vh;
  }
  .brand {
    display: flex;
    gap: 0.8rem;
    align-items: center;
    text-decoration: none;
    color: inherit;
  }
  .brand :global(svg) {
    border-radius: 14px;
  }
  .name {
    font-weight: 900;
    font-size: 1.35rem;
    letter-spacing: -0.02em;
  }
  .name span {
    color: var(--accent);
  }
  nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  nav a {
    display: flex;
    gap: 0.7rem;
    align-items: center;
    padding: 0.55rem 0.8rem;
    border-radius: 12px;
    color: var(--text);
    text-decoration: none;
    font-weight: 700;
  }
  nav a:hover {
    background: var(--surface-2);
  }
  nav a.active {
    background: var(--accent);
    color: #fff;
  }
  .icon {
    width: 1.2em;
    text-align: center;
    opacity: 0.85;
  }
  .foot {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  main {
    padding: 2rem clamp(1rem, 3vw, 2.5rem) 3rem;
    max-width: 1280px;
    width: 100%;
  }
  .menu {
    display: none;
    margin-bottom: 1rem;
  }
  .sim {
    background: repeating-linear-gradient(-45deg, var(--surface-2), var(--surface-2) 10px, var(--surface) 10px, var(--surface) 20px);
    border: 1px dashed var(--line);
    border-radius: 999px;
    padding: 0.3rem 1rem;
    margin-bottom: 1.2rem;
    display: inline-block;
  }
  @media (max-width: 860px) {
    .shell {
      grid-template-columns: 1fr;
    }
    aside {
      position: fixed;
      inset: 0 auto 0 0;
      width: 270px;
      z-index: 10;
      transform: translateX(-100%);
      transition: transform 0.2s;
    }
    aside.open {
      transform: none;
      box-shadow: 0 0 60px #000;
    }
    .menu {
      display: inline-block;
    }
  }
</style>
