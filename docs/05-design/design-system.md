---
version: alpha
name: yieldforge-design-system
concept: Aurora
description: A premium, dark, luminous interface for a Brazilian investment copilot (FIIs + fixed income). A near-black canvas is washed by soft aurora glows — colored radial gradients bleeding from the edges — with one warm gold accent for actions, glass cards that cast a colored ambient shadow, and a signature spectrum gradient repurposed as the allocation-by-sector bar. Display type runs Fraunces (an expressive high-contrast serif) over Inter body and IBM Plex Mono numbers. Dark-first: an optional light theme keeps the gold, gradient bar, and semantics but drops the ambient glow. Money renders as pt-BR strings (R$ 1.234,56 / 10,50%) from integer centavos/basis-points at the render edge, never a float. Two components encode the product's binding guards structurally: an InsightCard whose explanation slot is non-optional (explainability, FR-013) and a NonAdviceDisclaimer required on any AI surface (non-advice, FR-014). The glow is decorative and used sparingly — it never colors data.

colors:
  primary: "#e9a94c"
  primary-active: "#d3913a"
  primary-tint: "#f4c47e"
  primary-disabled: "#2a241a"
  on-primary: "#120f0a"
  canvas: "#0a0a0c"
  surface: "#141317"
  elevated: "#1c1b21"
  hairline: "#26242b"
  border-strong: "#3a3742"
  body: "#e7e5e4"
  on-dark: "#ffffff"
  muted: "#8b8681"
  muted-strong: "#b6b1ab"
  gain: "#34d399"
  loss: "#fb7185"
  caution: "#fbbf24"
  info: "#38bdf8"
  info-ring: "#38bdf8"
  aurora-1: "#6366f1"
  aurora-2: "#d946ef"
  aurora-3: "#f59e0b"
  aurora-4: "#34d399"
  aurora-5: "#38bdf8"
  canvas-light: "#faf9f7"
  surface-light: "#ffffff"
  ink: "#1c1a17"
  body-on-light: "#3a3630"
  hairline-light: "#eae7e1"

gradients:
  aurora-bg: "radial-gradient(55% 45% at 16% 8%, rgba(99,102,241,.20), transparent 60%), radial-gradient(48% 42% at 88% 18%, rgba(217,70,239,.16), transparent 60%), radial-gradient(55% 48% at 76% 98%, rgba(56,189,248,.14), transparent 60%), radial-gradient(40% 40% at 42% 58%, rgba(233,169,76,.10), transparent 60%)"
  spectrum-bar: "linear-gradient(90deg, {colors.aurora-1}, {colors.aurora-2}, {colors.aurora-3}, {colors.aurora-4}, {colors.aurora-5})"
  card-sheen: "linear-gradient(180deg, rgba(255,255,255,.04), transparent)"
  edge-accent: "linear-gradient(180deg, {colors.primary}, {colors.aurora-2})"

glow:
  gold-cta: "0 0 30px -6px rgba(233,169,76,.60)"
  gold-outline: "0 0 26px -8px rgba(233,169,76,.55)"
  card-ambient: "0 20px 50px -24px rgba(99,102,241,.50)"
  card-ambient-strong: "0 30px 70px -30px rgba(99,102,241,.60)"
  info-soft: "0 0 30px -12px rgba(56,189,248,.50)"

typography:
  hero-display:
    fontFamily: "Fraunces, Georgia, 'Times New Roman', serif"
    fontSize: 56px
    fontWeight: 600
    lineHeight: 1.08
    letterSpacing: 0.3px
  display-lg:
    fontFamily: "Fraunces, Georgia, serif"
    fontSize: 44px
    fontWeight: 600
    lineHeight: 1.1
    letterSpacing: 0.2px
  display-md:
    fontFamily: "Fraunces, Georgia, serif"
    fontSize: 36px
    fontWeight: 600
    lineHeight: 1.15
    letterSpacing: 0
  title-lg:
    fontFamily: "Fraunces, Georgia, serif"
    fontSize: 22px
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: 0
  title-md:
    fontFamily: "Inter, -apple-system, sans-serif"
    fontSize: 18px
    fontWeight: 600
    lineHeight: 1.35
    letterSpacing: 0
  title-sm:
    fontFamily: "Inter, sans-serif"
    fontSize: 15px
    fontWeight: 600
    lineHeight: 1.4
    letterSpacing: 0
  number-display:
    fontFamily: "'IBM Plex Mono', ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: 40px
    fontWeight: 600
    lineHeight: 1.1
    letterSpacing: -0.2px
  number-md:
    fontFamily: "'IBM Plex Mono', ui-monospace, monospace"
    fontSize: 16px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: 0
  number-sm:
    fontFamily: "'IBM Plex Mono', ui-monospace, monospace"
    fontSize: 14px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: 0
  body-md:
    fontFamily: "Inter, sans-serif"
    fontSize: 15px
    fontWeight: 400
    lineHeight: 1.6
    letterSpacing: 0
  body-sm:
    fontFamily: "Inter, sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.55
    letterSpacing: 0
  caption:
    fontFamily: "Inter, sans-serif"
    fontSize: 12px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: 0.2px
  button:
    fontFamily: "Inter, sans-serif"
    fontSize: 14px
    fontWeight: 600
    lineHeight: 1
    letterSpacing: 0
  nav-link:
    fontFamily: "Inter, sans-serif"
    fontSize: 14px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: 0

rounded:
  xs: 4px
  sm: 8px
  md: 10px
  lg: 16px
  xl: 20px
  pill: 9999px
  full: 9999px

spacing:
  xxs: 4px
  xs: 8px
  sm: 12px
  md: 16px
  lg: 24px
  xl: 32px
  xxl: 48px
  section: 80px

components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    padding: 13px 22px
    height: 44px
    shadow: "{glow.gold-cta}"
  button-primary-active:
    backgroundColor: "{colors.primary-active}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.md}"
  button-primary-disabled:
    backgroundColor: "{colors.primary-disabled}"
    textColor: "#6f6552"
    rounded: "{rounded.md}"
  button-outline-gold:
    backgroundColor: "rgba(233,169,76,.06)"
    textColor: "{colors.primary-tint}"
    border: "1px solid rgba(233,169,76,.50)"
    rounded: "{rounded.md}"
    padding: 13px 22px
    height: 44px
    shadow: "{glow.gold-outline}"
  button-secondary:
    backgroundColor: "{colors.elevated}"
    textColor: "{colors.on-dark}"
    border: "1px solid {colors.hairline}"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    padding: 13px 22px
  button-tertiary-text:
    backgroundColor: transparent
    textColor: "{colors.muted-strong}"
    typography: "{typography.button}"
  text-link:
    backgroundColor: transparent
    textColor: "{colors.primary-tint}"
    typography: "{typography.body-md}"
  app-sidebar:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.body}"
    typography: "{typography.nav-link}"
    padding: 16px
    width: 248px
  app-sidebar-item-active:
    backgroundColor: "{colors.elevated}"
    textColor: "{colors.on-dark}"
    borderLeft: "2px solid {colors.primary}"
    typography: "{typography.nav-link}"
    rounded: "{rounded.sm}"
    padding: 10px 12px
  glass-card:
    backgroundColor: "{gradients.card-sheen} , {colors.surface}"
    border: "1px solid {colors.hairline}"
    rounded: "{rounded.xl}"
    padding: 24px
    shadow: "{glow.card-ambient}"
  balance-card:
    backgroundColor: "{gradients.card-sheen} , {colors.surface}"
    border: "1px solid {colors.hairline}"
    rounded: "{rounded.xl}"
    padding: 26px 28px
    shadow: "{glow.card-ambient-strong}"
  metric-callout:
    backgroundColor: transparent
    textColor: "{colors.on-dark}"
    typography: "{typography.number-display}"
  metric-callout-label:
    backgroundColor: transparent
    textColor: "{colors.muted}"
    typography: "{typography.caption}"
  value-cell-gain:
    backgroundColor: transparent
    textColor: "{colors.gain}"
    typography: "{typography.number-md}"
  value-cell-loss:
    backgroundColor: transparent
    textColor: "{colors.loss}"
    typography: "{typography.number-md}"
  allocation-bar-spectrum:
    background: "{gradients.spectrum-bar}"
    rounded: "{rounded.pill}"
    height: 14px
    shadow: "0 0 24px -2px rgba(99,102,241,.5)"
  allocation-legend-item:
    backgroundColor: transparent
    textColor: "{colors.body}"
    typography: "{typography.body-sm}"
  holding-row:
    backgroundColor: transparent
    textColor: "{colors.on-dark}"
    typography: "{typography.number-md}"
    padding: 14px 0
    divider: "1px solid {colors.hairline}"
  health-score-gauge:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-dark}"
    typography: "{typography.number-display}"
    rounded: "{rounded.xl}"
    padding: 24px
    shadow: "{glow.card-ambient}"
  health-factor-row:
    backgroundColor: transparent
    textColor: "{colors.body}"
    typography: "{typography.body-md}"
    padding: 12px 0
  insight-card:
    backgroundColor: "{gradients.card-sheen} , {colors.surface}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.lg}"
    padding: 22px
    edgeAccent: "3px {gradients.edge-accent} (glowing)"
    shadow: "{glow.card-ambient}"
  insight-card-explanation:
    backgroundColor: "{colors.elevated}"
    textColor: "{colors.muted-strong}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.md}"
    padding: 12px 14px
  non-advice-disclaimer:
    backgroundColor: "{colors.elevated}"
    textColor: "{colors.muted-strong}"
    border: "1px solid {colors.hairline}"
    typography: "{typography.caption}"
    rounded: "{rounded.md}"
    padding: 11px 14px
  projection-chart-card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.xl}"
    padding: 24px
    shadow: "{glow.card-ambient}"
  chat-bubble-user:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.body-md}"
    rounded: "{rounded.lg}"
    padding: 12px 16px
  chat-bubble-assistant:
    backgroundColor: "{colors.elevated}"
    textColor: "{colors.body}"
    border: "1px solid {colors.hairline}"
    typography: "{typography.body-md}"
    rounded: "{rounded.lg}"
    padding: 12px 16px
  chat-input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-dark}"
    border: "1px solid {colors.hairline}"
    typography: "{typography.body-md}"
    rounded: "{rounded.lg}"
    padding: 12px 16px
    height: 48px
  badge-neutral:
    backgroundColor: "{colors.elevated}"
    textColor: "{colors.body}"
    typography: "{typography.caption}"
    rounded: "{rounded.pill}"
    padding: 4px 10px
  badge-caution:
    backgroundColor: "rgba(251,191,36,.12)"
    textColor: "{colors.caution}"
    border: "1px solid rgba(251,191,36,.25)"
    typography: "{typography.caption}"
    rounded: "{rounded.pill}"
    padding: 4px 10px
  text-input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-dark}"
    border: "1px solid {colors.hairline}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 11px 14px
    height: 44px
    focusRing: "0 0 0 2px {colors.info-ring}"
  objective-chip:
    backgroundColor: "{colors.elevated}"
    textColor: "{colors.body}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.pill}"
    padding: 8px 14px
  objective-chip-selected:
    backgroundColor: "rgba(233,169,76,.12)"
    textColor: "{colors.primary-tint}"
    border: "1px solid rgba(233,169,76,.4)"
    typography: "{typography.body-sm}"
    rounded: "{rounded.pill}"
    padding: 8px 14px
  alert-error:
    backgroundColor: "rgba(251,113,133,.06)"
    textColor: "{colors.loss}"
    border: "1px solid rgba(251,113,133,.30)"
    typography: "{typography.body-sm}"
    rounded: "{rounded.md}"
    padding: 12px 16px
  empty-state:
    backgroundColor: transparent
    textColor: "{colors.muted}"
    typography: "{typography.body-md}"
  loading-skeleton:
    backgroundColor: "{colors.elevated}"
    rounded: "{rounded.md}"
---

## Overview

YieldForge is a **personal investment copilot** for Brazilian retail investors — FIIs and fixed
income, reasoned about *as a whole portfolio*, with plain-language explanations the user can trust.
The **Aurora** concept gives that a premium, luminous skin: a **near-black canvas** washed by soft
**aurora glows** (colored radial gradients bleeding from the edges — the "illuminated shadow"),
**glass cards** that cast a colored ambient shadow, and one **warm gold** accent for actions.
Display type is an expressive high-contrast serif (**Fraunces**), which reads elegant and
considered — the opposite of a hype-y trading terminal.

It is **dark-first** by nature. An optional **light theme** keeps the gold accent, the semantic
gain/loss colors, and the signature spectrum bar, but **drops the ambient glow** (glows don't
translate to a light surface — there, depth comes from the hairline + surface system instead).

The design's signature is a **spectrum gradient** (indigo → fuchsia → amber → emerald → sky) that
is not decoration for its own sake: it is repurposed as the **allocation-by-sector bar**, a real
product component. Everywhere else, the glow is used **sparingly** (behind hero figures and key
cards) and **never colors data** — gain/loss and sector identity always stay legible; the aurora
reads as depth, not signal.

**Key characteristics:**

- **Single accent — warm gold** (`{colors.primary}`) for primary actions, often as a **glowing
  outline** (`{component.button-outline-gold}`) rather than a heavy fill. `{colors.primary-tint}`
  is the lighter sibling for gold-as-text/links.
- **Aurora glow** (`{gradients.aurora-bg}`) as an ambient background layer behind key surfaces;
  **glass cards** (`{component.glass-card}`) with a subtle top sheen and a colored ambient shadow
  (`{glow.card-ambient}`).
- **Semantic colors reserved:** `{colors.gain}` / `{colors.loss}` as figure text color only;
  `{colors.caution}` (amber) for a risk badge; `{colors.info}` (sky) for info + the focus ring.
  None is ever brand voltage or a card fill.
- **Three fonts:** Fraunces (display serif) for headlines, Inter for body/UI, IBM Plex Mono for
  every number (tabular figures so money columns align). All three are SIL Open Font License.
- **Money is integer, formatted at the edge:** centavos/basis points arrive as integers and render
  as pt-BR strings (`R$ 1.234,56`, `10,50%`) only at the render boundary — no float ever represents
  a balance or rate (ADR-0006, FR-2005).
- **Guards are components, not guidelines:** `{component.insight-card}` cannot render without its
  explanation slot; `{component.non-advice-disclaimer}` is required on every AI surface.
- **No order affordances:** there are deliberately **no Buy/Sell/Long/Short buttons**, no price
  targets, no quantity CTAs. A copilot never issues a transaction order (FR-014).
- **Softer, rounder shapes:** medium radii (`{rounded.md}` 10px buttons, `{rounded.xl}` 20px cards)
  for a calm, premium feel.

## Colors

### Brand & Signature
- **Gold** (`{colors.primary}` — #e9a94c): the single brand accent — primary actions, active nav
  edge, selected chips, the user chat bubble. Prefer the **glowing outline** treatment for a
  lighter, more premium touch; use the fill for the single most important action on a view.
- **Gold Active / Tint** (`{colors.primary-active}` #d3913a / `{colors.primary-tint}` #f4c47e):
  pressed state and gold-as-text (links, labels-on-dark).
- **Aurora gradient** (`{colors.aurora-1..5}`): the signature spectrum, used as the
  `{component.allocation-bar-spectrum}` and, softened, as the ambient `{gradients.aurora-bg}`.

### Surface & Text (dark, default)
- **Canvas** (`{colors.canvas}` — #0a0a0c): the app floor, sitting *under* the glow layer.
- **Surface** (`{colors.surface}` — #141317): the glass-card base.
- **Elevated** (`{colors.elevated}` — #1c1b21): nested cards, assistant bubble, inputs.
- **Body** (`{colors.body}` — #e7e5e4) primary text; **Muted Strong** (`{colors.muted-strong}` —
  #b6b1ab) explanation copy; **Muted** (`{colors.muted}` — #8b8681) labels/captions.
- **Hairline** (`{colors.hairline}` — #26242b) dividers; **Border Strong** (`{colors.border-strong}`)
  where a heavier edge is needed.

### Semantic (reserved — never brand, never a fill)
- **Gain** (`{colors.gain}` — #34d399) / **Loss** (`{colors.loss}` — #fb7185): figure direction, as
  text color, always paired with a ▲/▼ arrow (never color alone — accessibility).
- **Caution** (`{colors.caution}` — #fbbf24): a small risk/attention badge.
- **Info / Focus** (`{colors.info}` — #38bdf8): info badges and the keyboard focus ring.

### Light theme (optional)
`{colors.canvas-light}` #faf9f7 · `{colors.surface-light}` #ffffff · `{colors.ink}` #1c1a17 ·
`{colors.body-on-light}` #3a3630 · `{colors.hairline-light}` #eae7e1. Keeps gold + spectrum bar +
semantics; **omits** the ambient glows.

## Typography

**Fraunces** (display serif) · **Inter** (body/UI) · **IBM Plex Mono** (numbers). All SIL Open Font
License — free to embed and ship (ADR-0003). Every number is IBM Plex Mono with **tabular figures**
so money columns align and values don't reflow.

| Token | Size | Weight | Font | Use |
|---|---|---|---|---|
| `{typography.hero-display}` | 56px | 600 | Fraunces | Onboarding / marketing hero |
| `{typography.display-lg}` | 44px | 600 | Fraunces | Big page headline |
| `{typography.display-md}` | 36px | 600 | Fraunces | Section head |
| `{typography.title-lg}` | 22px | 600 | Fraunces | Card / sub-section title |
| `{typography.title-md}` | 18px | 600 | Inter | Dense card title |
| `{typography.title-sm}` | 15px | 600 | Inter | Row / badge label |
| `{typography.number-display}` | 40px | 600 | IBM Plex Mono | Hero figures — net worth, income |
| `{typography.number-md}` | 16px | 500 | IBM Plex Mono | Table figures, holdings |
| `{typography.number-sm}` | 14px | 500 | IBM Plex Mono | Inline figures, % changes |
| `{typography.body-md}` | 15px | 400 | Inter | Running text, insight copy |
| `{typography.body-sm}` | 13px | 400 | Inter | Explanations, captions |
| `{typography.caption}` | 12px | 500 | Inter | Meta labels, disclaimer |
| `{typography.button}` | 14px | 600 | Inter | Button labels |
| `{typography.nav-link}` | 14px | 500 | Inter | Nav items |

> Serif for display, sans for everything functional. Fraunces carries brand voice in headlines;
> the moment type gets dense or numeric it switches to Inter / IBM Plex Mono for legibility.

## Layout

- **Spacing** base 4px; bands at `{spacing.section}` (80px) — airy, premium rhythm.
- **App shell:** fixed `{component.app-sidebar}` (248px) + fluid content, max ~1200px on data views.
- **Dashboard:** a `{component.balance-card}` hero (net worth + spectrum allocation), then a grid of
  `{component.glass-card}` (sector exposure, health score, insights).
- **Forms:** single-column, centered (~560px); glow dialed back so inputs stay crisp.

## Elevation & Depth

| Level | Treatment | Use |
|---|---|---|
| Ambient | `{gradients.aurora-bg}` behind the view (fixed, non-interactive) | App background, hero areas |
| Glass card | `{colors.surface}` + `{gradients.card-sheen}` + `{glow.card-ambient}` | All cards |
| Glowing edge | 3px `{gradients.edge-accent}` with a soft glow | `{component.insight-card}` left edge |
| CTA glow | `{glow.gold-cta}` / `{glow.gold-outline}` | Primary + outline buttons |
| Focus ring | `0 0 0 2px {colors.info-ring}` | Keyboard focus |

Depth = the aurora glow + colored ambient card shadows + the surface step. Used **sparingly**; on
the light theme the glows are removed and depth falls back to hairlines + surface steps.

## Shapes

`{rounded.xs}` 4 · `{rounded.sm}` 8 · `{rounded.md}` 10 (buttons, inputs) · `{rounded.lg}` 16
(insight/chat) · `{rounded.xl}` 20 (cards) · `{rounded.pill}` (chips, badges, the spectrum bar).

## Components

### Buttons
- **`button-primary`** — gold fill, dark text, a soft `{glow.gold-cta}`. One per view.
- **`button-outline-gold`** — the preferred lighter treatment: gold text + gold border + glow.
- **`button-secondary`**, **`button-tertiary-text`**, **`text-link`** (gold-tint).

> **Deliberately absent:** no Buy/Sell/order button. The copilot never issues an order (FR-014);
> gain/loss are figure colors, not actions.

### Signature — Aurora & the spectrum bar
- **`allocation-bar-spectrum`** — the luminous spectrum bar as the **allocation-by-sector** view,
  with `{component.allocation-legend-item}` mapping each color to a sector + pt-BR %.
- **`balance-card`** / **`glass-card`** — glass surfaces with sheen + colored ambient glow; the
  `balance-card` anchors the dashboard hero (net worth in `{typography.number-display}`).

### Portfolio & Figures
- **`metric-callout`** + **`value-cell-gain/loss`** — hero figures and colored growth cells.
- **`holding-row`** — one FII / fixed-income holding, numbers in IBM Plex Mono, hairline divider.

### Health Score
- **`health-score-gauge`** — the 0–100 score as a radial gauge with a big mono center; arc uses a
  muted→gold→gain sweep (a health signal, not a pass/fail binary).
- **`health-factor-row`** — one factor with a bar and a `{component.badge-caution}` when it needs
  attention.

### AI Surfaces (guarded)
- **`insight-card`** — glass surface, a **glowing gradient left edge**, and a **non-optional**
  `{component.insight-card-explanation}` slot (the "por quê"). An insight without its explanation is
  unrepresentable (FR-013), mirroring the backend gate.
- **`non-advice-disclaimer`** — a required, quiet footer on every AI surface (FR-014):
  "Isto é conteúdo educacional, não recomendação de investimento."
- **`chat-bubble-user` / `-assistant`** + **`chat-input`** — the copilot dialogue (user bubble gold,
  assistant on elevated glass), streaming token-by-token; every turn ends with the disclaimer.

### Forms & Feedback
- **`text-input`** (sky focus ring), **`objective-chip` / `-selected`** (gold-tinted), **`badge-*`**,
  **`alert-error`** (`{"error":"..."}` envelope), **`empty-state`**, **`loading-skeleton`**.

## Do's and Don'ts

### Do
- Reserve gold for actions and brand moments; prefer the glowing **outline** for a premium touch.
- Use the aurora glow **sparingly** — behind hero figures and key cards — and never over data.
- Render the spectrum bar as the allocation view; render every number in IBM Plex Mono, tabular.
- Format money/rates as pt-BR strings at the render edge from integer centavos/bps.
- Always pair an AI insight with its explanation and a `{component.non-advice-disclaimer}`.
- Use gain/loss only as figure text color, with a direction arrow (never color alone).

### Don't
- Don't add a second brand accent; don't let the aurora tint text or data figures.
- Don't use gain/loss/caution/info as card or button fills, or as brand accents.
- Don't introduce Buy/Sell/order buttons, price targets, or quantity CTAs (FR-014).
- Don't render an AI insight/chat reply/rebalancing suggestion without its explanation + disclaimer.
- Don't put money through a float on the client, and don't hand-format currency — use the helper.
- Don't carry the ambient glow onto the light theme — drop it; use hairlines + surface steps there.

## Responsive Behavior

| Name | Width | Key changes |
|---|---|---|
| Mobile | < 768px | Sidebar → bottom tabs; cards stack 1-up; holdings reflow to stacked cards; chat full-screen; glow intensity reduced (perf + legibility) |
| Tablet | 768–1024px | Sidebar as icon rail; dashboard 2-up |
| Desktop | 1024–1440px | Full 248px sidebar; dashboard 2–3-up; chat as right rail or view |
| Wide | > 1440px | Content caps ~1200px, more outer breathing room |

Primary controls ≥ 44×44. Numbers never wrap — a large figure shrinks a step rather than breaking.
Theme toggle available at every breakpoint.

## Iteration Guide

1. Work on ONE component at a time; reference its YAML key (`{component.insight-card}`).
2. The system is dark-first; when adding a component, define the dark treatment first, then the
   light-theme fallback (which drops glows).
3. Variants (`-active`, `-selected`, `-disabled`) are separate `components:` entries.
4. Use `{token.refs}` everywhere prose names a color, gradient, glow, radius, type role, or spacing.
5. Numbers → IBM Plex Mono; headlines → Fraunces; everything else → Inter.
6. Glow is decorative and sparing — it never colors data; gain/loss/caution/info stay semantic.
7. Any AI-output component must include an explanation slot and require the disclaimer.

## Known Gaps / Open

- **Font loading** (self-host vs. Google Fonts; Fraunces is a variable font — pin the `opsz`/`wght`
  axes) deferred to SPEC-200.
- **Chart theming** (Recharts) — the three projection scenarios (pessimista/base/otimista) need one
  hue at three intensities, distinct from the aurora spectrum and from gain/loss.
- **Light theme** — tokens are drafted but the full light treatment (per-component) is unbuilt;
  confirm during SPEC-200 whether the app ships dark-only for MVP or both.
- **Glow performance** — large blurred radial gradients can cost paint on low-end devices; validate
  and provide a reduced-motion / reduced-glow fallback.
- **Accessibility** — full WCAG AA contrast sweep over the dark canvas (esp. muted text and
  gold-on-dark), and colorblind-safe gain/loss (arrows + text) — during SPEC-200.
