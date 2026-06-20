import React from "react";

export interface RoleCardProps {
  /** Circular sigil glyph (e.g. "☉", "⌖", "❒", "✜", "✶"). Omit to drop the medallion. */
  glyph?: React.ReactNode;
  /** Role name (rendered in the grimoire face). */
  role: React.ReactNode;
  /** Klass / sub-role label (uppercase, e.g. "Master · Orchestrator"). */
  klass?: React.ReactNode;
  /** Description copy. */
  children?: React.ReactNode;
  /** Style as the orange approval-gate variant. @default false */
  gate?: boolean;
  /** Span the full grid width (feature card). @default false */
  feature?: boolean;
  style?: React.CSSProperties;
}

/**
 * Party-member / role card with a circular sigil medallion.
 * @startingPoint section="Party" subtitle="Role card with sigil, klass + description" viewport="700x180"
 */
export function RoleCard(props: RoleCardProps): JSX.Element;
