/* @ds-bundle: {"format":3,"namespace":"StrategistDesignSystem_88a8a0","components":[{"name":"Logo","sourcePath":"components/brand/Logo.jsx"},{"name":"Badge","sourcePath":"components/core/Badge.jsx"},{"name":"Button","sourcePath":"components/core/Button.jsx"},{"name":"SectionLabel","sourcePath":"components/core/SectionLabel.jsx"},{"name":"GatePanel","sourcePath":"components/mission/GatePanel.jsx"},{"name":"InstallBlock","sourcePath":"components/mission/InstallBlock.jsx"},{"name":"RoleCard","sourcePath":"components/party/RoleCard.jsx"},{"name":"SkillPanel","sourcePath":"components/party/SkillPanel.jsx"}],"sourceHashes":{"components/brand/Logo.jsx":"ac856eb9011c","components/core/Badge.jsx":"dcdc9c7588fd","components/core/Button.jsx":"1ab50e1b85f8","components/core/SectionLabel.jsx":"39229322b6d7","components/mission/GatePanel.jsx":"aabdffa42819","components/mission/InstallBlock.jsx":"b2cad37cf499","components/party/RoleCard.jsx":"a1ad6a38550f","components/party/SkillPanel.jsx":"353e9448d063","ui_kits/console/i18n.js":"94168ff71f01","ui_kits/landing/Hero.jsx":"df7a44c7615d","ui_kits/landing/PipelineMap.jsx":"9bd8c905d483","ui_kits/landing/i18n.js":"d7cdf9b050e7"},"inlinedExternals":[],"unexposedExports":[]} */

(() => {

const __ds_ns = (window.StrategistDesignSystem_88a8a0 = window.StrategistDesignSystem_88a8a0 || {});

const __ds_scope = {};

(__ds_ns.__errors = __ds_ns.__errors || []);

// components/brand/Logo.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Logo — the Strategist seal: a diamond compass-seal emblem (4-point amber
 * star inside a runed diamond + ring) beside the grimoire wordmark.
 * Use `mark` for the emblem alone, `wordmark` for type alone, default = lockup.
 */
function Logo({
  variant = "lockup",
  size = 40,
  tagline = false,
  style,
  ...rest
}) {
  const id = React.useId ? React.useId().replace(/:/g, "") : "stg" + Math.random().toString(36).slice(2, 7);
  const Mark = /*#__PURE__*/React.createElement("svg", {
    width: size,
    height: size,
    viewBox: "0 0 100 100",
    role: "img",
    "aria-label": "Strategist seal",
    style: {
      filter: "drop-shadow(0 0 10px rgba(232,194,90,.45))",
      flex: "none",
      display: "block"
    }
  }, /*#__PURE__*/React.createElement("defs", null, /*#__PURE__*/React.createElement("linearGradient", {
    id: "g" + id,
    x1: "0",
    y1: "0",
    x2: "0",
    y2: "1"
  }, /*#__PURE__*/React.createElement("stop", {
    offset: "0",
    stopColor: "#f8df90"
  }), /*#__PURE__*/React.createElement("stop", {
    offset: "1",
    stopColor: "#b8933f"
  }))), /*#__PURE__*/React.createElement("path", {
    d: "M50 5 L95 50 L50 95 L5 50 Z",
    fill: "none",
    stroke: "#b8933f",
    strokeWidth: "1.6"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M50 14 L86 50 L50 86 L14 50 Z",
    fill: "none",
    stroke: "rgba(232,194,90,.28)",
    strokeWidth: "1"
  }), /*#__PURE__*/React.createElement("circle", {
    cx: "50",
    cy: "50",
    r: "29",
    fill: "none",
    stroke: "rgba(232,194,90,.55)",
    strokeWidth: "1"
  }), /*#__PURE__*/React.createElement("g", {
    stroke: "rgba(232,194,90,.22)",
    strokeWidth: "1"
  }, /*#__PURE__*/React.createElement("line", {
    x1: "50",
    y1: "5",
    x2: "50",
    y2: "95"
  }), /*#__PURE__*/React.createElement("line", {
    x1: "5",
    y1: "50",
    x2: "95",
    y2: "50"
  })), /*#__PURE__*/React.createElement("path", {
    d: "M50 17 L57 43 L83 50 L57 57 L50 83 L43 57 L17 50 L43 43 Z",
    fill: "url(#g" + id + ")"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M50 50 L68 32 M50 50 L32 32 M50 50 L68 68 M50 50 L32 68",
    stroke: "rgba(232,194,90,.35)",
    strokeWidth: "1"
  }), /*#__PURE__*/React.createElement("circle", {
    cx: "50",
    cy: "50",
    r: "3.4",
    fill: "#14100a",
    stroke: "#b8933f",
    strokeWidth: "1"
  }));
  if (variant === "mark") {
    return /*#__PURE__*/React.createElement("span", _extends({
      style: {
        display: "inline-flex",
        ...style
      }
    }, rest), Mark);
  }
  const Word = /*#__PURE__*/React.createElement("span", {
    style: {
      display: "flex",
      flexDirection: "column",
      justifyContent: "center",
      lineHeight: 1
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-grimoire)",
      fontWeight: 700,
      fontSize: Math.round(size * 0.52),
      color: "var(--amber-hi)",
      letterSpacing: ".06em",
      textShadow: "0 0 18px rgba(232,194,90,.3)"
    }
  }, "Strategist"), tagline && /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: Math.max(9, Math.round(size * 0.2)),
      letterSpacing: ".34em",
      textTransform: "uppercase",
      color: "var(--orange)",
      marginTop: Math.round(size * 0.12)
    }
  }, "autonomous mission skill"));
  if (variant === "wordmark") {
    return /*#__PURE__*/React.createElement("span", _extends({
      style: {
        display: "inline-flex",
        ...style
      }
    }, rest), Word);
  }
  return /*#__PURE__*/React.createElement("span", _extends({
    style: {
      display: "inline-flex",
      alignItems: "center",
      gap: Math.round(size * 0.32),
      ...style
    }
  }, rest), Mark, Word);
}
Object.assign(__ds_scope, { Logo });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/brand/Logo.jsx", error: String((e && e.message) || e) }); }

// components/core/Badge.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Split label/value badge — the CI / version / license chips.
 */
function Badge({
  label,
  value,
  tone = "amber",
  style,
  ...rest
}) {
  const tones = {
    ok: {
      color: "var(--green)",
      bg: "rgba(116,207,142,.08)"
    },
    amber: {
      color: "var(--amber)",
      bg: "rgba(232,194,90,.08)"
    },
    orange: {
      color: "var(--orange)",
      bg: "rgba(207,122,44,.10)"
    }
  };
  const t = tones[tone] || tones.amber;
  return /*#__PURE__*/React.createElement("span", _extends({
    style: {
      display: "inline-flex",
      alignItems: "stretch",
      fontFamily: "var(--font-mono)",
      fontSize: "var(--fs-label)",
      letterSpacing: ".04em",
      border: "1px solid var(--border-strong)",
      borderRadius: "var(--radius-xs)",
      overflow: "hidden",
      ...style
    }
  }, rest), label != null && /*#__PURE__*/React.createElement("b", {
    style: {
      background: "rgba(232,194,90,.08)",
      color: "var(--muted)",
      fontWeight: 500,
      padding: "4px 9px"
    }
  }, label), /*#__PURE__*/React.createElement("i", {
    style: {
      fontStyle: "normal",
      padding: "4px 9px",
      fontWeight: 600,
      color: t.color,
      background: t.bg
    }
  }, value));
}
Object.assign(__ds_scope, { Badge });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/Badge.jsx", error: String((e && e.message) || e) }); }

// components/core/Button.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Strategist primary/secondary/ghost button — uppercase mono, amber phosphor.
 */
function Button({
  children,
  variant = "secondary",
  // "primary" | "secondary" | "ghost"
  href,
  icon,
  disabled = false,
  onClick,
  style,
  ...rest
}) {
  const base = {
    display: "inline-flex",
    alignItems: "center",
    gap: "9px",
    fontFamily: "var(--font-mono)",
    fontSize: "var(--fs-sm)",
    letterSpacing: "var(--ls-wide)",
    textTransform: "uppercase",
    fontWeight: "var(--fw-medium)",
    padding: "12px 22px",
    borderRadius: "var(--radius-sm)",
    textDecoration: "none",
    cursor: disabled ? "not-allowed" : "pointer",
    border: "1px solid var(--border-strong)",
    color: "var(--accent)",
    background: "rgba(232,194,90,.05)",
    opacity: disabled ? 0.45 : 1,
    transition: "background var(--dur), border-color var(--dur), transform var(--dur-fast), box-shadow var(--dur)",
    ...(variant === "ghost" ? {
      border: "1px solid transparent",
      background: "transparent"
    } : null),
    ...(variant === "primary" ? {
      color: "#1c1408",
      borderColor: "var(--accent)",
      background: "var(--grad-amber-btn)",
      textShadow: "none",
      boxShadow: "var(--shadow-btn-amber)"
    } : null),
    ...style
  };
  const content = /*#__PURE__*/React.createElement(React.Fragment, null, icon ? /*#__PURE__*/React.createElement("span", {
    "aria-hidden": "true"
  }, icon) : null, children);
  if (href && !disabled) {
    return /*#__PURE__*/React.createElement("a", _extends({
      href: href,
      style: base,
      onClick: onClick
    }, rest), content);
  }
  return /*#__PURE__*/React.createElement("button", _extends({
    type: "button",
    style: base,
    disabled: disabled,
    onClick: onClick
  }, rest), content);
}
Object.assign(__ds_scope, { Button });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/Button.jsx", error: String((e && e.message) || e) }); }

// components/core/SectionLabel.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Centered section label + ornamental rule — the "⟡ ━━━ ⟡" divider
 * that opens every section on the landing page.
 */
function SectionLabel({
  children,
  rule = true,
  align = "center",
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      textAlign: align,
      ...style
    }
  }, rest), /*#__PURE__*/React.createElement("div", {
    style: {
      fontFamily: "var(--font-display)",
      color: "var(--text-label)",
      letterSpacing: "var(--ls-section)",
      textTransform: "uppercase",
      fontSize: "var(--fs-sm)",
      marginBottom: rule ? "6px" : 0
    }
  }, children), rule && /*#__PURE__*/React.createElement("div", {
    style: {
      color: "var(--amber-dim)",
      letterSpacing: "var(--ls-section)",
      fontSize: "var(--fs-sm)"
    }
  }, "\u27E1 \u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501\u2501 \u27E1"));
}
Object.assign(__ds_scope, { SectionLabel });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/core/SectionLabel.jsx", error: String((e && e.message) || e) }); }

// components/mission/GatePanel.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * GatePanel — the sealed approval-gate **portal**: stone jambs flanking an
 * arched doorway closed by an iron portcullis, a keystone, and a glowing
 * central lock, with the requirement headline + description beneath.
 */
function GatePanel({
  lock = "⛨",
  heading,
  children,
  style,
  ...rest
}) {
  const jamb = {
    position: "absolute",
    top: 0,
    bottom: 0,
    width: "46px",
    background: "repeating-linear-gradient(0deg, rgba(0,0,0,.55) 0 2px, transparent 2px 28px), repeating-linear-gradient(90deg, #7a3f22 0 6px, #2b1810 6px 15px)",
    boxShadow: "inset 0 0 16px rgba(0,0,0,.7)"
  };
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      position: "relative",
      border: "1px dashed var(--border-gate)",
      borderRadius: "var(--radius-md)",
      background: "linear-gradient(180deg, rgba(207,122,44,.10), rgba(20,15,9,.55))",
      boxShadow: "var(--shadow-gate)",
      padding: "34px 76px 30px",
      textAlign: "center",
      overflow: "hidden",
      ...style
    }
  }, rest), /*#__PURE__*/React.createElement("div", {
    style: {
      ...jamb,
      left: 0
    },
    "aria-hidden": "true"
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      ...jamb,
      right: 0
    },
    "aria-hidden": "true"
  }), /*#__PURE__*/React.createElement("div", {
    style: {
      position: "relative",
      width: "212px",
      height: "228px",
      margin: "4px auto 0",
      border: "2px solid var(--amber-dim)",
      borderBottom: "none",
      borderTopLeftRadius: "106px",
      borderTopRightRadius: "106px",
      background: "radial-gradient(120% 90% at 50% 18%, rgba(207,122,44,.12), transparent 60%), repeating-linear-gradient(90deg, rgba(232,194,90,.32) 0 2px, transparent 2px 22px), repeating-linear-gradient(0deg, rgba(232,194,90,.20) 0 2px, transparent 2px 30px), rgba(0,0,0,.5)",
      boxShadow: "inset 0 0 40px rgba(0,0,0,.75), 0 0 22px rgba(207,122,44,.12)"
    },
    "aria-hidden": "true"
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      position: "absolute",
      top: "-13px",
      left: "50%",
      transform: "translateX(-50%) rotate(45deg)",
      width: "20px",
      height: "20px",
      background: "linear-gradient(135deg, #cf7a2c, #7a3f22)",
      border: "1px solid var(--amber-dim)"
    }
  }), /*#__PURE__*/React.createElement("div", {
    style: {
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
      boxShadow: "0 0 26px rgba(207,122,44,.25)"
    }
  }, lock)), heading && /*#__PURE__*/React.createElement("div", {
    style: {
      fontFamily: "var(--font-mono)",
      fontSize: "16px",
      fontWeight: 600,
      color: "var(--amber-hi)",
      letterSpacing: ".04em",
      textShadow: "0 0 12px rgba(232,194,90,.4)",
      margin: "22px 0 10px"
    }
  }, heading), /*#__PURE__*/React.createElement("p", {
    style: {
      margin: "0 auto",
      maxWidth: "560px",
      fontSize: "var(--fs-body)",
      color: "var(--muted)",
      lineHeight: "var(--lh-loose)"
    }
  }, children));
}
Object.assign(__ds_scope, { GatePanel });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/mission/GatePanel.jsx", error: String((e && e.message) || e) }); }

// components/mission/InstallBlock.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * InstallBlock — terminal install panel. One or more {label, command}
 * rows, each with a faint uppercase header and a mono command line.
 * Pass `commands` (array) or a single `label`+`command`.
 */
function InstallBlock({
  commands,
  label,
  command,
  source,
  style,
  ...rest
}) {
  const rows = commands || [{
    label,
    command
  }];
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      border: "1px solid var(--border-hairline)",
      borderRadius: "var(--radius-sm)",
      background: "rgba(0,0,0,.3)",
      overflow: "hidden",
      ...style
    }
  }, rest), rows.map((r, i) => /*#__PURE__*/React.createElement("div", {
    key: i
  }, r.label && /*#__PURE__*/React.createElement("div", {
    style: {
      padding: "8px 16px",
      fontSize: "var(--fs-caption)",
      letterSpacing: ".2em",
      textTransform: "uppercase",
      color: "var(--faint)",
      borderTop: i > 0 ? "1px solid var(--border-hairline)" : "none",
      borderBottom: "1px solid var(--border-hairline)"
    }
  }, r.label), /*#__PURE__*/React.createElement("pre", {
    style: {
      margin: 0,
      padding: "16px 18px",
      fontSize: "var(--fs-sm)",
      fontFamily: "var(--font-mono)",
      color: "var(--text)",
      whiteSpace: "pre-wrap",
      wordBreak: "break-all"
    }
  }, r.command))), source && /*#__PURE__*/React.createElement("p", {
    style: {
      margin: 0,
      padding: "10px 18px 12px",
      fontSize: "11.5px",
      color: "var(--faint)",
      letterSpacing: ".04em"
    }
  }, source));
}
Object.assign(__ds_scope, { InstallBlock });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/mission/InstallBlock.jsx", error: String((e && e.message) || e) }); }

// components/party/RoleCard.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * RoleCard — a party-member card (Strategist / Ranger / Archivist / Sniper /
 * Wizard). Circular sigil glyph, role name in grimoire face, klass label,
 * and a short description. `gate` styles it as the orange approval gate.
 */
function RoleCard({
  glyph,
  role,
  klass,
  children,
  gate = false,
  feature = false,
  style,
  ...rest
}) {
  const accent = gate ? "var(--orange)" : "var(--amber)";
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      position: "relative",
      border: `1px ${gate ? "dashed" : "solid"} ${gate ? "var(--border-gate)" : "var(--border-strong)"}`,
      borderRadius: "var(--radius-md)",
      background: gate ? "linear-gradient(135deg, rgba(207,122,44,.12), transparent 70%)" : "var(--grad-panel)",
      boxShadow: "inset 0 0 34px rgba(0,0,0,.4)",
      padding: glyph ? "18px 24px 18px 96px" : "18px 24px",
      gridColumn: feature ? "1 / -1" : "auto",
      ...style
    }
  }, rest), glyph && /*#__PURE__*/React.createElement("div", {
    style: {
      position: "absolute",
      left: "24px",
      top: "22px",
      width: "52px",
      height: "52px",
      border: `1px solid ${gate ? "var(--border-gate)" : "var(--border-strong)"}`,
      borderRadius: "50%",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      fontSize: "26px",
      color: accent,
      background: "radial-gradient(circle, rgba(232,194,90,.14), transparent 70%)",
      textShadow: gate ? "0 0 14px rgba(207,122,44,.55)" : "0 0 12px rgba(232,194,90,.5)"
    },
    "aria-hidden": "true"
  }, glyph), /*#__PURE__*/React.createElement("div", {
    style: {
      fontFamily: "var(--font-grimoire)",
      fontWeight: 700,
      fontSize: "23px",
      color: gate ? "var(--orange)" : "var(--amber-hi)",
      letterSpacing: ".03em",
      lineHeight: 1.05
    }
  }, role), klass && /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "var(--fs-caption)",
      letterSpacing: "var(--ls-label)",
      textTransform: "uppercase",
      color: "var(--text-label)",
      margin: "6px 0 10px"
    }
  }, klass), /*#__PURE__*/React.createElement("p", {
    style: {
      margin: 0,
      fontSize: "13.5px",
      color: "var(--muted)",
      lineHeight: "var(--lh-body)"
    }
  }, children));
}
Object.assign(__ds_scope, { RoleCard });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/party/RoleCard.jsx", error: String((e && e.message) || e) }); }

// components/party/SkillPanel.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * SkillPanel — a dashed mini panel for a special ability / skill
 * (Opportunist Attack, Side Quest, Treasure Chest).
 * Circular icon medallion above an eyebrow, title, and description.
 */
function SkillPanel({
  icon,
  eyebrow,
  title,
  children,
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      position: "relative",
      border: "1px dashed var(--border-strong)",
      borderRadius: "var(--radius-md)",
      background: "linear-gradient(180deg, rgba(42,33,23,.42), rgba(20,15,9,.34))",
      boxShadow: "inset 0 0 30px rgba(0,0,0,.38)",
      padding: "26px 24px 24px",
      ...style
    }
  }, rest), icon && /*#__PURE__*/React.createElement("div", {
    style: {
      width: "50px",
      height: "50px",
      border: "1px solid var(--border-strong)",
      borderRadius: "50%",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      fontSize: "24px",
      color: "var(--amber)",
      background: "radial-gradient(circle, rgba(232,194,90,.13), transparent 70%)",
      textShadow: "0 0 12px rgba(232,194,90,.45)",
      marginBottom: "16px"
    },
    "aria-hidden": "true"
  }, icon), eyebrow && /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: "var(--fs-caption)",
      letterSpacing: ".2em",
      textTransform: "uppercase",
      color: "var(--text-label)",
      marginBottom: "5px"
    }
  }, eyebrow), /*#__PURE__*/React.createElement("h3", {
    style: {
      fontFamily: "var(--font-grimoire)",
      fontWeight: 700,
      fontSize: "20px",
      color: "var(--amber-hi)",
      margin: "0 0 8px",
      letterSpacing: ".02em",
      lineHeight: 1.12
    }
  }, title), /*#__PURE__*/React.createElement("p", {
    style: {
      margin: 0,
      fontSize: "var(--fs-sm)",
      color: "var(--muted)",
      lineHeight: "var(--lh-body)"
    }
  }, children));
}
Object.assign(__ds_scope, { SkillPanel });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/party/SkillPanel.jsx", error: String((e && e.message) || e) }); }

// ui_kits/console/i18n.js
try { (() => {
/* Strategist Console (v2) — bilingual copy (PT-BR primary / EN).
   Exposed as window.CONSOLE_I18N. */
window.CONSOLE_I18N = {
  pt: {
    nav: {
      overview: "Visão Geral",
      roles: "Papéis",
      skills: "Habilidades",
      mission: "Fluxo da Missão",
      invoke: "Invocação"
    },
    tagline: "A experiência com suas demandas nunca será a mesma.",
    lede: "Transforme demandas ambíguas em <b>missões governadas</b> para agentes IA. O Strategist orquestra discovery, refinamento, approval gate e execução controlada através de papéis plugáveis — reduzindo improviso, drift e risco em workflows com IA.",
    ctaPrimary: "Instalar Strategist",
    ctaSecondary: "Ver arquitetura",
    ghBtn: "Ver no GitHub ↗",
    heroBadges: ["Workflow de IA Governado", "Gate de Aprovação Humana", "Pronto para Plugins", "Compatível com SDD"],
    ovEyebrow: "console do grupo",
    ovTiles: [{
      tab: "roles",
      glyph: "☉",
      title: "Customize os papéis",
      desc: "Defina o grupo que conduz cada missão."
    }, {
      tab: "roles",
      glyph: "‡",
      title: "Especialize papéis com múltiplas skills",
      desc: "Rankeie classes e arme-as com sub-skills."
    }, {
      tab: "mission",
      glyph: "⛨",
      title: "Integração com modelos de governança",
      desc: "Approval gate e políticas dentro do fluxo."
    }, {
      tab: "mission",
      glyph: "❒",
      title: "Documentação gerada conforme seus padrões",
      desc: "Determine o padrão e o agente cumpre."
    }],
    rolesEyebrow: "a grupo",
    rolesTitle: "Papéis",
    rolesDesc: "Cada Skill(IA agent) atua em um papel(persona), sob orientação do estrategista, que define responsabilidade e limites de atuação.",
    roles: [{
      glyph: "🧠",
      role: "Estrategista",
      klass: "Mestre · Orquestrador",
      desc: "Orquestra a missão de ponta a ponta. Delegando e monitorando os papéis, reunindo o time e documentação para aguardar usuário no gate de aprovação.",
      feature: true
    }, {
      glyph: "⌕",
      role: "Ranger",
      klass: "Batedor · Discovery",
      desc: "Levanta requisitos, mapeia o contexto e devolve um relatório de discovery."
    }, {
      glyph: "✎",
      role: "Archivist",
      klass: "Arquivista · Refinamento",
      desc: "Aprofunda o relatório do Ranger em especificação acionável e análise detalhada."
    }, {
      glyph: "⌖",
      role: "Sniper",
      klass: "Executor · Implementação",
      desc: "<i>Somente após aprovação humana explícita</i>, executa a implementação refinada quando o escopo inclui entrega."
    }, {
      glyph: "✶",
      role: "Wizard",
      klass: "Invocador · Instalador",
      desc: "Conjura o grupo no repositório-alvo via <i>curl</i> / <i>irm</i>. Auxilia o usuário para consolidar o padrão desejado."
    }],
    mechEyebrow: "mecânica",
    mechTitle: "Rank & Armas",
    mechDesc: "Como o contrato define o papel, o provider rankeado e as armas de cada slot.",
    mech: [{
      glyph: "❖",
      title: "Rank / Especialização",
      desc: "Quando o provider declara <b>canonical_role</b> e <b>provider_class: rankeado</b>, ele se torna <b>rankeado</b> — uma especialização reconhecida para aquele papel, sem ganhar permissões extras."
    }, {
      glyph: "⚔",
      title: "Armas",
      desc: "As <b>armas</b> são os providers concretos configurados em cada <i>slot</i>. O papel é o guerreiro; a arma é a skill que ele empunha para executar discovery, refinement ou execution."
    }],
    examplesEyebrow: "exemplos",
    examplesTitle: "Skills rankeadas",
    rankLabel: "rank",
    skillLabel: "skill",
    weaponsLabel: "armas",
    examples: [{
      glyph: "⌕",
      role: "Ranger",
      skill: "brainstorming",
      weapons: ["writing-skills"]
    }, {
      glyph: "✎",
      role: "Archivist",
      skill: "openspec-explore",
      weapons: ["openspec-propose", "openspec-apply-change", "openspec-archive-change"]
    }],
    skillsEyebrow: "ações especiais",
    skillsTitle: "Habilidades",
    skillsDesc: "Capacidades especiais que o grupo pode acionar durante a missão.",
    skills: [{
      icon: "⚔",
      title: "Ataque de oportunidade",
      desc: "Em ações podemos encontrar problemas não previstos ou <i>baús de tesouros</i>."
    }, {
      icon: "⚑",
      title: "Missão opcional",
      desc: "Missões pequenas — mover tarefas prontas para a pasta correta ou hotfixes pontuais."
    }, {
      icon: "❖",
      title: "Baú de tesouro",
      desc: "Em cada estágio podemos encontrar fontes de documentação para enriquecer a análise."
    }, {
      icon: "⧗",
      title: "Iniciativa",
      desc: "Avalie seu time antes de começar suas missões."
    }, {
      icon: "⛩",
      title: "Dojo",
      desc: "Ambiente controlado para testar suas habilidades."
    }],
    missionEyebrow: "pipeline",
    missionTitle: "Fluxo da Missão",
    missionDesc: "Discovery e refinamento rodam autonomamente; a execução só avança após o portão de aprovação.",
    phaseLabel: "fase",
    phases: [{
      glyph: "⌕",
      n: "01",
      name: "Discovery",
      who: "Ranger",
      desc: "Levanta requisitos e mapeia o contexto."
    }, {
      glyph: "✎",
      n: "02",
      name: "Refinamento",
      who: "Archivist",
      desc: "Transforma o relatório em especificação acionável."
    }, {
      glyph: "⛨",
      n: "03",
      name: "Approval Gate",
      who: "Humano",
      desc: "Confirmação humana obrigatória — sem exceções.",
      gate: true
    }, {
      glyph: "⌖",
      n: "04",
      name: "Execução",
      who: "Sniper",
      desc: "Implementa o escopo refinado e aprovado."
    }],
    diagramLabel: "diagrama",
    gateTitle: "portão de aprovação",
    gateHead: "<b>REQUER ::</b> confirmação humana · sem exceções",
    gateP: "Discovery e refinamento ocorrem autonomamente. Com aprovação humana ao gate: finalizamos a missão de reconhecimento ou seguimos com a execução da implementação.",
    pipeline: " <span class=\"node\">[ WIZARD ]</span> ──────────<span class='g1'>▶</span> <span class=\"node\">[ ESTRATEGISTA ]</span>\n mago/install           mestre · orquestra\n                               │ delega\n      ┌─────────────────────┬──┘\n      <span class='g1'>▼</span>                     <span class='g1'>▼</span>\n  <span class=\"node\">[ RANGER ]</span> ───────<span class='g1'>▶</span> <span class=\"node\">[ ARCHIVIST ]</span> ──<span class='g1'>▶</span> <span class=\"gate\">╔═ APPROVAL GATE ══╗</span>\n  discovery           refinamento       <span class=\"gate\">║ aprovação humana ║</span>\n                                        <span class=\"gate\">╚══════════════════╝</span>\n                                                 │\n                                                 <span class='g1'>▼</span>\n                                            <span class=\"node\">[ SNIPER ]</span>\n                                             execução\n <span class=\"lp\">└╌╌╌╌╌╌╌ learning loop · feedback não-bloqueante ╌╌╌╌╌╌╌╌┘</span>",
    invokeEyebrow: "o feitiço de invocação",
    invokeTitle: "Invocação",
    invokeDesc: "Conjure o grupo no repositório-alvo. Um comando e a aventura começa.",
    step1Meta: "passo 1",
    step1Title: "bootstrap",
    step2Meta: "passo 2",
    step2Title: "invocar o CLI",
    step2Desc: "Após o bootstrap, invoque o wizard de instalação:",
    cliLabel: "Strategist CLI"
  },
  en: {
    nav: {
      overview: "Overview",
      roles: "Roles",
      skills: "Skills",
      mission: "Mission Flow",
      invoke: "Invoke"
    },
    tagline: "Your experience with your demands will never be the same.",
    lede: "Turn ambiguous demands into <b>governed missions</b> for AI agents. Strategist orchestrates discovery, refinement, approval gate and controlled execution through pluggable roles — reducing improvisation, drift and risk in AI workflows.",
    ctaPrimary: "Install Strategist",
    ctaSecondary: "View architecture",
    ghBtn: "View on GitHub ↗",
    heroBadges: ["Governed AI Workflow", "Human Approval Gate", "Plugin-ready", "SDD-compatible"],
    ovEyebrow: "party console",
    ovTiles: [{
      tab: "roles",
      glyph: "☉",
      title: "Customize the roles",
      desc: "Define the party that leads each mission."
    }, {
      tab: "roles",
      glyph: "‡",
      title: "Specialize roles with multiple skills",
      desc: "Rank classes and arm them with sub-skills."
    }, {
      tab: "mission",
      glyph: "⛨",
      title: "Integration with governance models",
      desc: "Approval gate and policies inside the flow."
    }, {
      tab: "mission",
      glyph: "❒",
      title: "Documentation generated to your standards",
      desc: "Define the standard and the agent follows it."
    }],
    rolesEyebrow: "the party",
    rolesTitle: "Roles",
    rolesDesc: "Each Skill(AI agent) acts in a role(persona), under the Strategist's guidance, which defines responsibility and operating boundaries.",
    roles: [{
      glyph: "🧠",
      role: "Strategist",
      klass: "Master · Orchestrator",
      desc: "Orchestrates the mission end to end. Delegating and monitoring the roles, assembling the team and documentation while waiting for the user at the approval gate.",
      feature: true
    }, {
      glyph: "⌕",
      role: "Ranger",
      klass: "Scout · Discovery",
      desc: "Gathers requirements, maps context and returns a discovery report."
    }, {
      glyph: "✎",
      role: "Archivist",
      klass: "Archivist · Refinement",
      desc: "Deepens the Ranger's report into actionable specification and detailed analysis."
    }, {
      glyph: "⌖",
      role: "Sniper",
      klass: "Executor · Implementation",
      desc: "<i>Only after explicit human approval</i>, executes the refined implementation when scope includes delivery."
    }, {
      glyph: "✶",
      role: "Wizard",
      klass: "Invoker · Installer",
      desc: "Conjures the party into the target repository via <i>curl</i> / <i>irm</i>. Helps the user consolidate the desired standard."
    }],
    mechEyebrow: "mechanics",
    mechTitle: "Rank & Weapons",
    mechDesc: "How the role is defined by contract, how a provider becomes ranked, and which weapons each slot wields.",
    mech: [{
      glyph: "❖",
      title: "Rank / Specialization",
      desc: "When a provider declares <b>canonical_role</b> and <b>provider_class: rankeado</b>, it becomes <b>ranked</b> — a specialization recognized for that role, without gaining extra permissions."
    }, {
      glyph: "⚔",
      title: "Weapons",
      desc: "<b>Weapons</b> are the concrete providers configured in each <i>slot</i>. The role is the warrior; the weapon is the skill it wields to execute discovery, refinement or execution."
    }],
    examplesEyebrow: "examples",
    examplesTitle: "Ranked skills",
    rankLabel: "rank",
    skillLabel: "skill",
    weaponsLabel: "weapons",
    examples: [{
      glyph: "⌕",
      role: "Ranger",
      skill: "brainstorming",
      weapons: ["writing-skills"]
    }, {
      glyph: "✎",
      role: "Archivist",
      skill: "openspec-explore",
      weapons: ["openspec-propose", "openspec-apply-change", "openspec-archive-change"]
    }],
    skillsEyebrow: "special actions",
    skillsTitle: "Skills",
    skillsDesc: "Special capabilities the party can trigger during a mission.",
    skills: [{
      icon: "⚔",
      title: "Opportunist Attack",
      desc: "During actions we may encounter unforeseen problems or <i>treasure chests</i>."
    }, {
      icon: "⚑",
      title: "Side Quest",
      desc: "Small missions — moving completed tasks to the correct folder or targeted hotfixes."
    }, {
      icon: "❖",
      title: "Treasure Chest",
      desc: "At each stage we may find documentation sources to enrich the analysis."
    }, {
      icon: "⧗",
      title: "Initiative",
      desc: "Assess your team before starting your missions."
    }, {
      icon: "⛩",
      title: "Dojo",
      desc: "A controlled environment to test your skills."
    }],
    missionEyebrow: "pipeline",
    missionTitle: "Mission Flow",
    missionDesc: "Discovery and refinement run autonomously; execution proceeds only after the approval gate.",
    phaseLabel: "phase",
    phases: [{
      glyph: "⌕",
      n: "01",
      name: "Discovery",
      who: "Ranger",
      desc: "Gathers requirements and maps the context."
    }, {
      glyph: "✎",
      n: "02",
      name: "Refinement",
      who: "Archivist",
      desc: "Turns the report into an actionable specification."
    }, {
      glyph: "⛨",
      n: "03",
      name: "Approval Gate",
      who: "Human",
      desc: "Human confirmation required — no exceptions.",
      gate: true
    }, {
      glyph: "⌖",
      n: "04",
      name: "Execution",
      who: "Sniper",
      desc: "Implements the refined, approved scope."
    }],
    diagramLabel: "diagram",
    gateTitle: "approval gate",
    gateHead: "<b>REQUIRES ::</b> human confirmation · no exceptions",
    gateP: "Discovery and refinement run autonomously. With human approval at the gate, we either conclude the reconnaissance mission or proceed with implementation execution.",
    pipeline: " <span class=\"node\">[ WIZARD ]</span> ──────────<span class='g1'>▶</span> <span class=\"node\">[ STRATEGIST ]</span>\n mage/install           master · orchestrates\n                               │ delegates\n      ┌─────────────────────┬──┘\n      <span class='g1'>▼</span>                     <span class='g1'>▼</span>\n  <span class=\"node\">[ RANGER ]</span> ───────<span class='g1'>▶</span> <span class=\"node\">[ ARCHIVIST ]</span> ──<span class='g1'>▶</span> <span class=\"gate\">╔═ APPROVAL GATE ══╗</span>\n  discovery           refinement        <span class=\"gate\">║ human approval   ║</span>\n                                        <span class=\"gate\">╚══════════════════╝</span>\n                                                 │\n                                                 <span class='g1'>▼</span>\n                                            <span class=\"node\">[ SNIPER ]</span>\n                                             execution\n <span class=\"lp\">└╌╌╌╌╌╌╌ learning loop · non-blocking feedback ╌╌╌╌╌╌╌╌┘</span>",
    invokeEyebrow: "the invocation spell",
    invokeTitle: "Invoke",
    invokeDesc: "Conjure the party into the target repository. One command and the adventure begins.",
    step1Meta: "step 1",
    step1Title: "bootstrap",
    step2Meta: "step 2",
    step2Title: "invoke the CLI",
    step2Desc: "After bootstrap, invoke the install wizard:",
    cliLabel: "Strategist CLI"
  }
};
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/console/i18n.js", error: String((e && e.message) || e) }); }

// ui_kits/landing/Hero.jsx
try { (() => {
/* Landing hero — crest, grimoire wordmark, tagline, lede, badges, CTAs.
   Composes Badge + Button from the design-system bundle. */
(function () {
  const {
    Badge,
    Button
  } = window.StrategistDesignSystem_88a8a0;
  function Hero({
    lang,
    onInvoke
  }) {
    const t = window.LANDING_I18N[lang];
    return /*#__PURE__*/React.createElement("div", {
      className: "hero"
    }, /*#__PURE__*/React.createElement("div", {
      className: "authors"
    }, /*#__PURE__*/React.createElement("div", {
      className: "lbl"
    }, t.authorsLbl), /*#__PURE__*/React.createElement("div", {
      className: "names"
    }, "Sergio Lacerda & Raphael Vernil")), /*#__PURE__*/React.createElement("div", {
      className: "crest"
    }, "\u2726"), /*#__PURE__*/React.createElement("div", {
      className: "ornament"
    }, "\u2501\u2501\u2501\u2501\u2501\u2501 \u27E1 \u2501\u2501\u2501\u2501\u2501\u2501"), /*#__PURE__*/React.createElement("h1", {
      className: "title"
    }, "Strategist"), /*#__PURE__*/React.createElement("div", {
      className: "sub"
    }, t.tagline), /*#__PURE__*/React.createElement("p", {
      className: "lede",
      dangerouslySetInnerHTML: {
        __html: t.lede
      }
    }), /*#__PURE__*/React.createElement("div", {
      className: "badges"
    }, /*#__PURE__*/React.createElement(Badge, {
      label: "CI",
      value: "passing",
      tone: "ok"
    }), /*#__PURE__*/React.createElement(Badge, {
      label: "version",
      value: "1.0",
      tone: "amber"
    }), /*#__PURE__*/React.createElement(Badge, {
      label: "license",
      value: "CC BY-NC 4.0",
      tone: "orange"
    }), /*#__PURE__*/React.createElement(Badge, {
      label: "mode",
      value: "pragmatic \xB7 epic",
      tone: "ok"
    })), /*#__PURE__*/React.createElement("div", {
      className: "cta"
    }, /*#__PURE__*/React.createElement(Button, {
      variant: "primary",
      icon: "\u276F",
      onClick: onInvoke
    }, t.ctaPrimary), /*#__PURE__*/React.createElement(Button, {
      href: "https://github.com/SergioLacerda/strategist-skill",
      target: "_blank",
      rel: "noopener"
    }, t.ghBtn)));
  }
  window.Hero = Hero;
})();
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/landing/Hero.jsx", error: String((e && e.message) || e) }); }

// ui_kits/landing/PipelineMap.jsx
try { (() => {
/* ASCII pipeline map — the WIZARD → STRATEGIST → RANGER → ARCHIVIST →
   GATE → SNIPER diagram, rendered as a bordered phosphor terminal well. */
(function () {
  function PipelineMap({
    lang
  }) {
    const t = window.LANDING_I18N[lang];
    return /*#__PURE__*/React.createElement("div", {
      className: "mapwrap"
    }, /*#__PURE__*/React.createElement("div", {
      className: "map pre",
      dangerouslySetInnerHTML: {
        __html: t.pipeline
      }
    }));
  }
  window.PipelineMap = PipelineMap;
})();
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/landing/PipelineMap.jsx", error: String((e && e.message) || e) }); }

// ui_kits/landing/i18n.js
try { (() => {
/* Bilingual copy for the landing recreation (PT-BR / EN), lifted from the
   canonical pages/index.html. Exposed as window.LANDING_I18N. */
window.LANDING_I18N = {
  pt: {
    authorsLbl: "Autores",
    tagline: "A experiência com suas demandas nunca será a mesma.",
    lede: "Uma skill autônoma que <b>orquestra missões multi-fase</b> através de três papéis plugáveis. O Estrategista orquestra a missão, confiando cada etapa a um especialista e aguarda no <b>portão de aprovação</b>.",
    ctaPrimary: "Invocar a skill",
    ghBtn: "Ver no GitHub ↗",
    nag: "Sou muito chato, quero a versão normal!",
    secPipeline: "pipeline da missão",
    secRoles: "papéis / party",
    secSkills: "habilidades",
    secGate: "approval gate · regra inviolável",
    secInvoke: "o feitiço de invocação",
    pipeline: " <span class=\"node\">[ WIZARD ]</span> ──────────<span class='g1'>▶</span> <span class=\"node\">[ ESTRATEGISTA ]</span>\n mago/install           mestre · orquestra\n                               │ delega\n      ┌─────────────────────┬──┘\n      <span class='g1'>▼</span>                     <span class='g1'>▼</span>\n  <span class=\"node\">[ RANGER ]</span> ───────<span class='g1'>▶</span> <span class=\"node\">[ ARCHIVIST ]</span> ──<span class='g1'>▶</span> <span class=\"gate\">╔═ APPROVAL GATE ══╗</span>\n  discovery           refinamento       <span class=\"gate\">║ aprovação humana ║</span>\n                                        <span class=\"gate\">╚══════════════════╝</span>\n                                                 │\n                                                 <span class='g1'>▼</span>\n                                            <span class=\"node\">[ SNIPER ]</span>\n                                             execução\n <span class=\"lp\">└╌╌╌╌╌╌╌ learning loop · feedback não-bloqueante ╌╌╌╌╌╌╌╌┘</span>",
    roles: [{
      glyph: "☉",
      role: "Estrategista",
      klass: "Mestre · Orquestrador",
      feature: true,
      desc: "Conduz a missão de ponta a ponta. Seleciona o conhecimento por <i>task_type</i>, aplica <i>mission_mode</i> e governa o approval gate."
    }, {
      glyph: "⌖",
      role: "Ranger",
      klass: "Batedor · Discovery",
      desc: "Explora o terreno: levanta requisitos, mapeia o contexto e devolve um relatório de discovery."
    }, {
      glyph: "❒",
      role: "Archivist",
      klass: "Arquivista · Refinamento",
      desc: "Aprofunda o relatório do Ranger em especificação acionável e inscreve cada decisão em <i>.analysis/</i>."
    }, {
      glyph: "✜",
      role: "Sniper",
      klass: "Executor · Implementação",
      desc: "<i>Após a aprovação humana e conforme política</i>. Executa a implementação refinada quando o escopo inclui entrega."
    }, {
      glyph: "✶",
      role: "Wizard",
      klass: "Invocador · Instalador",
      desc: "Conjura a party no repositório-alvo via <i>curl</i> / <i>irm</i>. Roda o wizard e pergunta o escopo do DONE."
    }],
    skills: [{
      icon: "⚲",
      title: "Ataque de oportunidade",
      desc: "Em ações podemos encontrar problemas não previstos ou <i>baús de tesouros</i>."
    }, {
      icon: "⚑",
      title: "Missão opcional",
      desc: "Missões pequenas — mover tarefas prontas para a pasta correta ou hotfixes pontuais."
    }, {
      icon: "❖",
      title: "Baú de tesouro",
      desc: "Em cada estágio podemos encontrar fontes de documentação para enriquecer a análise."
    }],
    gateHead: "<b>REQUER ::</b> confirmação humana · sem exceções",
    gateP: "Discovery e refinamento ocorrem autonomamente. Com aprovação humana, a execução só avança quando o mission_mode permitir implementação."
  },
  en: {
    authorsLbl: "Authors",
    tagline: "Your experience with your demands will never be the same.",
    lede: "An autonomous skill that <b>orchestrates multi-phase missions</b> through three pluggable roles. The Strategist orchestrates the mission, delegating each step to a specialist and waiting at the <b>approval gate</b>.",
    ctaPrimary: "Invoke the skill",
    ghBtn: "View on GitHub ↗",
    nag: "Too fancy? I want the normal version!",
    secPipeline: "mission pipeline",
    secRoles: "roles / party",
    secSkills: "skills",
    secGate: "approval gate · inviolable rule",
    secInvoke: "the invocation spell",
    pipeline: " <span class=\"node\">[ WIZARD ]</span> ──────────<span class='g1'>▶</span> <span class=\"node\">[ STRATEGIST ]</span>\n mage/install           master · orchestrates\n                               │ delegates\n      ┌─────────────────────┬──┘\n      <span class='g1'>▼</span>                     <span class='g1'>▼</span>\n  <span class=\"node\">[ RANGER ]</span> ───────<span class='g1'>▶</span> <span class=\"node\">[ ARCHIVIST ]</span> ──<span class='g1'>▶</span> <span class=\"gate\">╔═ APPROVAL GATE ══╗</span>\n  discovery           refinement        <span class=\"gate\">║ human approval   ║</span>\n                                        <span class=\"gate\">╚══════════════════╝</span>\n                                                 │\n                                                 <span class='g1'>▼</span>\n                                            <span class=\"node\">[ SNIPER ]</span>\n                                             execution\n <span class=\"lp\">└╌╌╌╌╌╌╌ learning loop · non-blocking feedback ╌╌╌╌╌╌╌╌┘</span>",
    roles: [{
      glyph: "☉",
      role: "Strategist",
      klass: "Master · Orchestrator",
      feature: true,
      desc: "Leads the mission end to end. Selects knowledge by <i>task_type</i>, enforces <i>mission_mode</i>, and guards the approval gate."
    }, {
      glyph: "⌖",
      role: "Ranger",
      klass: "Scout · Discovery",
      desc: "Explores the terrain: gathers requirements, maps context and returns a discovery report."
    }, {
      glyph: "❒",
      role: "Archivist",
      klass: "Archivist · Refinement",
      desc: "Deepens the Ranger's report into an actionable specification and inscribes each decision in <i>.analysis/</i>."
    }, {
      glyph: "✜",
      role: "Sniper",
      klass: "Executor · Implementation",
      desc: "<i>After human approval and policy check</i>. Executes refined implementation only when scope includes delivery."
    }, {
      glyph: "✶",
      role: "Wizard",
      klass: "Invoker · Installer",
      desc: "Conjures the party into the target repository via <i>curl</i> / <i>irm</i>. Runs the wizard and asks the DONE scope."
    }],
    skills: [{
      icon: "⚲",
      title: "Opportunist Attack",
      desc: "During actions we may encounter unforeseen problems or <i>treasure chests</i>."
    }, {
      icon: "⚑",
      title: "Side Quest",
      desc: "Small missions — moving completed tasks to the correct folder or targeted hotfixes."
    }, {
      icon: "❖",
      title: "Treasure Chest",
      desc: "At each stage we may find documentation sources to enrich the analysis."
    }],
    gateHead: "<b>REQUIRES ::</b> human confirmation · no exceptions",
    gateP: "Discovery and refinement run autonomously. After human approval, execution proceeds only when mission_mode allows implementation."
  }
};
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/landing/i18n.js", error: String((e && e.message) || e) }); }

__ds_ns.Logo = __ds_scope.Logo;

__ds_ns.Badge = __ds_scope.Badge;

__ds_ns.Button = __ds_scope.Button;

__ds_ns.SectionLabel = __ds_scope.SectionLabel;

__ds_ns.GatePanel = __ds_scope.GatePanel;

__ds_ns.InstallBlock = __ds_scope.InstallBlock;

__ds_ns.RoleCard = __ds_scope.RoleCard;

__ds_ns.SkillPanel = __ds_scope.SkillPanel;

})();
