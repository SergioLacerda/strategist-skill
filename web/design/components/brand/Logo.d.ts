import React from "react";

export interface LogoProps {
  /** "lockup" = emblem + wordmark, "mark" = emblem only, "wordmark" = type only. @default "lockup" */
  variant?: "lockup" | "mark" | "wordmark";
  /** Emblem edge length in px; wordmark scales from it. @default 40 */
  size?: number;
  /** Show the "AUTONOMOUS MISSION SKILL" tagline under the wordmark. @default false */
  tagline?: boolean;
  style?: React.CSSProperties;
}

/**
 * The Strategist brand mark — diamond compass-seal emblem + grimoire wordmark.
 * @startingPoint section="Brand" subtitle="Strategist seal logo — lockup / mark / wordmark" viewport="700x140"
 */
export function Logo(props: LogoProps): JSX.Element;
