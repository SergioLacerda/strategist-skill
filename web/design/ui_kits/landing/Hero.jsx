/* Landing hero — crest, grimoire wordmark, tagline, lede, badges, CTAs.
   Composes Badge + Button from the design-system bundle. */
(function () {
  const { Badge, Button } = window.StrategistDesignSystem_88a8a0;

  function Hero({ lang, onInvoke }) {
    const t = window.LANDING_I18N[lang];
    return (
      <div className="hero">
        <div className="authors">
          <div className="lbl">{t.authorsLbl}</div>
          <div className="names">Sergio Lacerda &amp; Raphael Vernil</div>
        </div>
        <div className="crest">✦</div>
        <div className="ornament">━━━━━━ ⟡ ━━━━━━</div>
        <h1 className="title">Strategist</h1>
        <div className="sub">{t.tagline}</div>
        <p className="lede" dangerouslySetInnerHTML={{ __html: t.lede }} />
        <div className="badges">
          <Badge label="CI" value="passing" tone="ok" />
          <Badge label="version" value="1.0" tone="amber" />
          <Badge label="license" value="CC BY-NC 4.0" tone="orange" />
          <Badge label="mode" value="pragmatic · epic" tone="ok" />
        </div>
        <div className="cta">
          <Button variant="primary" icon="❯" onClick={onInvoke}>{t.ctaPrimary}</Button>
          <Button href="https://github.com/SergioLacerda/strategist-skill" target="_blank" rel="noopener">{t.ghBtn}</Button>
        </div>
      </div>
    );
  }

  window.Hero = Hero;
})();
