import React from "react";

/**
 * RoleCard — a party-member card (Strategist / Ranger / Archivist / Sniper /
 * Wizard). Circular sigil glyph, role name in grimoire face, klass label,
 * and a short description. `gate` styles it as the orange approval gate.
 */
export function RoleCard({ glyph, role, klass, children, gate = false, feature = false, style, ...rest }) {
  const accent = gate ? "var(--orange)" : "var(--amber)";
  return (
    <div
      style={{
        position: "relative",
        border: `1px ${gate ? "dashed" : "solid"} ${gate ? "var(--border-gate)" : "var(--border-strong)"}`,
        borderRadius: "var(--radius-md)",
        background: gate
          ? "linear-gradient(135deg, rgba(207,122,44,.12), transparent 70%)"
          : "var(--grad-panel)",
        boxShadow: "inset 0 0 34px rgba(0,0,0,.4)",
        padding: glyph ? "18px 24px 18px 96px" : "18px 24px",
        gridColumn: feature ? "1 / -1" : "auto",
        ...style,
      }}
      {...rest}
    >
      {glyph && (
        <div
          style={{
            position: "absolute",
            left: "24px",
            top: "22px",
            width: "52px",
            height: "52px",
            border: `1px solid ${gate ? "var(--border-gate)" : "var(--border-strong)"}`,
            borderRadius: "50%",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: "26px",
            color: accent,
            background: "radial-gradient(circle, rgba(232,194,90,.14), transparent 70%)",
            textShadow: gate ? "0 0 14px rgba(207,122,44,.55)" : "0 0 12px rgba(232,194,90,.5)",
          }}
          aria-hidden="true"
        >
          {glyph}
        </div>
      )}
      <div
        style={{
          fontFamily: "var(--font-grimoire)",
          fontWeight: 700,
          fontSize: "23px",
          color: gate ? "var(--orange)" : "var(--amber-hi)",
          letterSpacing: ".03em",
          lineHeight: 1.05,
        }}
      >
        {role}
      </div>
      {klass && (
        <div
          style={{
            fontSize: "var(--fs-caption)",
            letterSpacing: "var(--ls-label)",
            textTransform: "uppercase",
            color: "var(--text-label)",
            margin: "6px 0 10px",
          }}
        >
          {klass}
        </div>
      )}
      <p style={{ margin: 0, fontSize: "13.5px", color: "var(--muted)", lineHeight: "var(--lh-body)" }}>{children}</p>
    </div>
  );
}
