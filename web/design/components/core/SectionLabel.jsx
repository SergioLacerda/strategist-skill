import React from "react";

/**
 * Centered section label + ornamental rule — the "⟡ ━━━ ⟡" divider
 * that opens every section on the landing page.
 */
export function SectionLabel({ children, rule = true, align = "center", style, ...rest }) {
  return (
    <div style={{ textAlign: align, ...style }} {...rest}>
      <div
        style={{
          fontFamily: "var(--font-display)",
          color: "var(--text-label)",
          letterSpacing: "var(--ls-section)",
          textTransform: "uppercase",
          fontSize: "var(--fs-sm)",
          marginBottom: rule ? "6px" : 0,
        }}
      >
        {children}
      </div>
      {rule && (
        <div style={{ color: "var(--amber-dim)", letterSpacing: "var(--ls-section)", fontSize: "var(--fs-sm)" }}>
          ⟡ ━━━━━━━━━━━━━━━━━━ ⟡
        </div>
      )}
    </div>
  );
}
