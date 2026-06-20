import React from "react";

/**
 * GatePanel — the sealed approval-gate **portal**: stone jambs flanking an
 * arched doorway closed by an iron portcullis, a keystone, and a glowing
 * central lock, with the requirement headline + description beneath.
 */
export function GatePanel({ lock = "⛨", heading, children, style, ...rest }) {
  const jamb = {
    position: "absolute",
    top: 0,
    bottom: 0,
    width: "46px",
    background:
      "repeating-linear-gradient(0deg, rgba(0,0,0,.55) 0 2px, transparent 2px 28px), repeating-linear-gradient(90deg, #7a3f22 0 6px, #2b1810 6px 15px)",
    boxShadow: "inset 0 0 16px rgba(0,0,0,.7)",
  };

  return (
    <div
      style={{
        position: "relative",
        border: "1px dashed var(--border-gate)",
        borderRadius: "var(--radius-md)",
        background: "linear-gradient(180deg, rgba(207,122,44,.10), rgba(20,15,9,.55))",
        boxShadow: "var(--shadow-gate)",
        padding: "34px 76px 30px",
        textAlign: "center",
        overflow: "hidden",
        ...style,
      }}
      {...rest}
    >
      <div style={{ ...jamb, left: 0 }} aria-hidden="true" />
      <div style={{ ...jamb, right: 0 }} aria-hidden="true" />

      {/* arched doorway closed by a portcullis grille */}
      <div
        style={{
          position: "relative",
          width: "212px",
          height: "228px",
          margin: "4px auto 0",
          border: "2px solid var(--amber-dim)",
          borderBottom: "none",
          borderTopLeftRadius: "106px",
          borderTopRightRadius: "106px",
          background:
            "radial-gradient(120% 90% at 50% 18%, rgba(207,122,44,.12), transparent 60%), repeating-linear-gradient(90deg, rgba(232,194,90,.32) 0 2px, transparent 2px 22px), repeating-linear-gradient(0deg, rgba(232,194,90,.20) 0 2px, transparent 2px 30px), rgba(0,0,0,.5)",
          boxShadow: "inset 0 0 40px rgba(0,0,0,.75), 0 0 22px rgba(207,122,44,.12)",
        }}
        aria-hidden="true"
      >
        {/* keystone */}
        <div
          style={{
            position: "absolute",
            top: "-13px",
            left: "50%",
            transform: "translateX(-50%) rotate(45deg)",
            width: "20px",
            height: "20px",
            background: "linear-gradient(135deg, #cf7a2c, #7a3f22)",
            border: "1px solid var(--amber-dim)",
          }}
        />
        {/* central lock medallion */}
        <div
          style={{
            position: "absolute",
            left: "50%",
            top: "46%",
            transform: "translate(-50%, -50%)",
            width: "74px",
            height: "74px",
            borderRadius: "50%",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            border: "1px solid var(--border-gate)",
            background: "radial-gradient(circle, rgba(207,122,44,.22), rgba(12,8,5,.85) 72%)",
            color: "var(--orange)",
            fontSize: "36px",
            textShadow: "0 0 20px rgba(207,122,44,.7)",
            boxShadow: "0 0 26px rgba(207,122,44,.25)",
          }}
        >
          {lock}
        </div>
      </div>

      {heading && (
        <div
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: "16px",
            fontWeight: 600,
            color: "var(--amber-hi)",
            letterSpacing: ".04em",
            textShadow: "0 0 12px rgba(232,194,90,.4)",
            margin: "22px 0 10px",
          }}
        >
          {heading}
        </div>
      )}
      <p style={{ margin: "0 auto", maxWidth: "560px", fontSize: "var(--fs-body)", color: "var(--muted)", lineHeight: "var(--lh-loose)" }}>
        {children}
      </p>
    </div>
  );
}
