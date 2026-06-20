import React from "react";

/**
 * SkillPanel — a dashed mini panel for a special ability / skill
 * (Opportunist Attack, Side Quest, Treasure Chest, Quick Draw).
 * Circular icon medallion above an eyebrow, title, and description.
 */
export function SkillPanel({ icon, eyebrow, title, children, style, ...rest }) {
  return (
    <div
      style={{
        position: "relative",
        border: "1px dashed var(--border-strong)",
        borderRadius: "var(--radius-md)",
        background: "linear-gradient(180deg, rgba(42,33,23,.42), rgba(20,15,9,.34))",
        boxShadow: "inset 0 0 30px rgba(0,0,0,.38)",
        padding: "26px 24px 24px",
        ...style,
      }}
      {...rest}
    >
      {icon && (
        <div
          style={{
            width: "50px",
            height: "50px",
            border: "1px solid var(--border-strong)",
            borderRadius: "50%",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: "24px",
            color: "var(--amber)",
            background: "radial-gradient(circle, rgba(232,194,90,.13), transparent 70%)",
            textShadow: "0 0 12px rgba(232,194,90,.45)",
            marginBottom: "16px",
          }}
          aria-hidden="true"
        >
          {icon}
        </div>
      )}
      {eyebrow && (
        <div
          style={{
            fontSize: "var(--fs-caption)",
            letterSpacing: ".2em",
            textTransform: "uppercase",
            color: "var(--text-label)",
            marginBottom: "5px",
          }}
        >
          {eyebrow}
        </div>
      )}
      <h3
        style={{
          fontFamily: "var(--font-grimoire)",
          fontWeight: 700,
          fontSize: "20px",
          color: "var(--amber-hi)",
          margin: "0 0 8px",
          letterSpacing: ".02em",
          lineHeight: 1.12,
        }}
      >
        {title}
      </h3>
      <p style={{ margin: 0, fontSize: "var(--fs-sm)", color: "var(--muted)", lineHeight: "var(--lh-body)" }}>{children}</p>
    </div>
  );
}
