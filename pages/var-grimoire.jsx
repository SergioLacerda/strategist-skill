/* global React */
// VARIATION B — ARCANE GRIMOIRE (refinada)
// Hero grimório + mapa ASCII da masmorra (da A) + cards dos papéis (estilo A,
// título em fonte grimório, sem atributos/rodapé) + bloco Invocação (da C).

function VarGrimoire() {
  const html = `
  <div class="wrap">
    <div class="hero">
      <div class="crest">✦</div>
      <div class="ornament">━━━━━━ ⟡ ━━━━━━</div>
      <h1 class="title">Strategist</h1>
      <div class="sub">o mestre da masmorra das missões</div>
      <p class="lede">Uma skill autônoma que <b>orquestra missões multi-fase</b> através de cinco papéis plugáveis. O Estrategista delega a cada câmara da masmorra e guarda o <b>portão de aprovação</b> — nunca empunha a lâmina por conta própria.</p>
      <div class="badges">
        <span class="badge ok"><b>CI</b><i>passing</i></span>
        <span class="badge amb"><b>version</b><i>1.0</i></span>
        <span class="badge org"><b>license</b><i>MIT</i></span>
        <span class="badge ok"><b>mode</b><i>pragmatic · epic</i></span>
      </div>
    </div>

    <div class="scroll-sec">
      <div class="seclabel">mapa da masmorra · pipeline da missão</div>
      <div class="secrule">⟡ ━━━━━━━━━━━━━━━━━━ ⟡</div>

      <div class="mapwrap">
        <div class="map pre"> <span class="node">[ WIZARD ]</span> ──summon──▶ <span class="node">[ ESTRATEGISTA ]</span>
 mago/install        mestre · orquestra
                          │ delega
        ┌─────────────────┼──────────────────┐
        ▼                 ▼                  
  <span class="node">[ RANGER ]</span> ───────▶ <span class="node">[ ARCHIVIST ]</span> ──▶ <span class="gate">╔═ APPROVAL GATE ═╗</span>
  discovery            refino · docs            ║ humano  aprova ║
                                                ╚════════╤═══════╝
                                                         ▼
                                                   <span class="node">[ SNIPER ]</span>
                                                    execução
 <span class="lp">└╌╌╌╌╌╌╌╌╌╌╌ learning loop · feedback não-bloqueante ╌╌╌╌╌╌╌╌╌╌╌╌╌┘</span></div>
      </div>
    </div>

    <div class="scroll-sec">
      <div class="seclabel">a party · cinco papéis</div>
      <div class="secrule">⟡ ━━━━━━━━━━━━━━━━━━ ⟡</div>

      <div class="cards-wrap">
        <div class="cards">
          ${card("☉", "Estrategista", "Mestre · Orquestrador", "Conduz a missão de ponta a ponta. Seleciona o conhecimento por <i>task_type</i>, delega a cada slot e guarda o approval gate. Decide tudo; executa nada.", true)}
          ${card("⌖", "Ranger", "Batedor · Discovery", "Explora o terreno antes da batalha: levanta requisitos, mapeia o contexto e devolve um relatório de discovery limpo ao mestre.")}
          ${card("❒", "Archivist", "Arquivista · Refino & Docs", "Refina os achados do Ranger em especificação acionável e inscreve cada decisão em <i>.analysis/</i> — a crônica viva da missão.")}
          ${card("✜", "Sniper", "Executor · Implementação", "Só dispara <i>após a aprovação humana</i>. Executa a implementação refinada com precisão cirúrgica — um tiro, um commit.")}
          ${card("✶", "Wizard", "Invocador · Instalador", "Conjura o harness no repositório-alvo via <i>curl</i> / <i>irm</i>. Roda o wizard de configuração e sela os arcanos em <i>.strategist/</i>.")}
        </div>
      </div>
    </div>

    <div class="scroll-sec">
      <div class="seclabel">approval gate · regra inviolável</div>
      <div class="secrule">⟡ ━━━━━━━━━━━━━━━━━━ ⟡</div>
      <div class="gatewrap">
        <div class="gate-portal">
          <div class="gate-wall"></div>
          <div class="gate-body">
            <div class="lock">⛨</div>
            <div class="req-head"><b>REQUER ::</b> confirmação humana · sem exceções · sem auto-merge</div>
            <p>Discovery e refino correm em fluxo livre, mas a câmara de execução permanece trancada. O Estrategista nunca invoca o Sniper sem aprovação humana explícita — a porta só abre pela sua mão.</p>
          </div>
          <div class="gate-wall"></div>
        </div>
      </div>
    </div>

    <div class="install-wrap">
      <div class="seclabel">o feitiço de invocação</div>
      <div class="secrule">⟡ ━━━━━━━━━━━━━━━━━━ ⟡</div>
      <div class="invoke-panel">
        <div class="install">
          <div class="ihead">Linux / macOS / WSL</div>
          <pre><span class="c">curl</span> -fsSL …/bootstrap.sh <span class="o">| bash</span></pre>
          <div class="ihead" style="border-top:1px solid var(--line)">Windows PowerShell</div>
          <pre><span class="c">irm</span> …/bootstrap.ps1 <span class="o">| iex</span></pre>
        </div>
        <p class="src">raw.githubusercontent.com/SergioLacerda/strategist-skill/main/</p>
      </div>
    </div>

    <div class="runefoot">✦ SergioLacerda / strategist-skill ✦ standalone + SDD harness ✦</div>
  </div>`;

  return React.createElement("div", {
    className: "page vb crt",
    dangerouslySetInnerHTML: { __html: html },
  });
}

function card(glyph, role, klass, desc, feat) {
  return `<div class="card hasglyph${feat ? " feature" : ""}">
    <div class="glyph">${glyph}</div>
    <div class="role">${role}</div>
    <div class="klass">${klass}</div>
    <p>${desc}</p>
  </div>`;
}

function gateCard_unused() {
  return `<div class="card gate hasglyph">
    <div class="glyph">⛨</div>
    <div class="role">Approval Gate</div>
    <div class="klass">Portão · Humano-no-loop</div>
    <p>A porta selada entre refino e execução. O Estrategista <i>jamais</i> invoca o Sniper sem aprovação humana explícita — sem exceções, sem auto-merge.</p>
  </div>`;
}

window.VarGrimoire = VarGrimoire;
