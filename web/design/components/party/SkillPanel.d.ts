import React from "react";

export interface SkillPanelProps {
  /** Circular medallion glyph (e.g. "⚲", "⚑", "❖", "⚡"). */
  icon?: React.ReactNode;
  /** Small uppercase eyebrow above the title. */
  eyebrow?: React.ReactNode;
  /** Panel title (grimoire face). */
  title: React.ReactNode;
  /** Description copy. */
  children?: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * Dashed mini panel for a special ability / skill (icon + title + description).
 */
export function SkillPanel(props: SkillPanelProps): JSX.Element;
