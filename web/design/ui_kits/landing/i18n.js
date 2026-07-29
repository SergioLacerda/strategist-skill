/* Bilingual copy for the landing recreation (PT-BR / EN), lifted from the
   canonical pages/index.html. Exposed as window.LANDING_I18N.
   LEGACY / REFERENCE ONLY — not the live landing (see web/landing/src/pages/).
   Sniper's "Executor · Implementação" wording below predates the current
   documentation-only Sniper contract; left as-is deliberately (see
   .analysis/refined/20260712-docs-landing-updates-treasure-scout/tasks.md T12). */
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
    roles: [
      { glyph: "☉", role: "Estrategista", klass: "Mestre · Orquestrador", feature: true, desc: "Conduz a missão de ponta a ponta. Seleciona o conhecimento por <i>task_type</i>, aplica <i>mission_mode</i> e governa o approval gate." },
      { glyph: "⌖", role: "Ranger", klass: "Batedor · Discovery", desc: "Explora o terreno: levanta requisitos, mapeia o contexto e devolve um relatório de discovery." },
      { glyph: "❒", role: "Archivist", klass: "Arquivista · Refinamento", desc: "Aprofunda o relatório do Ranger em especificação acionável e inscreve cada decisão em <i>.analysis/</i>." },
      { glyph: "✜", role: "Sniper", klass: "Executor · Implementação", desc: "<i>Após a aprovação humana e conforme política</i>. Executa a implementação refinada quando o escopo inclui entrega." },
      { glyph: "✶", role: "Wizard", klass: "Invocador · Instalador", desc: "Conjura a party no repositório-alvo via <i>curl</i> / <i>irm</i>. Roda o wizard e pergunta o escopo do DONE." },
    ],
    skills: [
      { icon: "⚲", title: "Ataque de oportunidade", desc: "Avalia se a missão principal merece ADR; quando cabe, propõe a ADR como side quest no Approval Gate." },
      { icon: "⚑", title: "Side quests", desc: "Outras missões fora do escopo principal; são notificadas no Approval Gate antes de qualquer materialização." },
      { icon: "❖", title: "Baú do tesouro", desc: "Fontes offline já indexáveis e listáveis por comando, usadas para enriquecer a missão com contexto local." },
      { icon: "✦", title: "Acerto crítico", desc: "Resolve demandas pequenas de organização, movendo análises concluídas, refinadas ou pendentes para as pastas corretas." },
    ],
    gateHead: "<b>REQUER ::</b> confirmação humana · sem exceções",
    gateP: "Discovery e refinamento ocorrem autonomamente. Com aprovação humana, a execução só avança quando o mission_mode permitir implementação.",
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
    roles: [
      { glyph: "☉", role: "Strategist", klass: "Master · Orchestrator", feature: true, desc: "Leads the mission end to end. Selects knowledge by <i>task_type</i>, enforces <i>mission_mode</i>, and guards the approval gate." },
      { glyph: "⌖", role: "Ranger", klass: "Scout · Discovery", desc: "Explores the terrain: gathers requirements, maps context and returns a discovery report." },
      { glyph: "❒", role: "Archivist", klass: "Archivist · Refinement", desc: "Deepens the Ranger's report into an actionable specification and inscribes each decision in <i>.analysis/</i>." },
      { glyph: "✜", role: "Sniper", klass: "Executor · Implementation", desc: "<i>After human approval and policy check</i>. Executes refined implementation only when scope includes delivery." },
      { glyph: "✶", role: "Wizard", klass: "Invoker · Installer", desc: "Conjures the party into the target repository via <i>curl</i> / <i>irm</i>. Runs the wizard and asks the DONE scope." },
    ],
    skills: [
      { icon: "⚲", title: "Opportunity Attack", desc: "Evaluates whether the main mission deserves an ADR; when it fits, proposes the ADR as a side quest at the Approval Gate." },
      { icon: "⚑", title: "Side Quests", desc: "Other missions outside the main scope; they are reported at the Approval Gate before any materialization." },
      { icon: "❖", title: "Treasure Chest", desc: "Offline sources that can already be indexed and listed by command, used to enrich the mission with local context." },
      { icon: "✦", title: "Critical Hit", desc: "Handles small organization requests by moving completed, refined, or pending analyses into the correct folders." },
    ],
    gateHead: "<b>REQUIRES ::</b> human confirmation · no exceptions",
    gateP: "Discovery and refinement run autonomously. After human approval, execution proceeds only when mission_mode allows implementation.",
  },
};
