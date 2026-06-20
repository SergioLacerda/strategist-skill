Terminal install / invocation panel — one or more labelled command rows in a sunken mono well, with an optional faint source line. Use for setup / CLI snippets ("the invocation spell").

```jsx
<InstallBlock
  commands={[
    { label: "Linux / macOS / WSL", command: "curl -fsSL https://…/bootstrap.sh | bash" },
    { label: "Windows PowerShell", command: "irm https://…/bootstrap.ps1 | iex" },
  ]}
  source="raw.githubusercontent.com/SergioLacerda/strategist-skill/main/"
/>
```

Single row: pass `label` + `command` directly.
