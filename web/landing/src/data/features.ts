export type Lang = 'pt' | 'en';
export type Localized = { pt: string; en: string };
export type Family = 'ROLE' | 'WEAPON' | 'FEAT' | 'TOOL' | 'MECHANISM' | 'STAGE' | 'ARTIFACT';

export interface FeatureNode {
  key: string;
  family: Family;
  name: Localized;
  planned?: boolean;
  sub?: Localized;
}

export interface FeatureDefinition {
  id: string;
  glyph: string;
  title: Localized;
  tag: Localized;
  how: Localized[];
  origin: Array<[Localized, Localized]>;
  rules: Localized[];
  flow: FeatureNode[];
  planned?: boolean;
}

const text = (pt: string, en: string): Localized => ({ pt, en });
const node = (key: string, family: Family, pt: string, en = pt, options: Pick<FeatureNode, 'planned' | 'sub'> = {}): FeatureNode => ({
  key,
  family,
  name: text(pt, en),
  ...options,
});

export const FEATURE_ORDER = ['treasure', 'effort', 'cartography', 'initiative', 'opportunity', 'crit', 'roster', 'dojo'] as const;

export const FEATURES: Record<(typeof FEATURE_ORDER)[number], FeatureDefinition> = {
  treasure: {
    id: 'treasure', glyph: '❖', title: text('Baú do Tesouro', 'Treasure Chest'),
    tag: text('conhecimento reutilizável', 'reusable knowledge'),
    how: [text('Indexa fontes offline e organiza conhecimento em joias vinculadas à origem.', 'Indexes offline sources and organizes knowledge into source-linked jewels.')],
    origin: [[text('Weapon', 'Weapon'), text('skill-for-hire', 'skill-for-hire')], [text('Artifact', 'Artifact'), text('Gem / Scroll', 'Gem / Scroll')]],
    rules: [text('A joia proposta não é uma joia verificada.', 'A proposed jewel is not a verified jewel.'), text('A fonte completa continua disponível.', 'The full source remains available.')],
    flow: [node('jeweler', 'ROLE', 'Jeweler', 'Jeweler', { planned: true }), node('chest', 'WEAPON', 'Treasure Chest'), node('gems', 'ARTIFACT', 'Gem / Scroll')],
  },
  effort: {
    id: 'effort', glyph: '⚖', title: text('Esforço proporcional', 'Proportional effort'),
    tag: text('confiança e orçamento', 'confidence and budget'),
    how: [text('Relaciona evidência, confiança e orçamento antes de ampliar a missão.', 'Relates evidence, confidence and budget before widening a mission.')],
    origin: [[text('Tool', 'Tool'), text('LEVELING', 'LEVELING')], [text('Mechanism', 'Mechanism'), text('Budget enforcement', 'Budget enforcement')]],
    rules: [text('Confiança é sinal, não autorização.', 'Confidence is a signal, not authorization.')],
    flow: [node('sharp', 'ROLE', 'Sharpshooter', 'Sharpshooter', { planned: true }), node('shot', 'WEAPON', 'Precise Shot', 'Precise Shot', { planned: true }), node('report', 'ARTIFACT', 'Confidence Report'), node('valid', 'MECHANISM', 'Confidence validation'), node('leveling', 'TOOL', 'LEVELING')],
  },
  cartography: {
    id: 'cartography', glyph: '⌘', title: text('Cartografia', 'Cartography'),
    tag: text('conhecimento estrutural', 'structural knowledge'),
    how: [text('Mantém um mapa do território e transforma navegação em artefato consultável.', 'Maintains a map of the territory and turns navigation into a queryable artifact.')],
    origin: [[text('Role', 'Role'), text('Cartographer', 'Cartographer')], [text('Tool', 'Tool'), text('Sextant', 'Sextant')]],
    rules: [text('O mapa informa o contexto; não concede autoridade.', 'The map informs context; it grants no authority.')],
    flow: [node('carto', 'ROLE', 'Cartographer', 'Cartographer', { planned: true }), node('map', 'WEAPON', 'Cartography', 'Cartography', { planned: true }), node('atlas', 'ARTIFACT', 'Atlas'), node('sextant', 'TOOL', 'Sextant')],
  },
  initiative: {
    id: 'initiative', glyph: '⧗', title: text('Iniciativa', 'Initiative'),
    tag: text('composição do grupo', 'party composition'),
    how: [text('Avalia a composição antes de iniciar uma missão.', 'Assesses the party composition before a mission begins.')],
    origin: [[text('Feat', 'Feat'), text('Initiative', 'Initiative')], [text('Tool', 'Tool'), text('LEVELING', 'LEVELING')]],
    rules: [text('A recomendação não substitui a decisão humana.', 'A recommendation does not replace human decision.')],
    flow: [node('initiative', 'FEAT', 'Initiative'), node('role', 'ROLE', 'Papel ativo', 'Active role'), node('budget', 'MECHANISM', 'Budget enforcement')],
  },
  opportunity: {
    id: 'opportunity', glyph: '⚔', title: text('Ataque de oportunidade', 'Opportunity Attack'),
    tag: text('ação lateral', 'lateral action'),
    how: [text('Detecta uma decisão arquitetural que pode ser proposta como side quest.', 'Detects an architectural decision that may be proposed as a side quest.')],
    origin: [[text('Feat', 'Feat'), text('Opportunity Attack', 'Opportunity Attack')], [text('Artifact', 'Artifact'), text('ADR · side quest', 'ADR · side quest')]],
    rules: [text('A side quest aparece no Approval Gate antes da materialização.', 'The side quest appears at the Approval Gate before materialization.')],
    flow: [node('opp', 'FEAT', 'Opportunity Attack'), node('adr', 'ARTIFACT', 'ADR · side quest'), node('gate', 'MECHANISM', 'Approval Gate')],
  },
  crit: {
    id: 'crit', glyph: '↯', title: text('Acerto crítico / Riposte', 'Critical Hit / Riposte'),
    tag: text('rota proporcional', 'proportional route'),
    how: [text('Resolve uma demanda delimitada sem ampliar a autoridade do papel.', 'Resolves a bounded request without widening the role authority.')],
    origin: [[text('Stage', 'Stage'), text('SHORT', 'SHORT')], [text('Artifact', 'Artifact'), text('Completion Report', 'Completion Report')]],
    rules: [text('Casos incompletos permanecem em FULL.', 'Incomplete cases remain on FULL.')],
    flow: [node('feats', 'FEAT', 'Critical Hit / Riposte'), node('short', 'STAGE', 'SHORT'), node('approval', 'MECHANISM', 'Approval rules'), node('sniper', 'ROLE', 'Sniper'), node('completion', 'ARTIFACT', 'Completion Report')],
  },
  roster: {
    id: 'roster', glyph: '❒', title: text('Roster de armas', 'Weapon roster'),
    tag: text('configuração de providers', 'provider configuration'),
    how: [text('Descobre pacotes, cruza contratos e persiste bindings para os slots da missão.', 'Discovers packages, crosses contracts and persists bindings for mission slots.')],
    origin: [[text('Stage', 'Stage'), text('ROSTER', 'ROSTER')], [text('Tool', 'Tool'), text('Compatibility Resolver', 'Compatibility Resolver')]],
    rules: [text('Binding explícito; heurística não escolhe provider.', 'Explicit binding; heuristics do not choose a provider.')],
    flow: [node('wizard', 'ROLE', 'Wizard'), node('roster', 'STAGE', 'ROSTER'), node('discovery', 'TOOL', 'Weapon Discovery'), node('binding', 'ARTIFACT', 'Weapon Binding')],
  },
  dojo: {
    id: 'dojo', glyph: '⛩', title: text('Dojo', 'Dojo'), tag: text('planejado', 'planned'), planned: true,
    how: [text('Ambiente controlado para testar habilidades.', 'A controlled environment for testing skills.')],
    origin: [[text('Stage', 'Stage'), text('a definir', 'to be defined')]],
    rules: [text('A classificação e o conteúdo ainda serão definidos.', 'Classification and content remain to be defined.')],
    flow: [],
  },
};
