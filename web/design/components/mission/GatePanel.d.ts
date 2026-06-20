import React from "react";

export interface GatePanelProps {
  /** Lock glyph in the center. @default "⛨" */
  lock?: React.ReactNode;
  /** Requirement headline (e.g. "REQUIRES :: human confirmation"). */
  heading?: React.ReactNode;
  /** Supporting description. */
  children?: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * The inviolable approval-gate portal — iron portcullis walls around a sealed lock.
 * @startingPoint section="Mission" subtitle="The sealed human-approval gate portal" viewport="900x220"
 */
export function GatePanel(props: GatePanelProps): JSX.Element;
