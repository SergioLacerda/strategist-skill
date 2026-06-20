/* ASCII pipeline map — the WIZARD → STRATEGIST → RANGER → ARCHIVIST →
   GATE → SNIPER diagram, rendered as a bordered phosphor terminal well. */
(function () {
  function PipelineMap({ lang }) {
    const t = window.LANDING_I18N[lang];
    return (
      <div className="mapwrap">
        <div className="map pre" dangerouslySetInnerHTML={{ __html: t.pipeline }} />
      </div>
    );
  }
  window.PipelineMap = PipelineMap;
})();
