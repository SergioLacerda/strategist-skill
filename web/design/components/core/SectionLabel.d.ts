import React from "react";

export interface SectionLabelProps {
  /** The label text (rendered uppercase, wide-tracked). */
  children: React.ReactNode;
  /** Show the ornamental "⟡ ━━━ ⟡" rule beneath. @default true */
  rule?: boolean;
  /** Text alignment. @default "center" */
  align?: "left" | "center" | "right";
  style?: React.CSSProperties;
}

/**
 * Ornamental section header used to open every region of a Strategist page.
 */
export function SectionLabel(props: SectionLabelProps): JSX.Element;
