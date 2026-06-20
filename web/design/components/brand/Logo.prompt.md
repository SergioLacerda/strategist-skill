The Strategist brand mark — a diamond compass-seal emblem (amber 4-point star inside a runed diamond) beside the grimoire wordmark. Use in app bars, hero units, slide title cards, and favicons (mark only).

```jsx
<Logo />                              {/* full lockup */}
<Logo variant="mark" size={56} />     {/* emblem only — favicons, badges */}
<Logo size={48} tagline />            {/* lockup with "AUTONOMOUS MISSION SKILL" */}
```

Variants: `lockup` (default), `mark`, `wordmark`. `size` drives the emblem; the wordmark scales from it.
