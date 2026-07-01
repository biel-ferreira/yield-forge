---
version: alpha
name: yieldforge-design-system
description: A calm, modern-fintech interface for a Brazilian investment copilot (FIIs + fixed income). One token set drives a light and a dark theme; a confident indigo (#4f46e5) carries every primary action and brand moment, held apart from the semantic gain-green / loss-red / info-cyan lanes so a primary button is never misread as an "up" or "info" signal. Type runs Inter (display + body) and IBM Plex Mono (numbers / financial data) — the system trusts size, tabular numerals, and a single accent over heavy weight or hype. Money renders as pt-BR strings (R$ 1.234,56 / 10,50%) from integer centavos/basis-points at the render edge, never a float. Two components encode the product's binding guards structurally: an InsightCard whose explanation slot is non-optional (explainability, FR-013) and a NonAdviceDisclaimer required on any AI surface (non-advice, FR-014).

colors:
  primary: "#4f46e5"
  primary-active: "#4338ca"
  primary-disabled: "#262546"
  primary-tint: "#818cf8"
  ink: "#181a20"
  body: "#eaecef"
  body-on-light: "#181a20"
  muted: "#707a8a"
  muted-strong: "#929aa5"
  hairline-on-light: "#eaecef"
  hairline-on-dark: "#2b3139"
  border-strong: "#cdd1d6"
  canvas-light: "#ffffff"
  canvas-dark: "#0b0e11"
  surface-card-dark: "#1e2329"
  surface-elevated-dark: "#2b3139"
  surface-soft-light: "#fafafa"
  surface-strong-light: "#f5f5f5"
  on-primary: "#ffffff"
  on-dark: "#ffffff"
  gain: "#0ecb81"
  loss: "#f6465d"
  caution: "#f59e0b"
  info: "#06b6d4"
  info-ring: "#06b6d4"

typography:
  hero-display:
    fontFamily: "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
    fontSize: 56px
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: -1px
  display-lg:
    fontFamily: "Inter, sans-serif"
    fontSize: 44px
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: -0.5px
  display-md:
    fontFamily: "Inter, sans-serif"
    fontSize: 36px
    fontWeight: 600
    lineHeight: 1.15
    letterSpacing: -0.3px
  display-sm:
    fontFamily: "Inter, sans-serif"
    fontSize: 30px
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: 0
  title-lg:
    fontFamily: "Inter, sans-serif"
    fontSize: 24px
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: 0
  title-md:
    fontFamily: "Inter, sans-serif"
    fontSize: 20px
    fontWeight: 600
    lineHeight: 1.35
    letterSpacing: 0
  title-sm:
    fontFamily: "Inter, sans-serif"
    fontSize: 16px
    fontWeight: 600
    lineHeight: 1.4
    letterSpacing: 0
  number-display:
    fontFamily: "'IBM Plex Mono', ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: 40px
    fontWeight: 600
    lineHeight: 1.1
    letterSpacing: -0.3px
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
    lineHeight: 1.55
    letterSpacing: 0
  body-sm:
    fontFamily: "Inter, sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.5
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
  xs: 2px
  sm: 4px
  md: 8px
  lg: 12px
  xl: 16px
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
  section: 64px

components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    padding: 12px 20px
    height: 40px
  button-primary-active:
    backgroundColor: "{colors.primary-active}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.md}"
  button-primary-disabled:
    backgroundColor: "{colors.primary-disabled}"
    textColor: "{colors.muted}"
    rounded: "{rounded.md}"
  button-primary-pill:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: 12px 28px
  button-secondary-on-dark:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    padding: 12px 20px
  button-secondary-on-light:
    backgroundColor: "{colors.canvas-light}"
    textColor: "{colors.ink}"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    padding: 12px 20px
  button-tertiary-text:
    backgroundColor: transparent
    textColor: "{colors.body}"
    typography: "{typography.button}"
  text-link-on-dark:
    backgroundColor: transparent
    textColor: "{colors.primary-tint}"
    typography: "{typography.body-md}"
  text-link-on-light:
    backgroundColor: transparent
    textColor: "{colors.primary}"
    typography: "{typography.body-md}"
  app-sidebar:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.body}"
    typography: "{typography.nav-link}"
    padding: 16px
    width: 248px
  app-sidebar-item-active:
    backgroundColor: "{colors.surface-elevated-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.nav-link}"
    rounded: "{rounded.md}"
    padding: 10px 12px
  top-nav-dark:
    backgroundColor: "{colors.canvas-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.nav-link}"
    height: 64px
  top-nav-light:
    backgroundColor: "{colors.canvas-light}"
    textColor: "{colors.ink}"
    typography: "{typography.nav-link}"
    height: 64px
  content-card-dark:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.xl}"
    padding: 24px
  content-card-light:
    backgroundColor: "{colors.canvas-light}"
    textColor: "{colors.ink}"
    typography: "{typography.body-md}"
    rounded: "{rounded.xl}"
    padding: 24px
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
  holding-row:
    backgroundColor: transparent
    textColor: "{colors.on-dark}"
    typography: "{typography.number-md}"
    padding: 14px 0
  portfolio-summary-card:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.xl}"
    padding: 24px
  allocation-donut-card:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.xl}"
    padding: 24px
  allocation-legend-item:
    backgroundColor: transparent
    textColor: "{colors.body}"
    typography: "{typography.body-sm}"
  health-score-gauge:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.number-display}"
    rounded: "{rounded.xl}"
    padding: 24px
  health-factor-row:
    backgroundColor: transparent
    textColor: "{colors.body}"
    typography: "{typography.body-md}"
    padding: 12px 0
  insight-card:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.lg}"
    padding: 20px
    borderLeft: "3px solid {colors.primary}"
  insight-card-explanation:
    backgroundColor: transparent
    textColor: "{colors.muted-strong}"
    typography: "{typography.body-sm}"
  non-advice-disclaimer:
    backgroundColor: "{colors.surface-elevated-dark}"
    textColor: "{colors.muted-strong}"
    typography: "{typography.caption}"
    rounded: "{rounded.md}"
    padding: 10px 14px
  projection-chart-card:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.xl}"
    padding: 24px
  projection-scenario-legend:
    backgroundColor: transparent
    textColor: "{colors.body}"
    typography: "{typography.caption}"
  chat-bubble-user:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.body-md}"
    rounded: "{rounded.lg}"
    padding: 12px 16px
  chat-bubble-assistant:
    backgroundColor: "{colors.surface-elevated-dark}"
    textColor: "{colors.body}"
    typography: "{typography.body-md}"
    rounded: "{rounded.lg}"
    padding: 12px 16px
  chat-input:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.lg}"
    padding: 12px 16px
    height: 48px
  badge-neutral:
    backgroundColor: "{colors.surface-elevated-dark}"
    textColor: "{colors.body}"
    typography: "{typography.caption}"
    rounded: "{rounded.pill}"
    padding: 4px 10px
  badge-gain:
    backgroundColor: transparent
    textColor: "{colors.gain}"
    typography: "{typography.caption}"
    rounded: "{rounded.pill}"
    padding: 4px 10px
  badge-caution:
    backgroundColor: transparent
    textColor: "{colors.caution}"
    typography: "{typography.caption}"
    rounded: "{rounded.pill}"
    padding: 4px 10px
  text-input-on-light:
    backgroundColor: "{colors.canvas-light}"
    textColor: "{colors.ink}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 10px 14px
    height: 40px
  text-input-on-dark:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 10px 14px
    height: 40px
  risk-profile-segmented:
    backgroundColor: "{colors.surface-strong-light}"
    textColor: "{colors.ink}"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    padding: 4px
  objective-chip:
    backgroundColor: "{colors.surface-elevated-dark}"
    textColor: "{colors.body}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.pill}"
    padding: 8px 14px
  objective-chip-selected:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.pill}"
    padding: 8px 14px
  alert-error:
    backgroundColor: "{colors.surface-card-dark}"
    textColor: "{colors.loss}"
    typography: "{typography.body-sm}"
    rounded: "{rounded.md}"
    padding: 12px 16px
  empty-state:
    backgroundColor: transparent
    textColor: "{colors.muted}"
    typography: "{typography.body-md}"
  loading-skeleton:
    backgroundColor: "{colors.surface-elevated-dark}"
    rounded: "{rounded.md}"
  footer-light:
    backgroundColor: "{colors.surface-soft-light}"
    textColor: "{colors.body-on-light}"
    typography: "{typography.body-sm}"
    padding: 48px
---

## Overview

YieldForge is a **personal investment copilot** for Brazilian retail investors — FIIs and
fixed income, reasoned about *as a whole portfolio*, with plain-language explanations the user
can trust. The two personas (a self-directed dividend investor and a goal-driven long-term
planner) share one trait the design must answer: **they distrust black-box "hot tips."** So the
system reads as **calm, transparent, and modern** — confident but never hype. It is the visual
counterpart of the product's non-advice posture.

The base is a **deep near-black canvas** (`{colors.canvas-dark}` — #0b0e11) for the app surface
and a **clean white** (`{colors.canvas-light}` — #ffffff) for forms and reading-heavy views,
driven by **one token set** with light and dark themes (only canvas, surface, and text tones
flip). A single brand accent — **indigo** (`{colors.primary}` — #4f46e5) — carries every
primary action and brand moment. Crucially, indigo is held apart from the three tokens that
already *mean* something: **gain** green (`{colors.gain}`), **loss** red (`{colors.loss}`), and
**info/focus** cyan (`{colors.info}`). A primary button must never be mistaken for an "up,"
"positive," or "info" signal.

Type runs **Inter** (display + body) and **IBM Plex Mono** (numbers / financial data). Numbers —
balances, yields, percentages, projections — always render in IBM Plex Mono with tabular figures
so money columns align and a value never "jumps" as digits change. The system trusts size and a
single accent over heavy weight; it does not shout.

**Key characteristics:**

- **Single accent:** `{colors.primary}` (indigo) for primary CTAs, active nav, selected chips,
  the user chat bubble, and brand marks. `{colors.primary-tint}` (#818cf8) is its lighter
  sibling for **indigo-as-text on the dark canvas** (links, small accents) where the fill indigo
  would be too low-contrast.
- **Semantic colors are reserved:** `{colors.gain}` / `{colors.loss}` express figure direction
  as **text color only** (never a button or card fill); `{colors.caution}` (amber) flags a risk
  or attention state on badges; `{colors.info}` (cyan) is info + the focus ring. None of these is
  ever repurposed as brand voltage.
- **Two-font split:** Inter for words, IBM Plex Mono for numbers. Mixing them on a figure breaks
  the "trustworthy number" voice.
- **Money is integer, formatted at the edge:** centavos and basis points arrive as integers and
  render as pt-BR strings (`R$ 1.234,56`, `10,50%`) only at the render boundary — no float ever
  represents a balance or rate (ADR-0006, FR-2005).
- **Guards are components, not guidelines:** `{component.insight-card}` cannot render without its
  explanation slot; `{component.non-advice-disclaimer}` is required on every AI surface.
- **No order affordances:** there are deliberately **no Buy/Sell/Long/Short buttons**. A copilot
  never issues a transaction order (FR-014); the UI offers *considerations*, never an order.
- **Flat surfaces, color-block depth:** elevation comes from the step between
  `{colors.canvas-dark}` and `{colors.surface-card-dark}`, not heavy shadows or glassmorphism.
- **Rounded, calm shapes:** medium radii (`{rounded.md}` 8px buttons, `{rounded.xl}` 16px cards)
  read softer and more modern than the tighter Binance scale this system started from.

## Colors

### Brand & Accent
- **Indigo** (`{colors.primary}` — #4f46e5): the single brand color — primary CTAs, active nav
  item, selected states, the user chat bubble, focus emphasis on primary actions. White text
  (`{colors.on-primary}`) sits on it (AA, ~5.9:1).
- **Indigo Active** (`{colors.primary-active}` — #4338ca): the press state, one step darker.
- **Indigo Tint** (`{colors.primary-tint}` — #818cf8): indigo-as-text on the dark canvas — links
  (`{component.text-link-on-dark}`) and small accents where the fill indigo would fall below the
  contrast floor on #0b0e11.
- **Indigo Disabled** (`{colors.primary-disabled}` — #262546): a desaturated indigo for disabled
  primary actions on dark.

### Surface
Two canvas modes from one token set:

**Dark (default app surface):**
- **Canvas Dark** (`{colors.canvas-dark}` — #0b0e11): the app floor. Near-black, slight warmth,
  never pure black.
- **Surface Card Dark** (`{colors.surface-card-dark}` — #1e2329): cards, the sidebar, dropdowns.
- **Surface Elevated Dark** (`{colors.surface-elevated-dark}` — #2b3139): nested cards, the
  assistant chat bubble, hovered items, chart panels.

**Light (forms & reading views):**
- **Canvas Light** (`{colors.canvas-light}` — #ffffff): forms, profile, onboarding.
- **Surface Soft Light** (`{colors.surface-soft-light}` — #fafafa): footer, muted panels.
- **Surface Strong Light** (`{colors.surface-strong-light}` — #f5f5f5): input/segment tracks.

### Hairlines, Borders & Text
- **Hairline on Dark / Light** (`{colors.hairline-on-dark}` #2b3139 / `{colors.hairline-on-light}`
  #eaecef): 1px dividers between rows, table cells, list items. Used liberally — separation comes
  from hairlines and surface steps, not shadow.
- **Border Strong** (`{colors.border-strong}` — #cdd1d6): heavier borders on light disabled controls.
- **Ink** (`{colors.ink}` — #181a20) / **Body on Dark** (`{colors.body}` — #eaecef): primary text
  on light / dark. **Muted** (`{colors.muted}` — #707a8a) and **Muted Strong**
  (`{colors.muted-strong}` — #929aa5) for labels, captions, and explanation copy.

### Semantic (reserved — never brand)
- **Gain** (`{colors.gain}` — #0ecb81): positive figures — growth, yield up — as **text color**.
- **Loss** (`{colors.loss}` — #f6465d): negative figures. Same rule; also the error text tone.
- **Caution** (`{colors.caution}` — #f59e0b): a risk / attention badge (e.g. an over-concentration
  flag on the Health Score). Small, semantic, never a surface fill and never brand voltage.
- **Info / Focus** (`{colors.info}` — #06b6d4): info badges and the keyboard focus ring
  (`{colors.info-ring}`). Cyan keeps it clearly distinct from the indigo primary.

## Typography

### Font Family
**Inter** (display + body) with the system fallback stack, and **IBM Plex Mono** (numbers /
financial data). Both are open-source (SIL Open Font License) — free to embed and ship, in
keeping with the zero-cost posture (ADR-0003). The split is functional:

- Inter → headlines, section titles, body copy, button and nav labels.
- IBM Plex Mono → every number: balances, prices, yields, percentages, projection values, stat
  counters. Always enable **tabular figures** so columns align and values don't reflow as digits
  change — the system's "reliable number" voice.

### Hierarchy

| Token | Size | Weight | Font | Use |
|---|---|---|---|---|
| `{typography.hero-display}` | 56px | 700 | Inter | Onboarding / marketing hero only |
| `{typography.display-lg}` | 44px | 700 | Inter | Big page headline |
| `{typography.display-md}` | 36px | 600 | Inter | Section head |
| `{typography.display-sm}` | 30px | 600 | Inter | Card-band headline |
| `{typography.title-lg}` | 24px | 600 | Inter | Sub-section title |
| `{typography.title-md}` | 20px | 600 | Inter | Card title |
| `{typography.title-sm}` | 16px | 600 | Inter | Row / badge label |
| `{typography.number-display}` | 40px | 600 | IBM Plex Mono | Hero figures — current value, monthly income |
| `{typography.number-md}` | 16px | 500 | IBM Plex Mono | Table figures, holding rows |
| `{typography.number-sm}` | 14px | 500 | IBM Plex Mono | Inline figures, % changes |
| `{typography.body-md}` | 15px | 400 | Inter | Default running text, insight copy |
| `{typography.body-sm}` | 13px | 400 | Inter | Explanations, captions, footer |
| `{typography.caption}` | 12px | 500 | Inter | Meta labels, disclaimer |
| `{typography.button}` | 14px | 600 | Inter | Button labels |
| `{typography.nav-link}` | 14px | 500 | Inter | Nav items |

### Principles
- Every number is IBM Plex Mono with tabular figures — no exceptions, even inline in a sentence
  when it's a portfolio figure.
- Display weight tops out at 700, but the system favours **size and whitespace** over weight; it
  should read calm and legible, not loud.
- Body runs at 15px/1.55 — a touch larger and airier than a trading terminal, because the
  audience is non-professional and reads explanations, not tickers.

## Layout

### Spacing
Base unit **4px**: `{spacing.xxs}` 4 · `{spacing.xs}` 8 · `{spacing.sm}` 12 · `{spacing.md}` 16 ·
`{spacing.lg}` 24 · `{spacing.xl}` 32 · `{spacing.xxl}` 48 · `{spacing.section}` 64. Card padding
is `{spacing.lg}` (24px); grid gutters `{spacing.lg}`; major bands `{spacing.section}` (64px) —
a little airier than the dense source system, matching the calm-modern tone.

### Grid & Container
- **App shell:** a fixed `{component.app-sidebar}` (248px) + fluid content area; max content
  width ~1200px on data views.
- **Dashboard:** a summary band of `{component.metric-callout}` figures, then a responsive grid
  of `{component.content-card-dark}` (allocation donut, sector exposure, health score, insights).
- **Forms (profile, holdings, auth):** single-column, centered, on the light canvas, ~560px wide.

### Whitespace
Calmer than a trading platform: uniform `{spacing.section}` rhythm between bands, generous card
padding, and hairlines rather than dense borders. Let figures and one accent do the work.

## Elevation & Depth

| Level | Treatment | Use |
|---|---|---|
| Flat | No shadow, no border | Page sections, nav, hero bands |
| Hairline | 1px `{colors.hairline-on-dark}` / `{colors.hairline-on-light}` | Inputs, table dividers, list rows |
| Card surface | `{colors.surface-card-dark}` on dark / `{colors.canvas-light}` on light — no shadow | All cards, sidebar |
| Soft shadow | Faint shadow only when a card floats over content (modals, popovers) | Dialogs, the chat composer |
| Focus ring | `0 0 0 2px {colors.info-ring}` | Keyboard focus on inputs and buttons |

Depth is the lightness step between `{colors.canvas-dark}` and `{colors.surface-card-dark}` — no
heavy drop shadows, no glassmorphism, no atmospheric gradients.

## Shapes

| Token | Value | Use |
|---|---|---|
| `{rounded.xs}` | 2px | Tiny inline marks |
| `{rounded.sm}` | 4px | Small badges, skeleton bars |
| `{rounded.md}` | 8px | Buttons, inputs, small controls |
| `{rounded.lg}` | 12px | Insight cards, chat bubbles, inputs |
| `{rounded.xl}` | 16px | Content cards, summary/health/projection cards |
| `{rounded.pill}` | 9999px | Chips, badges, pill CTAs |
| `{rounded.full}` | 9999px | Avatars, donut center |

Radii sit a step softer than the source system (8/12/16 vs 6/8/12) for a calmer, more modern feel.

## Components

### Buttons
- **`button-primary`** — the one primary CTA: indigo fill, white text, `{rounded.md}`, 40px.
  Press → `{component.button-primary-active}`; disabled → `{component.button-primary-disabled}`.
- **`button-primary-pill`** — a pill variant for a single top-of-flow action (e.g. "Começar").
- **`button-secondary-on-dark` / `-on-light`** — lower-emphasis actions (Cancel, secondary nav).
- **`button-tertiary-text`** — inline text button, no background.
- **`text-link-on-dark` / `-on-light`** — inline links; **indigo-tint on dark**, indigo on light.

> **Deliberately absent:** there is no `button-buy`, `button-sell`, or green/red action button.
> The copilot never issues an order (FR-014); gain/loss are figure colors, not actions.

### App Shell
- **`app-sidebar`** + **`app-sidebar-item-active`** — the authenticated left nav (Dashboard,
  Portfolio, Insights, Health, Projections, Chat, Profile). Active item on
  `{colors.surface-elevated-dark}`.
- **`top-nav-dark` / `-light`** — 64px top bar (brand mark, theme toggle, account menu).

### Portfolio & Figures
- **`metric-callout`** + **`metric-callout-label`** — a hero figure (current value / net worth,
  monthly passive income) in `{typography.number-display}` over a small muted label.
- **`value-cell-gain` / `value-cell-loss`** — colored number cells for growth in R$/%. Text color
  only, paired with a small ▲/▼ arrow.
- **`holding-row`** — one FII or fixed-income holding: name/ticker, quantity, cost basis, current
  value, growth cell. Numbers in IBM Plex Mono; hairline divider between rows.
- **`portfolio-summary-card`** — invested vs. current value, monthly income, growth.
- **`allocation-donut-card`** + **`allocation-legend-item`** — allocation by asset class and FII
  sector exposure; slice shares shown as pt-BR percentages.

### Health Score
- **`health-score-gauge`** — the 0–100 score as a radial/arc gauge with a big
  `{typography.number-display}` center; the arc uses a muted-to-gain sweep (not a green/red pass/fail
  binary — it's a health signal, not advice).
- **`health-factor-row`** — one factor (diversification, concentration, liquidity, goal alignment,
  risk exposure) with a small bar and a `{component.badge-caution}` when a factor needs attention.

### AI Surfaces (guarded)
- **`insight-card`** — the core AI unit: a left indigo border, a short insight in
  `{typography.body-md}`, and a **non-optional** `{component.insight-card-explanation}` slot below
  (the "por quê" in muted copy). *An insight without its explanation is unrepresentable* — the
  component contract enforces FR-013, mirroring the backend gate.
- **`non-advice-disclaimer`** — a required, quiet footer on any AI surface (insights, rebalancing,
  health narrative, chat): "Isto é conteúdo educacional, não recomendação de investimento." Its
  presence is a contract of every AI view (FR-014).
- **`chat-bubble-user` / `chat-bubble-assistant`** — the copilot dialogue. User bubble in indigo;
  assistant on `{colors.surface-elevated-dark}`, streaming token-by-token (a typing indicator
  while the SSE stream is open). Every assistant turn ends with the disclaimer.
- **`chat-input`** — the composer; grows to a few lines, primary send action.

### Forms & Inputs
- **`text-input-on-light` / `-on-dark`**, **`risk-profile-segmented`** (Conservador / Moderado /
  Agressivo), **`objective-chip` / `-selected`** (Aposentadoria, Renda Passiva, Preservação,
  Crescimento — multi-select), all with the `{colors.info-ring}` focus ring.

### Feedback & States
- **`badge-neutral` / `-gain` / `-caution`** — status pills. **`alert-error`** — the
  `{"error":"..."}` envelope surfaced from the API, loss-toned. **`empty-state`** — muted guidance
  when a portfolio/insight set is empty (a first-run investor sees this a lot — make it warm, not
  bleak). **`loading-skeleton`** — shimmer blocks while data or a stream loads.

### Footer
- **`footer-light`** — a light closing band (links, non-advice statement, version).

## Do's and Don'ts

### Do
- Reserve `{colors.primary}` (indigo) for primary actions, active/selected states, and brand
  moments. One accent, used with intent.
- Use `{colors.primary-tint}` for indigo **text** on the dark canvas; use `{colors.primary}` for
  indigo **fills** and for indigo text on light.
- Render every number in IBM Plex Mono with tabular figures; format money/rates as pt-BR strings
  at the render edge from integer centavos/bps.
- Always pair an AI insight with its explanation and a `{component.non-advice-disclaimer}`.
- Use `{colors.gain}` / `{colors.loss}` only as figure text color, with a direction arrow for
  accessibility (never color alone).
- Keep the calm rhythm: `{spacing.section}` bands, generous card padding, hairline separation.

### Don't
- Don't add a second brand color, and don't tint the primary toward blue — it must stay clearly
  apart from `{colors.info}` cyan.
- Don't use `{colors.gain}` / `{colors.loss}` / `{colors.caution}` as card or button fills, or as
  brand accents — they are semantic figure/status signals only.
- Don't introduce Buy/Sell/order buttons, price targets, or quantity CTAs — that crosses into
  advice (FR-014).
- Don't render an AI insight, chat reply, or rebalancing suggestion without its explanation and
  the disclaimer.
- Don't put money through a float anywhere on the client, and don't hand-format currency — use the
  formatting helper so R$/% are consistent and locale-correct.
- Don't lean on shadows or gradients for depth; use the surface-step + hairline system.

## Responsive Behavior

| Name | Width | Key changes |
|---|---|---|
| Mobile | < 768px | Sidebar collapses to a bottom tab bar / hamburger; dashboard cards stack 1-up; holding tables reflow to stacked cards (label + value pairs); chat goes full-screen |
| Tablet | 768–1024px | Sidebar as icons-only rail; dashboard 2-up; forms stay single-column centered |
| Desktop | 1024–1440px | Full 248px sidebar; dashboard 2–3-up grid; chat as a right rail or dedicated view |
| Wide | > 1440px | Content caps ~1200px with more outer breathing room |

**Touch targets:** primary controls ≥ 40×40 (44×44 effective with spacing). Holding/insight rows
are fully tappable. **Numbers never wrap** — a large figure shrinks a step rather than breaking
across lines. The theme toggle is available at every breakpoint.

## Iteration Guide

1. Work on ONE component at a time; reference its YAML key (`{component.insight-card}`).
2. Decide a component's theme context (dark app surface vs. light form) first; the same component
   appears in both with only surface/text tones flipped.
3. Variants (`-active`, `-selected`, `-disabled`) are separate `components:` entries, never nested
   state objects.
4. Use `{token.refs}` everywhere prose names a color, radius, type role, or spacing value.
5. Document Default and Active/Pressed (and Selected where relevant) states only — not hover.
6. Numbers → IBM Plex Mono; words → Inter. Never mix them on a figure.
7. `{colors.gain}` / `{colors.loss}` / `{colors.caution}` / `{colors.info}` are semantic — never
   repurpose them as brand or decoration.
8. Any AI-output component must include an explanation slot and require the disclaimer — treat it
   as part of the component's contract, not an optional prop.

## Known Gaps / Open

- **Font loading strategy** (self-host vs. Google Fonts CDN) and the exact tabular-figure feature
  settings are deferred to SPEC-200 implementation.
- **Chart theming** (Recharts palettes for allocation, sector, and the three projection scenarios —
  pessimistic / base / optimistic) needs a dedicated token set; scenarios should read as one hue at
  three intensities, not gain/loss colors.
- **Animation / streaming timings** (chat token stream, skeleton shimmer, figure count-ups) are not
  yet specified.
- **Empty / error / loading** variants exist as base components but need per-screen copy (pt-BR).
- **Accessibility audit** — full WCAG AA contrast sweep across both themes, and colorblind-safe
  gain/loss (arrows + text, not color alone) — to be confirmed during SPEC-200.
- The gain/loss/caution/info hexes are inherited defaults; validate them against the final indigo
  in context during the first screens.
