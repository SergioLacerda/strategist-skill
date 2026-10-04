import type { Localized } from './features';

export interface StructureFeature { id: string; glyph: string; name: string; description: Localized; link?: string; }
export interface StructureRole { glyph: string; name: string; weapon?: string; description?: Localized; fixed?: boolean; planned?: boolean; external?: boolean; }
export interface StructureTool { name: string; description?: Localized; }
const t = (pt: string, en: string): Localized => ({ pt, en });

export const STRUCTURE_FEATURES: StructureFeature[] = [
  { id: 'initiative', glyph: '⧗', name: 'Initiative', description: t('aconselha a composição', 'advises the composition'), link: 'initiative' },
  { id: 'crit', glyph: '↯', name: 'Critical Hit', description: t('solicita o fluxo SHORT', 'requests the SHORT flow'), link: 'crit' },
  { id: 'opportunity', glyph: '⚔', name: 'Opportunity Attack', description: t('propõe uma ação lateral', 'proposes a lateral action'), link: 'opportunity' },
  { id: 'senses', glyph: '◉', name: 'Keen Senses', description: t('percepção e seleção contextual', 'contextual perception and selection') },
];

export const STRUCTURE_ROLES: StructureRole[] = [
  { glyph: '🧭', name: 'Scout', fixed: true, description: t('operações nativas', 'native operations') },
  { glyph: '⌕', name: 'Ranger', weapon: 'Brainstorming' },
  { glyph: '✎', name: 'Archivist', weapon: 'OpenSpec' },
  { glyph: '⌖', name: 'Sniper', weapon: 'Implementation Weapon' },
  { glyph: '✶', name: 'Wizard', fixed: true, description: t('conduz o ROSTER', 'drives the ROSTER') },
  { glyph: '◎', name: 'Sharpshooter', weapon: 'Precise Shot', planned: true, external: true },
  { glyph: '⌘', name: 'Cartographer', weapon: 'Cartography', planned: true },
  { glyph: '◈', name: 'Jeweler', weapon: 'Treasure Chest', planned: true, external: true },
];

export const STRUCTURE_TOOLS: StructureTool[] = [
  { name: 'LEVELING', description: t('resolve esforço e orçamento', 'resolves effort and budget') },
  { name: 'Prompt Intake' }, { name: 'Context Enrichment' }, { name: 'Dossier Builder' }, { name: 'Response Critic' }, { name: 'Learning Curator' }, { name: 'Weapon Discovery', description: t('inventaria as armas', 'inventories the weapons') }, { name: 'Compatibility Resolver', description: t('cruza armas e contratos', 'crosses weapons and contracts') }, { name: 'Sextant', description: t('navegação estrutural', 'structural navigation') },
];
