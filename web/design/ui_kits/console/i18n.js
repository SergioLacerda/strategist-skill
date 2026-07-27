/* Strategist Console (v2) — bilingual copy (PT-BR primary / EN).
   Exposed as window.CONSOLE_I18N.
   LEGACY / REFERENCE ONLY — not the live landing (see web/landing/src/pages/).
   Sniper's "Executor · Implementação" wording below predates the current
   documentation-only Sniper contract; left as-is deliberately (see
   .analysis/refined/20260712-docs-landing-updates-treasure-scout/tasks.md T12). */
window.CONSOLE_I18N = {
  pt: {
    nav: { overview: "Visão Geral", roles: "Papéis", skills: "Habilidades", mission: "Fluxo da Missão", invoke: "Invocação" },
    tagline: "A experiência com suas demandas nunca será a mesma.",
    lede: "Transforme demandas ambíguas em <b>missões governadas</b> para agentes IA. O Strategist orquestra discovery, refinamento, approval gate e execução controlada através de papéis plugáveis — reduzindo improviso, drift e risco em workflows com IA.",
    ctaPrimary: "Instalar Strategist",
    ctaSecondary: "Ver arquitetura",
    ghBtn: "Ver no GitHub ↗",
    heroBadges: ["Workflow de IA Governado", "Gate de Aprovação Humana", "Pronto para Plugins", "Compatível com SDD"],

    ovEyebrow: "console do grupo",
    ovTiles: [
      { tab: "roles", glyph: "☉", title: "Customize os papéis", desc: "Defina o grupo que conduz cada missão." },
      { tab: "roles", glyph: "‡", title: "Especialize papéis com múltiplas skills", desc: "Rankeie classes e arme-as com sub-skills." },
      { tab: "mission", glyph: "⛨", title: "Integração com modelos de governança", desc: "Approval gate e políticas dentro do fluxo." },
      { tab: "mission", glyph: "❒", title: "Documentação gerada conforme seus padrões", desc: "Determine o padrão e o agente cumpre." },
    ],

    rolesEyebrow: "a grupo",
    rolesTitle: "Papéis",
    rolesDesc: "Cada Skill(IA agent) atua em um papel(persona), sob orientação do estrategista, que define responsabilidade e limites de atuação.",
    roles: [
      { glyph: "🧠", role: "Estrategista", klass: "Mestre · Orquestrador", desc: "Orquestra a missão de ponta a ponta. Delegando e monitorando os papéis, reunindo o time e documentação para aguardar usuário no gate de aprovação.", feature: true },
      { glyph: "⌕", role: "Ranger", klass: "Batedor · Discovery", desc: "Levanta requisitos, mapeia o contexto e devolve um relatório de discovery." },
      { glyph: "✎", role: "Archivist", klass: "Arquivista · Refinamento", desc: "Aprofunda o relatório do Ranger em especificação acionável e análise detalhada." },
      { glyph: "⌖", role: "Sniper", klass: "Executor · Implementação", desc: "<i>Somente após aprovação humana explícita</i>, executa a implementação refinada quando o escopo inclui entrega." },
      { glyph: "✶", role: "Wizard", klass: "Invocador · Instalador", desc: "Conjura o grupo no repositório-alvo via <i>curl</i> / <i>irm</i>. Auxilia o usuário para consolidar o padrão desejado." },
    ],

    mechEyebrow: "mecânica",
    mechTitle: "Rank & Armas",
    mechDesc: "Como o contrato define o papel, o provider rankeado e as armas de cada slot.",
    mech: [
      { glyph: "❖", title: "Rank / Especialização", desc: "Quando o provider declara <b>canonical_role</b> e <b>provider_class: rankeado</b>, ele se torna <b>rankeado</b> — uma especialização reconhecida para aquele papel, sem ganhar permissões extras." },
      { glyph: "⚔", title: "Armas", desc: "As <b>armas</b> são os providers concretos configurados em cada <i>slot</i>. O papel é o guerreiro; a arma é a skill que ele empunha para executar discovery, refinement ou execution." },
    ],
    examplesEyebrow: "exemplos",
    examplesTitle: "Skills rankeadas",
    rankLabel: "rank",
    skillLabel: "skill",
    weaponsLabel: "armas",
    examples: [
      { glyph: "⌕", role: "Ranger", skill: "brainstorming", weapons: ["writing-skills"] },
      { glyph: "✎", role: "Archivist", skill: "openspec-explore", weapons: ["openspec-propose", "openspec-apply-change", "openspec-archive-change"] },
    ],

    skillsEyebrow: "ações especiais",
    skillsTitle: "Habilidades",
    skillsDesc: "Capacidades especiais que o grupo pode acionar durante a missão.",
    skills: [
      { icon: "⚔", title: "Ataque de oportunidade", desc: "Avalia se a missão principal merece ADR; quando cabe, propõe a ADR como side quest no Approval Gate." },
      { icon: "⚑", title: "Side quests", desc: "Outras missões fora do escopo principal; são notificadas no Approval Gate antes de qualquer materialização." },
      { icon: "❖", title: "Baú do tesouro", desc: "Fontes offline já indexáveis e listáveis por comando, usadas para enriquecer a missão com contexto local." },
      { icon: "⧗", title: "Iniciativa", desc: "Avalie seu time antes de começar suas missões." },
      { icon: "✦", title: "Acerto crítico", desc: "Resolve demandas pequenas de organização, movendo análises concluídas, refinadas ou pendentes para as pastas corretas." },
      { icon: "⛩", title: "Dojo", desc: "Ambiente controlado para testar suas habilidades." },
    ],

    missionEyebrow: "pipeline",
    missionTitle: "Fluxo da Missão",
    missionDesc: "Discovery e refinamento rodam autonomamente; a execução só avança após o portão de aprovação.",
    phaseLabel: "fase",
    phases: [
      { glyph: "⌕", n: "01", name: "Discovery", who: "Ranger", desc: "Levanta requisitos e mapeia o contexto." },
      { glyph: "✎", n: "02", name: "Refinamento", who: "Archivist", desc: "Transforma o relatório em especificação acionável." },
      { glyph: "⛨", n: "03", name: "Approval Gate", who: "Humano", desc: "Confirmação humana obrigatória — sem exceções.", gate: true },
      { glyph: "⌖", n: "04", name: "Execução", who: "Sniper", desc: "Implementa o escopo refinado e aprovado." },
    ],
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
    cliLabel: "Strategist CLI",
  },

  en: {
    nav: { overview: "Overview", roles: "Roles", skills: "Skills", mission: "Mission Flow", invoke: "Invoke" },
    tagline: "Your experience with your demands will never be the same.",
    lede: "Turn ambiguous demands into <b>governed missions</b> for AI agents. Strategist orchestrates discovery, refinement, approval gate and controlled execution through pluggable roles — reducing improvisation, drift and risk in AI workflows.",
    ctaPrimary: "Install Strategist",
    ctaSecondary: "View architecture",
    ghBtn: "View on GitHub ↗",
    heroBadges: ["Governed AI Workflow", "Human Approval Gate", "Plugin-ready", "SDD-compatible"],

    ovEyebrow: "party console",
    ovTiles: [
      { tab: "roles", glyph: "☉", title: "Customize the roles", desc: "Define the party that leads each mission." },
      { tab: "roles", glyph: "‡", title: "Specialize roles with multiple skills", desc: "Rank classes and arm them with sub-skills." },
      { tab: "mission", glyph: "⛨", title: "Integration with governance models", desc: "Approval gate and policies inside the flow." },
      { tab: "mission", glyph: "❒", title: "Documentation generated to your standards", desc: "Define the standard and the agent follows it." },
    ],

    rolesEyebrow: "the party",
    rolesTitle: "Roles",
    rolesDesc: "Each Skill(AI agent) acts in a role(persona), under the Strategist's guidance, which defines responsibility and operating boundaries.",
    roles: [
      { glyph: "🧠", role: "Strategist", klass: "Master · Orchestrator", desc: "Orchestrates the mission end to end. Delegating and monitoring the roles, assembling the team and documentation while waiting for the user at the approval gate.", feature: true },
      { glyph: "⌕", role: "Ranger", klass: "Scout · Discovery", desc: "Gathers requirements, maps context and returns a discovery report." },
      { glyph: "✎", role: "Archivist", klass: "Archivist · Refinement", desc: "Deepens the Ranger's report into actionable specification and detailed analysis." },
      { glyph: "⌖", role: "Sniper", klass: "Executor · Implementation", desc: "<i>Only after explicit human approval</i>, executes the refined implementation when scope includes delivery." },
      { glyph: "✶", role: "Wizard", klass: "Invoker · Installer", desc: "Conjures the party into the target repository via <i>curl</i> / <i>irm</i>. Helps the user consolidate the desired standard." },
    ],

    mechEyebrow: "mechanics",
    mechTitle: "Rank & Weapons",
    mechDesc: "How the role is defined by contract, how a provider becomes ranked, and which weapons each slot wields.",
    mech: [
      { glyph: "❖", title: "Rank / Specialization", desc: "When a provider declares <b>canonical_role</b> and <b>provider_class: rankeado</b>, it becomes <b>ranked</b> — a specialization recognized for that role, without gaining extra permissions." },
      { glyph: "⚔", title: "Weapons", desc: "<b>Weapons</b> are the concrete providers configured in each <i>slot</i>. The role is the warrior; the weapon is the skill it wields to execute discovery, refinement or execution." },
    ],
    examplesEyebrow: "examples",
    examplesTitle: "Ranked skills",
    rankLabel: "rank",
    skillLabel: "skill",
    weaponsLabel: "weapons",
    examples: [
      { glyph: "⌕", role: "Ranger", skill: "brainstorming", weapons: ["writing-skills"] },
      { glyph: "✎", role: "Archivist", skill: "openspec-explore", weapons: ["openspec-propose", "openspec-apply-change", "openspec-archive-change"] },
    ],

    skillsEyebrow: "special actions",
    skillsTitle: "Skills",
    skillsDesc: "Special capabilities the party can trigger during a mission.",
    skills: [
      { icon: "⚔", title: "Opportunity Attack", desc: "Evaluates whether the main mission deserves an ADR; when it fits, proposes the ADR as a side quest at the Approval Gate." },
      { icon: "⚑", title: "Side Quests", desc: "Other missions outside the main scope; they are reported at the Approval Gate before any materialization." },
      { icon: "❖", title: "Treasure Chest", desc: "Offline sources that can already be indexed and listed by command, used to enrich the mission with local context." },
      { icon: "⧗", title: "Initiative", desc: "Assess your team before starting your missions." },
      { icon: "✦", title: "Critical Hit", desc: "Handles small organization requests by moving completed, refined, or pending analyses into the correct folders." },
      { icon: "⛩", title: "Dojo", desc: "A controlled environment to test your skills." },
    ],

    missionEyebrow: "pipeline",
    missionTitle: "Mission Flow",
    missionDesc: "Discovery and refinement run autonomously; execution proceeds only after the approval gate.",
    phaseLabel: "phase",
    phases: [
      { glyph: "⌕", n: "01", name: "Discovery", who: "Ranger", desc: "Gathers requirements and maps the context." },
      { glyph: "✎", n: "02", name: "Refinement", who: "Archivist", desc: "Turns the report into an actionable specification." },
      { glyph: "⛨", n: "03", name: "Approval Gate", who: "Human", desc: "Human confirmation required — no exceptions.", gate: true },
      { glyph: "⌖", n: "04", name: "Execution", who: "Sniper", desc: "Implements the refined, approved scope." },
    ],
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
    cliLabel: "Strategist CLI",
  },
};
