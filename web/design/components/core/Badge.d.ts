import React from "react";

export interface BadgeProps {
  /** Left segment (muted label). Omit for a single-segment badge. */
  label?: React.ReactNode;
  /** Right segment (the value). */
  value: React.ReactNode;
  /** Colour of the value segment. @default "amber" */
  tone?: "ok" | "amber" | "orange";
  style?: React.CSSProperties;
}

/**
 * Two-segment status badge (label · value), terminal/shields.io style.
 */
export function Badge(props: BadgeProps): JSX.Element;
