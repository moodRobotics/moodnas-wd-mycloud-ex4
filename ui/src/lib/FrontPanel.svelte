<script>
  // Virtual preview of the EX4's front OLED: what MoodNAS is showing on the
  // real device (eyes + two short lines).
  import MoodEyes from './MoodEyes.svelte';
  import { t } from './i18n.svelte.js';
  let { panel } = $props();
</script>

<figure class="panel">
  <div class="oled" aria-hidden="true">
    <MoodEyes mood={panel?.mood || 'calm'} size={92} follow={false} mono bare />
    <div class="lines">
      <div>{panel?.line1 || ''}</div>
      <div>{panel?.line2 || ''}</div>
    </div>
  </div>
  <figcaption class="muted small">{t('panel.caption')}</figcaption>
</figure>

<style>
  .panel {
    margin: 0;
  }
  .oled {
    display: flex;
    align-items: center;
    gap: 14px;
    background: #000;
    border: 6px solid #1b1d22;
    border-radius: 12px;
    padding: 10px 16px;
    box-shadow: inset 0 0 18px rgba(80, 120, 255, 0.15);
  }
  .lines {
    font-family: var(--mono);
    font-size: 13px;
    color: #e8f0ff;
    text-shadow: 0 0 6px rgba(140, 180, 255, 0.6);
    letter-spacing: 0.05em;
    line-height: 1.6;
    white-space: nowrap;
    overflow: hidden;
  }
  figcaption {
    margin-top: 0.4rem;
  }
</style>
