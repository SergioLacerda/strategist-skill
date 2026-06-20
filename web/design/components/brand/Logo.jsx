import React from "react";

/**
 * Logo — the Strategist seal: a diamond compass-seal emblem (4-point amber
 * star inside a runed diamond + ring) beside the grimoire wordmark.
 * Use `mark` for the emblem alone, `wordmark` for type alone, default = lockup.
 */
export function Logo({ variant = "lockup", size = 40, tagline = false, style, ...rest }) {
  const id = React.useId ? React.useId().replace(/:/g, "") : "stg" + Math.random().toString(36).slice(2, 7);

  const Mark = (
    <svg
      width={size}
      height={size}
      viewBox="0 0 100 100"
      role="img"
      aria-label="Strategist seal"
      style={{ filter: "drop-shadow(0 0 10px rgba(232,194,90,.45))", flex: "none", display: "block" }}
    >
      <defs>
        <linearGradient id={"g" + id} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#f8df90" />
          <stop offset="1" stopColor="#b8933f" />
        </linearGradient>
      </defs>
      {/* runed diamond seal */}
      <path d="M50 5 L95 50 L50 95 L5 50 Z" fill="none" stroke="#b8933f" strokeWidth="1.6" />
      <path d="M50 14 L86 50 L50 86 L14 50 Z" fill="none" stroke="rgba(232,194,90,.28)" strokeWidth="1" />
      {/* inner ring */}
      <circle cx="50" cy="50" r="29" fill="none" stroke="rgba(232,194,90,.55)" strokeWidth="1" />
      {/* compass crosshairs */}
      <g stroke="rgba(232,194,90,.22)" strokeWidth="1">
        <line x1="50" y1="5" x2="50" y2="95" />
        <line x1="5" y1="50" x2="95" y2="50" />
      </g>
      {/* 4-point star */}
      <path
        d="M50 17 L57 43 L83 50 L57 57 L50 83 L43 57 L17 50 L43 43 Z"
        fill={"url(#g" + id + ")"}
      />
      {/* diagonal minor rays */}
      <path d="M50 50 L68 32 M50 50 L32 32 M50 50 L68 68 M50 50 L32 68" stroke="rgba(232,194,90,.35)" strokeWidth="1" />
      <circle cx="50" cy="50" r="3.4" fill="#14100a" stroke="#b8933f" strokeWidth="1" />
    </svg>
  );

  if (variant === "mark") {
    return <span style={{ display: "inline-flex", ...style }} {...rest}>{Mark}</span>;
  }

  const Word = (
    <span style={{ display: "flex", flexDirection: "column", justifyContent: "center", lineHeight: 1 }}>
      <span
        style={{
          fontFamily: "var(--font-grimoire)",
          fontWeight: 700,
          fontSize: Math.round(size * 0.52),
          color: "var(--amber-hi)",
          letterSpacing: ".06em",
          textShadow: "0 0 18px rgba(232,194,90,.3)",
        }}
      >
        Strategist
      </span>
      {tagline && (
        <span
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: Math.max(9, Math.round(size * 0.2)),
            letterSpacing: ".34em",
            textTransform: "uppercase",
            color: "var(--orange)",
            marginTop: Math.round(size * 0.12),
          }}
        >
          autonomous mission skill
        </span>
      )}
    </span>
  );

  if (variant === "wordmark") {
    return <span style={{ display: "inline-flex", ...style }} {...rest}>{Word}</span>;
  }

  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: Math.round(size * 0.32), ...style }} {...rest}>
      {Mark}
      {Word}
    </span>
  );
}
