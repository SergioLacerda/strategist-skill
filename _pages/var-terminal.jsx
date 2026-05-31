/* global React */
// VARIATION A — TERMINAL CRAWL
// Pure amber phosphor terminal. Boot log, ASCII pipeline, role stat-cards.

function VarTerminal() {
  const html = `
  <div class="winbar">
    <span class="dot r"></span><span class="dot y"></span><span class="dot g"></span>
    <span style="margin-left:10px">strategist@dungeon — bash — 120×40</span>
  </div>

  <div class="wrap">
    <div class="boot pre"><span class="prompt">guildmaster@keep</span>:<span class="amber">~</span>$ ./summon strategist
[<span class="ok">ok</span>] arcane runtime carregado            [<span class="ok">ok</span>] approval gate :: ARMADO
[<span class="ok">ok</span>] knowledge.index montado               [<span class="ok">ok</span>] learning loop :: standby
[<span class="ok">ok</span>] 5 slots vinculados à party            modo :: <span class="amber">epic</span></div>

    <pre class="ascii glow"> ____ _____ ____      _  _____ _____ ____ ___ ____ _____
/ ___|_   _|  _ \\    / \\|_   _| ____/ ___|_ _/ ___|_   _|
\\___ \\ | | | |_) |  / _ \\ | | |  _|| |  _ | |\\___ \\ | |
 ___) || | |  _ <  / ___ \\| | | |__| |_| || | ___) || |
|____/ |_| |_| \\_\\/_/   \\_\\_| |_____\\____|___|____/ |_|</pre>

    <p class="tagline">Uma skill autônoma que <b>orquestra missões multi-fase</b> através de cinco papéis plugáveis — guiados por um <b>approval gate</b> obrigatório. O mestre delega; nunca empunha a espada sozinho.</p>

    <div class="badges" style="margin-top:22px">
      <span class="badge ok"><b>CI</b><i>passing</i></span>
      <span class="badge amb"><b>version</b><i>1.0</i></span>
      <span class="badge org"><b>license</b><i>MIT</i></span>
      <span class="badge ok"><b>mode</b><i>pragmatic · epic</i></span>
    </div>

    <h2 class="sec">mapa da masmorra · pipeline da missão</h2>
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

    <h2 class="sec">a party · cinco papéis</h2>
    <div class="cards">
      ${roleCard("☉", "Estrategista", "Mestre · Orquestrador", "INT 20 · WIS 18", "Comando", 95, "Conduz a missão de ponta a ponta. Seleciona o conhecimento por <span class='amber'>task_type</span>, delega a cada slot e mantém o approval gate. Decide tudo; executa nada.")}
      ${roleCard("✶", "Wizard", "Invocador · Instalador", "ARC 17 · DEX 12", "Bootstrap", 70, "Conjura o harness no repositório-alvo via <span class='amber'>curl</span> / <span class='amber'>irm</span>. Roda o wizard de configuração e grava os arcanos em <span class='amber'>.strategist/</span>.")}
      ${roleCard("⌖", "Ranger", "Batedor · Discovery", "PER 18 · DEX 16", "Recon", 78, "Explora o terreno antes da batalha: levanta requisitos, mapeia o contexto e devolve um relatório de discovery limpo para o mestre.")}
      ${roleCard("❒", "Archivist", "Arquivista · Refino & Docs", "INT 18 · WIS 16", "Lore", 82, "Refina os achados do Ranger em especificação acionável e registra tudo em <span class='amber'>.analysis/</span> — a crônica viva da missão.")}
      ${roleCard("✜", "Sniper", "Executor · Implementação", "DEX 19 · STR 15", "Strike", 88, "Só dispara <b>após aprovação humana</b>. Executa a implementação refinada com precisão cirúrgica — um tiro, um commit.")}
      ${gateCardInline()}
    </div>

    <h2 class="sec">approval gate · regra inviolável</h2>
    <div class="gate-panel">
      <div class="lock">⛨</div>
      <div>
        <h3>O Sniper jamais dispara sem aprovação</h3>
        <p>Discovery e refino correm em fluxo livre, mas a câmara de execução permanece trancada. O Estrategista nunca invoca o Sniper sem aprovação humana explícita — a porta só abre pela sua mão.</p>
        <div class="req">REQUER :: confirmação humana · sem exceções · sem auto-merge</div>
      </div>
    </div>

    <h2 class="sec">invocação · instalação</h2>
    <div class="install">
      <div class="ihead">Linux / macOS / WSL</div>
      <pre><span class="c">curl</span> -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh <span class="o">| bash</span></pre>
      <div class="ihead" style="border-top:1px solid var(--line)">Windows PowerShell</div>
      <pre><span class="c">irm</span> https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.ps1 <span class="o">| iex</span></pre>
    </div>

    <div class="foot">
      <span>SergioLacerda/strategist-skill · standalone + SDD harness</span>
      <span class="prompt">█</span>
    </div>
  </div>`;

  return React.createElement("div", {
    className: "page va crt",
    dangerouslySetInnerHTML: { __html: html },
  });
}

function roleCard(glyph, role, klass, stats, barName, pct, desc) {
  return `<div class="card">
    <div class="glyph">${glyph}</div>
    <div class="role">${role}</div>
    <div class="klass">${klass}</div>
    <div class="stats">
      <div class="stat">ATRIBUTOS<b>${stats}</b></div>
    </div>
    <p>${desc}</p>
    <div class="bar"><i style="width:${pct}%"></i></div>
    <div class="stat" style="margin-top:6px">${barName.toUpperCase()} · ${pct}/100</div>
  </div>`;
}

function gateCardInline() {
  return `<div class="card" style="border-style:dashed;border-color:var(--orange);background:linear-gradient(135deg,rgba(207,122,44,.08),transparent)">
    <div class="glyph" style="color:var(--orange)">⛨</div>
    <div class="role" style="color:var(--orange)">Approval Gate</div>
    <div class="klass">Portão · Humano-no-loop</div>
    <p style="margin-top:18px">Não é um membro da party — é a porta trancada entre refino e execução. Sem o selo humano, o Sniper não passa.</p>
    <div class="bar"><i style="width:100%;background:var(--orange)"></i></div>
    <div class="stat" style="margin-top:6px;color:var(--orange)">SELO · obrigatório</div>
  </div>`;
}

window.VarTerminal = VarTerminal;
