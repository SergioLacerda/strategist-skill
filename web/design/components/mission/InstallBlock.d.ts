import React from "react";

export interface InstallCommand {
  /** Uppercase header above the command (e.g. "Linux / macOS / WSL"). */
  label?: React.ReactNode;
  /** The command line text. */
  command: React.ReactNode;
}

export interface InstallBlockProps {
  /** Multiple labelled command rows. */
  commands?: InstallCommand[];
  /** Single-row shorthand — header. */
  label?: React.ReactNode;
  /** Single-row shorthand — command. */
  command?: React.ReactNode;
  /** Faint source line under the block. */
  source?: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * Terminal install / invocation panel with labelled command rows.
 */
export function InstallBlock(props: InstallBlockProps): JSX.Element;
