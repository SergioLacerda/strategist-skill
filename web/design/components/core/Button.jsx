import React from "react";

/**
 * Strategist primary/secondary/ghost button — uppercase mono, amber phosphor.
 */
export function Button({
  children,
  variant = "secondary", // "primary" | "secondary" | "ghost"
  href,
  icon,
  disabled = false,
  onClick,
  style,
  ...rest
}) {
  const base = {
    display: "inline-flex",
    alignItems: "center",
    gap: "9px",
    fontFamily: "var(--font-mono)",
    fontSize: "var(--fs-sm)",
    letterSpacing: "var(--ls-wide)",
    textTransform: "uppercase",
    fontWeight: "var(--fw-medium)",
    padding: "12px 22px",
    borderRadius: "var(--radius-sm)",
    textDecoration: "none",
    cursor: disabled ? "not-allowed" : "pointer",
    border: "1px solid var(--border-strong)",
    color: "var(--accent)",
    background: "rgba(232,194,90,.05)",
    opacity: disabled ? 0.45 : 1,
    transition: "background var(--dur), border-color var(--dur), transform var(--dur-fast), box-shadow var(--dur)",
    ...(variant === "ghost" ? { border: "1px solid transparent", background: "transparent" } : null),
    ...(variant === "primary"
      ? {
          color: "#1c1408",
          borderColor: "var(--accent)",
          background: "var(--grad-amber-btn)",
          textShadow: "none",
          boxShadow: "var(--shadow-btn-amber)",
        }
      : null),
    ...style,
  };

  const content = (
    <>
      {icon ? <span aria-hidden="true">{icon}</span> : null}
      {children}
    </>
  );

  if (href && !disabled) {
    return (
      <a href={href} style={base} onClick={onClick} {...rest}>
        {content}
      </a>
    );
  }
  return (
    <button type="button" style={base} disabled={disabled} onClick={onClick} {...rest}>
      {content}
    </button>
  );
}
