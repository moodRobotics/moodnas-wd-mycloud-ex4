<script>
  // The Mood Robotics eyes, animated. Each NAS mood is an expression built
  // from the same parts as the logo: two tilted brow bars, two round eyes,
  // two pupils, plus eyelids for blinking and sleepiness.
  import { Tween } from 'svelte/motion';
  import { cubicOut } from 'svelte/easing';

  // bare: no black frame/padding (used inside the OLED preview)
  let { mood = 'calm', size = 260, follow = true, mono = false, bare = false, label = '' } = $props();

  const expressions = {
    calm: { brow: -22, browY: 20, lid: 0, pupil: 12, px: 0, py: 0 },
    focused: { brow: -6, browY: 30, lid: 0.28, pupil: 11, px: -9, py: 9 },
    sleepy: { brow: -2, browY: 30, lid: 0.78, pupil: 11, px: 0, py: 10 },
    worried: { brow: -34, browY: 16, lid: 0, pupil: 9, px: 0, py: -3 },
    alarmed: { brow: -16, browY: 4, lid: 0, pupil: 6.5, px: 0, py: 0 },
  };

  const opts = { duration: 450, easing: cubicOut };
  const brow = new Tween(expressions.calm.brow, opts);
  const browY = new Tween(expressions.calm.browY, opts);
  const lid = new Tween(0, { duration: 300, easing: cubicOut });
  const pupil = new Tween(12, opts);
  const px = new Tween(0, opts);
  const py = new Tween(0, opts);

  let blink = $state(0);
  let look = $state({ x: 0, y: 0 });
  let svgEl = $state();

  $effect(() => {
    const e = expressions[mood] ?? expressions.calm;
    brow.target = e.brow;
    browY.target = e.browY;
    lid.target = e.lid;
    pupil.target = e.pupil;
  });

  // pupils: mood offset + a little attention to the pointer
  $effect(() => {
    const e = expressions[mood] ?? expressions.calm;
    const k = mood === 'sleepy' ? 0.2 : 1;
    px.target = e.px + look.x * k;
    py.target = e.py + look.y * k;
  });

  // blinking; alarmed blinks fast and nervously, sleepy rarely
  $effect(() => {
    let timer;
    const schedule = () => {
      const base = mood === 'alarmed' ? 1200 : mood === 'sleepy' ? 9000 : 4200;
      timer = setTimeout(
        () => {
          blink = 1;
          setTimeout(() => {
            blink = 0;
            schedule();
          }, 130);
        },
        base + Math.random() * base * 0.6,
      );
    };
    schedule();
    return () => clearTimeout(timer);
  });

  $effect(() => {
    if (!follow) return;
    const onMove = (ev) => {
      if (!svgEl) return;
      const r = svgEl.getBoundingClientRect();
      const dx = ev.clientX - (r.left + r.width / 2);
      const dy = ev.clientY - (r.top + r.height * 0.6);
      const d = Math.hypot(dx, dy) || 1;
      const m = Math.min(1, d / 500) * 13;
      look = { x: (dx / d) * m, y: (dy / d) * m * 0.8 };
    };
    window.addEventListener('pointermove', onMove);
    return () => window.removeEventListener('pointermove', onMove);
  });

  // worried pupils tremble a little
  let jitter = $state(0);
  $effect(() => {
    if (mood !== 'worried' && mood !== 'alarmed') {
      jitter = 0;
      return;
    }
    const id = setInterval(() => (jitter = (Math.random() - 0.5) * (mood === 'alarmed' ? 2.4 : 1.4)), 90);
    return () => clearInterval(id);
  });

  const EYE = { ry: 46, rx: 44, cy: 82 };
  const eyes = [
    { cx: 52, side: 1 },
    { cx: 148, side: -1 },
  ];
  let lidAmount = $derived(Math.max(lid.current, blink));
  let fg = $derived(mono ? '#fff' : 'var(--eye-white, #fff)');
  let glow = $derived(!mono && mood === 'alarmed');
  const uid = Math.random().toString(36).slice(2, 8);
</script>

<svg
  bind:this={svgEl}
  class:glow
  viewBox="0 0 200 135"
  width={size}
  height={size * 0.675}
  style={bare ? 'padding:0' : `padding:${size * 0.06}px; border-radius:${Math.max(8, size * 0.09)}px`}
  role="img"
  aria-label={label}
>
  <defs>
    {#each eyes as eye, i (i)}
      <clipPath id="eye-{uid}-{i}">
        <ellipse cx={eye.cx} cy={EYE.cy} rx={EYE.rx} ry={EYE.ry} />
      </clipPath>
    {/each}
  </defs>

  {#each eyes as eye, i (i)}
    <!-- brow: a bar tilted like the logo's, mirrored for the right eye -->
    <rect
      x={eye.cx - 30}
      y={browY.current}
      width="60"
      height="11"
      fill={fg}
      transform="rotate({brow.current * eye.side} {eye.cx} {browY.current + 5.5})"
    />
    <ellipse cx={eye.cx} cy={EYE.cy} rx={EYE.rx} ry={EYE.ry} fill={fg} />
    <g clip-path="url(#eye-{uid}-{i})">
      <ellipse
        cx={eye.cx + px.current + jitter}
        cy={EYE.cy + py.current}
        rx={pupil.current}
        ry={pupil.current * 1.04}
        fill="#000"
      />
      <!-- eyelid -->
      <rect
        x={eye.cx - EYE.rx - 2}
        y={EYE.cy - EYE.ry - 2}
        width={EYE.rx * 2 + 4}
        height={(EYE.ry * 2 + 4) * lidAmount}
        fill="#000"
      />
    </g>
  {/each}

  {#if mood === 'sleepy'}
    <text x="182" y="22" class="z" fill={fg}>z</text>
    <text x="190" y="10" class="z z2" fill={fg}>z</text>
  {/if}
</svg>

<style>
  /* padding/radius are inline and proportional to `size`: a % padding would
     be relative to the parent's width and crush small instances */
  svg {
    display: block;
    flex-shrink: 0;
    box-sizing: content-box;
    overflow: visible;
    background: #000;
    --eye-white: #fff;
  }
  svg.glow {
    box-shadow:
      0 0 0 2px var(--bad),
      0 0 40px color-mix(in srgb, var(--bad) 45%, transparent);
    animation: pulse 1.4s ease-in-out infinite;
  }
  @keyframes pulse {
    50% {
      box-shadow:
        0 0 0 2px var(--bad),
        0 0 10px color-mix(in srgb, var(--bad) 20%, transparent);
    }
  }
  .z {
    font: 800 14px var(--font);
    opacity: 0.8;
    animation: float 3s ease-in-out infinite;
  }
  .z2 {
    font-size: 10px;
    animation-delay: 1.2s;
  }
  @keyframes float {
    0%,
    100% {
      transform: translateY(0);
      opacity: 0.2;
    }
    50% {
      transform: translateY(-6px);
      opacity: 0.9;
    }
  }
</style>
