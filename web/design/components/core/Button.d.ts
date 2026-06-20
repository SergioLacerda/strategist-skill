import React from "react";

export interface ButtonProps {
  /** Button label / content. */
  children?: React.ReactNode;
  /** Visual weight. @default "secondary" */
  variant?: "primary" | "secondary" | "ghost";
  /** Render as an anchor to this URL instead of a <button>. */
  href?: string;
  /** Leading glyph (e.g. "❯"). */
  icon?: React.ReactNode;
  disabled?: boolean;
  onClick?: (e: React.MouseEvent) => void;
  style?: React.CSSProperties;
}

/**
 * Uppercase mono action button with amber phosphor styling.
 * @startingPoint section="Core" subtitle="Primary / secondary / ghost actions" viewport="700x140"
 */
export function Button(props: ButtonProps): JSX.Element;
