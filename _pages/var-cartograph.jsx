/* global React */
// VARIATION C — DUNGEON CARTOGRAPH
// Top-down 2D tile map of stone rooms + corridors, with flow & install panels.

function VarCartograph() {
  const conn = (label) => `
    <div style="display:flex;flex-direction:column;align-items:center;justify-content:center;width:42px;color:var(--amber-dim)">
      <div style="font-size:9px;letter-spacing:.08em;color:var(--faint);margin-bottom:4px">${label}</div>
      <div style="font-size:18px">──▶</div>
    </div>`;

  const room = (sig, role, klass, no, desc, cls = "") => `
    <div class="room ${cls}" style="width:198px">
      <div class="no">${no}</div>
      <div class="sig">${sig}</div>
      <div class="role">${role}</div>
      <div class="klass">${klass}</div>
      <p>${desc}</p>
    </div>`;

  const html = `
  <div class="wrap">
    <div class="top">
      <div>
        <div class="brand">SergioLacerda · strategist-skill</div>
        <h1>Mapa da Masmorra</h1>
      </div>
      <p class="lede">Cinco câmaras, um mestre. O <b>Estrategista</b> orquestra a travessia — discovery, refino e execução — sempre atravessando o <b>portão de aprovação</b> antes do golpe final.</p>
    </div>

    <div class="maparea">
      <div class="legend">◆ mestre · ✶ entrada · ⛨ portão<br><span>— caminho da missão →</span></div>

      <div style="display:flex;flex-direction:column;align-items:center;gap:0;margin-top:6px">

        <div style="display:flex;align-items:center;justify-content:center;gap:0">
          ${room("✶", "Wizard", "Invocador · Install", "ENTRADA", "Conjura o harness no repo via curl/irm e sela os arcanos em .strategist/.")}
          ${conn("summon")}
          ${room("◆", "Estrategista", "Mestre · Orquestrador", "SALÃO", "Seleciona conhecimento por task_type, delega aos slots e guarda o gate. Decide tudo; executa nada.", "master")}
        </div>

        <div style="display:flex;flex-direction:column;align-items:center">
          <div style="font-size:9px;letter-spacing:.1em;color:var(--faint);margin:8px 0 2px">delega ↓</div>
          <div style="width:2px;height:26px;background:repeating-linear-gradient(var(--line-2) 0 5px,transparent 5px 11px)"></div>
        </div>

        <div style="display:flex;align-items:stretch;justify-content:center;gap:0">
          ${room("⌖", "Ranger", "Batedor · Discovery", "RECON", "Avança no escuro, levanta requisitos e mapeia o contexto.")}
          ${conn("refina")}
          ${room("❒", "Archivist", "Arquivista · Refino", "LORE", "Refina os achados em spec acionável e registra em .analysis/.")}
          ${conn("aprova")}
          ${room("⛨", "Approval Gate", "Humano-no-loop", "PORTÃO", "Porta selada: sem aprovação humana, o Sniper não passa.", "gate")}
          ${conn("dispara")}
          ${room("✜", "Sniper", "Executor · Implementação", "STRIKE", "Após o selo, executa a implementação. Um tiro, um commit.")}
        </div>

        <div style="display:flex;align-items:center;gap:10px;margin-top:18px;font-size:11px;color:var(--green);letter-spacing:.06em">
          <span style="flex:1;height:1px;background:repeating-linear-gradient(90deg,var(--green-dim) 0 6px,transparent 6px 12px)"></span>
          ◀ learning loop · feedback não-bloqueante
          <span style="flex:1;height:1px;background:repeating-linear-gradient(90deg,var(--green-dim) 0 6px,transparent 6px 12px)"></span>
        </div>
      </div>
    </div>

    <div class="panels">
      <div class="panel">
        <h3>Fluxo da missão</h3>
        <div class="flowline">
          <span class="step">Wizard</span><span class="arr">▶</span>
          <span class="step">Estrategista</span><span class="arr">▶</span>
          <span class="step">Ranger</span><span class="arr">▶</span>
          <span class="step">Archivist</span><span class="arr">▶</span>
          <span class="step lock">⛨ Gate</span><span class="arr">▶</span>
          <span class="step">Sniper</span>
        </div>
        <p style="font-size:12.5px;color:var(--muted);margin:16px 0 0;line-height:1.6">Dois modos, mesmo pipeline: <span class="amber">pragmatic</span> (tom analítico) e <span class="amber">epic</span> (tom estratégico). Knowledge Index alimenta cada fase; o learning loop registra os resultados sem nunca bloquear a missão.</p>
        <div class="badges" style="margin-top:16px">
          <span class="badge ok"><b>CI</b><i>passing</i></span>
          <span class="badge amb"><b>version</b><i>1.0</i></span>
          <span class="badge org"><b>license</b><i>MIT</i></span>
        </div>
      </div>

      <div class="panel">
        <h3>Invocação</h3>
        <div class="install" style="border:none">
          <div class="ihead" style="padding-left:0">Linux / macOS / WSL</div>
          <pre style="padding-left:0"><span class="c">curl</span> -fsSL …/bootstrap.sh <span class="o">| bash</span></pre>
          <div class="ihead" style="padding-left:0;border-top:1px solid var(--line)">Windows PowerShell</div>
          <pre style="padding-left:0"><span class="c">irm</span> …/bootstrap.ps1 <span class="o">| iex</span></pre>
        </div>
        <p style="font-size:11.5px;color:var(--faint);margin:10px 0 0">raw.githubusercontent.com/SergioLacerda/strategist-skill/main/</p>
      </div>
    </div>
  </div>`;

  return React.createElement("div", {
    className: "page vc crt",
    dangerouslySetInnerHTML: { __html: html },
  });
}

window.VarCartograph = VarCartograph;
