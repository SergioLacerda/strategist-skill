import type { Localized } from './features';

export interface StructureFeature { id: string; glyph: string; name: string; description: Localized; link?: string; }
export interface OrchestratorItem { glyph: string; name: string; description: Localized; }
export interface StructureRole { glyph: string; name: string; weapon?: string; description?: Localized; internal?: boolean; planned?: boolean; external?: boolean; group: 0 | 1; }
export interface StructureTool { name: string; description?: Localized; }
const t = (pt: string, en: string): Localized => ({ pt, en });

export const STRUCTURE_FEATURES: StructureFeature[] = [
  { id: 'initiative', glyph: '⧗', name: 'Initiative', description: t('aconselha a composição', 'advises the composition'), link: 'initiative' },
  { id: 'crit', glyph: '↯', name: 'Critical Hit', description: t('solicita o fluxo SHORT', 'requests the SHORT flow'), link: 'crit' },
  { id: 'riposte', glyph: '↯', name: 'Riposte', description: t('solicita o fluxo SHORT', 'requests the SHORT flow'), link: 'crit' },
  { id: 'opp', glyph: '⚔', name: 'Opportunity Attack', description: t('propõe uma ação lateral', 'proposes a lateral action'), link: 'opportunity' },
  { id: 'keen', glyph: '◉', name: 'Keen Senses', description: t('percepção e seleção contextual', 'contextual perception and selection') },
];

export const STRUCTURE_ORCHESTRATOR: OrchestratorItem[] = [
  { glyph: '▶', name: 'FULL', description: t('missão completa', 'full mission') },
  { glyph: '▷', name: 'SHORT', description: t('execução delimitada', 'bounded execution') },
  { glyph: '❒', name: 'ROSTER', description: t('configuração de armas', 'weapon configuration') },
  { glyph: '⛨', name: 'Approval Gate', description: t('autorização humana', 'human authorization') },
  { glyph: '⚖', name: 'Leveling', description: t('esforço proporcional', 'proportional effort') },
];

export const STRUCTURE_ROLES: StructureRole[] = [
  { glyph: '🧭', name: 'Scout', internal: true, description: t('operações nativas', 'native operations'), group: 0 },
  { glyph: '⌕', name: 'Ranger', weapon: 'Brainstorming', group: 0 },
  { glyph: '✎', name: 'Archivist', weapon: 'OpenSpec', group: 0 },
  { glyph: '⌖', name: 'Sniper', weapon: 'Implementation Weapon', group: 0 },
  { glyph: '✶', name: 'Wizard', internal: true, description: t('conduz o ROSTER', 'drives the ROSTER'), group: 0 },
  { glyph: '◎', name: 'Sharpshooter', weapon: 'Precise Shot', planned: true, external: true, group: 1 },
  { glyph: '⌘', name: 'Cartographer', weapon: 'Cartography', planned: true, group: 1 },
  { glyph: '◈', name: 'Jeweler', weapon: 'Treasure Chest', planned: true, external: true, group: 1 },
];

export const STRUCTURE_TOOLS: StructureTool[] = [
  { name: 'Leveling', description: t('resolve esforço e orçamento', 'resolves effort and budget') },
  { name: 'Prompt Intake' }, { name: 'Context Enrichment' }, { name: 'Dossier Builder' }, { name: 'Response Critic' }, { name: 'Learning Curator' },
  { name: 'Weapon Discovery', description: t('inventaria as armas', 'inventories the weapons') },
  { name: 'Compatibility Resolver', description: t('cruza armas e contratos', 'crosses weapons and contracts') },
  { name: 'Binding Resolver', description: t('formaliza o binding', 'formalizes the binding') },
  { name: 'Sextant', description: t('navegação estrutural', 'structural navigation') },
];
