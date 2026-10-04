import type { Localized } from './features';

export interface MissionPhase {
  id: string;
  glyph: string;
  number: string;
  title: Localized;
  who: Localized;
  description: Localized;
  roles: string[];
  tools: string[];
  features: string[];
  artifacts: string[];
  rules: string[];
  gate?: boolean;
}

export interface RouteCell { family?: string; name: Localized; sub?: Localized; code?: boolean; optional?: boolean; }
export interface MissionRoute { id: string; tag: string; label: Localized; cells: RouteCell[]; }

const t = (pt: string, en: string): Localized => ({ pt, en });

export const MISSION_PHASES: MissionPhase[] = [
  { id: 'discovery', glyph: '⌕', number: '01', title: t('Discovery', 'Discovery'), who: t('Ranger', 'Ranger'), description: t('Levanta requisitos e mapeia o contexto.', 'Gathers requirements and maps the context.'), roles: ['Ranger'], tools: ['Prompt Intake', 'Context Enrichment', 'LEVELING'], features: ['treasure', 'cartography', 'effort', 'initiative', 'crit'], artifacts: ['Pending analysis', 'Evidence Pack'], rules: ['Role Contract', 'Budget enforcement'] },
  { id: 'refinement', glyph: '✎', number: '02', title: t('Refinamento', 'Refinement'), who: t('Archivist', 'Archivist'), description: t('Transforma o relatório em especificação acionável.', 'Turns the report into an actionable specification.'), roles: ['Archivist'], tools: ['Dossier Builder', 'Response Critic', 'LEVELING'], features: ['treasure', 'effort', 'initiative'], artifacts: ['Refined dossier', 'Confidence Report'], rules: ['Role Contract', 'Handoff validation'] },
  { id: 'approval', glyph: '⛨', number: '03', title: t('Approval Gate', 'Approval Gate'), who: t('Humano', 'Human'), description: t('Confirmação humana obrigatória — sem exceções.', 'Human confirmation required — no exceptions.'), roles: ['Human', 'Strategist'], tools: [], features: ['opportunity'], artifacts: [], rules: ['Approval rules', 'Fingerprint / integrity', 'Stage transition rules'], gate: true },
  { id: 'materialization', glyph: '⌖', number: '04', title: t('Materialização', 'Materialization'), who: t('Sniper', 'Sniper'), description: t('Materializa documentação aprovada e prepara o executor handoff.', 'Materializes approved documentation and prepares the executor handoff.'), roles: ['Sniper'], tools: ['Learning Curator'], features: [], artifacts: ['Approved docs', 'Archived report', 'Executor handoff'], rules: ['Approval rules', 'Handoff validation', 'Budget enforcement'] },
  { id: 'external', glyph: '⚙', number: '05', title: t('Executor Externo', 'External Executor'), who: t('SDD / CI / Humano', 'SDD / CI / Human'), description: t('Consome o handoff e é dono da implementação.', 'Consumes the handoff and owns implementation.'), roles: ['External executor'], tools: [], features: [], artifacts: ['Executor handoff'], rules: [] },
];

export const MISSION_ROUTES: MissionRoute[] = [
  { id: 'full', tag: 'FULL', label: t('missão completa · discovery → refinamento → gate → entrega', 'full mission · discovery → refinement → gate → delivery'), cells: [
    { family: 'ROLE', name: t('Scout', 'Scout') }, { family: 'ROLE', name: t('Ranger', 'Ranger') }, { family: 'ROLE', name: t('Archivist', 'Archivist') }, { family: 'MECHANISM', name: t('Approval Gate', 'Approval Gate') }, { family: 'ROLE', name: t('Sniper', 'Sniper') }, { name: t('Executor externo', 'External executor') },
  ] },
  { id: 'short', tag: 'SHORT', label: t('execução delimitada · preparação proporcional', 'bounded execution · proportional preparation'), cells: [
    { family: 'ROLE', name: t('Scout', 'Scout') }, { family: 'FEAT', name: t('Critical Hit / Riposte', 'Critical Hit / Riposte') }, { family: 'STAGE', name: t('SHORT', 'SHORT') }, { family: 'MECHANISM', name: t('Aprovação (se exigida)', 'Approval (if required)'), optional: true }, { family: 'ROLE', name: t('Sniper', 'Sniper') }, { family: 'ARTIFACT', name: t('Completion Report', 'Completion Report') },
  ] },
  { id: 'wizard', tag: 'WIZARD', label: t('instalação · do binário à primeira missão', 'installation · from binary to first mission'), cells: [
    { name: t('strategist install --wizard', 'strategist install --wizard'), code: true, sub: t('go install · curl · Releases', 'go install · curl · Releases') }, { family: 'STAGE', name: t('ROSTER', 'ROSTER'), sub: t('importa e mapeia as armas', 'imports and maps the weapons') }, { family: 'ROLE', name: t('Ranger | Archivist', 'Ranger | Archivist'), sub: t('seleção das armas', 'weapon selection') }, { name: t('Customização*', 'Customization*'), sub: t('dependências · ex.: JEV', 'dependencies · e.g. JEV'), optional: true }, { family: 'ARTIFACT', name: t('.strategist/', '.strategist/'), sub: t('config do grupo', 'party config') }, { name: t('Primeira missão', 'First mission'), sub: t('FULL · SHORT', 'FULL · SHORT') },
  ] },
  { id: 'roster', tag: 'ROSTER', label: t('configuração · das armas disponíveis ao binding', 'configuration · from available weapons to binding'), cells: [
    { name: t('Pacotes disponíveis', 'Available packages') }, { family: 'TOOL', name: t('Weapon Discovery', 'Weapon Discovery') }, { family: 'TOOL', name: t('Compatibility Resolver', 'Compatibility Resolver') }, { family: 'ROLE', name: t('Wizard · seleção', 'Wizard · selection') }, { family: 'ARTIFACT', name: t('Weapon Binding', 'Weapon Binding') }, { name: t('Runtime · resolução', 'Runtime · resolution') },
  ] },
];
