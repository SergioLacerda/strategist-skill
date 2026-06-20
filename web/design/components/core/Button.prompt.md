Uppercase mono action button with amber phosphor styling — use for CTAs and inline actions; `primary` is the filled amber gradient (one per view), `secondary` is the bordered default, `ghost` is borderless.

```jsx
<Button variant="primary" icon="❯" href="#invocation">Invoke the skill</Button>
<Button>View on GitHub ↗</Button>
<Button variant="ghost" disabled>Locked</Button>
```

Variants: `primary` (filled amber, glows on hover), `secondary` (bordered, default), `ghost` (no border). Props: `href` (renders an `<a>`), `icon`, `disabled`, `onClick`.
