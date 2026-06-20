import React from "react";

/**
 * Split label/value badge — the CI / version / license chips.
 */
export function Badge({ label, value, tone = "amber", style, ...rest }) {
  const tones = {
    ok:     { color: "var(--green)",  bg: "rgba(116,207,142,.08)" },
    amber:  { color: "var(--amber)",  bg: "rgba(232,194,90,.08)" },
    orange: { color: "var(--orange)", bg: "rgba(207,122,44,.10)" },
  };
  const t = tones[tone] || tones.amber;
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "stretch",
        fontFamily: "var(--font-mono)",
        fontSize: "var(--fs-label)",
        letterSpacing: ".04em",
        border: "1px solid var(--border-strong)",
        borderRadius: "var(--radius-xs)",
        overflow: "hidden",
        ...style,
      }}
      {...rest}
    >
      {label != null && (
        <b style={{ background: "rgba(232,194,90,.08)", color: "var(--muted)", fontWeight: 500, padding: "4px 9px" }}>
          {label}
        </b>
      )}
      <i style={{ fontStyle: "normal", padding: "4px 9px", fontWeight: 600, color: t.color, background: t.bg }}>
        {value}
      </i>
    </span>
  );
}
