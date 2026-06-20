import React from "react";

/**
 * InstallBlock — terminal install panel. One or more {label, command}
 * rows, each with a faint uppercase header and a mono command line.
 * Pass `commands` (array) or a single `label`+`command`.
 */
export function InstallBlock({ commands, label, command, source, style, ...rest }) {
  const rows = commands || [{ label, command }];
  return (
    <div
      style={{
        border: "1px solid var(--border-hairline)",
        borderRadius: "var(--radius-sm)",
        background: "rgba(0,0,0,.3)",
        overflow: "hidden",
        ...style,
      }}
      {...rest}
    >
      {rows.map((r, i) => (
        <div key={i}>
          {r.label && (
            <div
              style={{
                padding: "8px 16px",
                fontSize: "var(--fs-caption)",
                letterSpacing: ".2em",
                textTransform: "uppercase",
                color: "var(--faint)",
                borderTop: i > 0 ? "1px solid var(--border-hairline)" : "none",
                borderBottom: "1px solid var(--border-hairline)",
              }}
            >
              {r.label}
            </div>
          )}
          <pre
            style={{
              margin: 0,
              padding: "16px 18px",
              fontSize: "var(--fs-sm)",
              fontFamily: "var(--font-mono)",
              color: "var(--text)",
              whiteSpace: "pre-wrap",
              wordBreak: "break-all",
            }}
          >
            {r.command}
          </pre>
        </div>
      ))}
      {source && (
        <p style={{ margin: 0, padding: "10px 18px 12px", fontSize: "11.5px", color: "var(--faint)", letterSpacing: ".04em" }}>
          {source}
        </p>
      )}
    </div>
  );
}
