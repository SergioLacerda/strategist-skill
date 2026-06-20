Two-segment status badge (muted label + coloured value) styled like the repo's shields.io chips — use for CI / version / license / mode metadata.

```jsx
<Badge label="CI" value="passing" tone="ok" />
<Badge label="version" value="1.0" tone="amber" />
<Badge label="license" value="CC BY-NC 4.0" tone="orange" />
```

Tones: `ok` (green), `amber` (default), `orange`. Drop `label` for a single-segment chip.
