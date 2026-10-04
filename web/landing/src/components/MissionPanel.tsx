import { useEffect, useState } from 'react';
import { FEATURES } from '../data/features';
import { MISSION_PHASES } from '../data/architecture';
import type { Lang } from '../data/features';

function read(key: string, fallback: string): string { try { return localStorage.getItem(key) || fallback; } catch { return fallback; } }

export default function MissionPanel() {
  const [lang, setLang] = useState<Lang>(() => read('strategist_console_lang', 'pt') === 'en' ? 'en' : 'pt');
  const [openPhase, setOpenPhase] = useState<number | null>(null);
  const [touched, setTouched] = useState(false);
  useEffect(() => {
    const onLang = (event: Event) => setLang((event as CustomEvent<string>).detail === 'en' ? 'en' : 'pt');
    window.addEventListener('strategist:lang', onLang);
    return () => window.removeEventListener('strategist:lang', onLang);
  }, []);
  const label = (value: string) => ({ Human: lang === 'pt' ? 'Humano' : 'Human', 'External executor': lang === 'pt' ? 'Executor externo' : 'External executor' }[value] || value);
  const toggle = (index: number) => { setTouched(true); setOpenPhase((current) => current === index ? null : index); };
  const openFeature = (id: string) => { try { localStorage.setItem('strategist_console_feature', id); } catch {} window.dispatchEvent(new CustomEvent('strategist:tab', { detail: 'features' })); };
  return <div className="mission-flow">
    <div className="mission-flow-heading"><div className="subhead"><span className="h">{lang === 'pt' ? 'Fluxo da missão' : 'Mission flow'}</span><span className="rule" /><span className="meta interaction-hint" aria-hidden={touched}>☝ {lang === 'pt' ? 'clique numa fase' : 'click a phase'}</span></div></div>
    <div className="phases accordion">
      {MISSION_PHASES.map((phase, index) => <article className={`phase-card${phase.gate ? ' gate' : ''}${openPhase === index ? ' open' : ''}${!touched && index === 0 ? ' hint-phase' : ''}`} key={phase.id}>
        <button type="button" className="phase-trigger" onClick={() => toggle(index)} aria-expanded={openPhase === index}>
          <span className="pico">{phase.glyph}</span><span><small>{lang === 'pt' ? 'fase' : 'phase'} {phase.number}</small><strong>{phase.title[lang]}</strong><span>{phase.description[lang]}</span></span><em>{label(phase.who[lang])}</em><b aria-hidden="true">{openPhase === index ? '⌃' : '⌄'}</b>
        </button>
        {openPhase === index && <div className="phase-detail">
          {phase.gate && <GateVisual lang={lang} />}
          <div className="phase-columns"><RoleDetail values={phase.roles} lang={lang} /><Detail title="tools" values={phase.tools} kind="tools" /><div className="phase-feature-detail"><h4>{lang === 'pt' ? 'funcionalidades' : 'features'}</h4>{phase.features.length ? <div className="detail-links">{phase.features.map((id) => <button type="button" onClick={() => openFeature(id)} key={id}>{FEATURES[id as keyof typeof FEATURES]?.title[lang] || id}</button>)}</div> : <p className="empty-note">{lang === 'pt' ? 'nenhum nesta fase' : 'none in this phase'}</p>}</div><Detail title={lang === 'pt' ? 'artefatos' : 'artifacts'} values={phase.artifacts} kind="artifacts" /><Detail title={lang === 'pt' ? 'regras' : 'rules'} values={phase.rules} kind="rules" /></div>
        </div>}
      </article>)}
    </div>
  </div>;
}

function GateVisual({ lang }: { lang: Lang }) {
  return <div className="gate-panel-visual">
    <div className="gate-jamb gate-jamb-left" aria-hidden="true" /><div className="gate-jamb gate-jamb-right" aria-hidden="true" />
    <div className="gate-arch" aria-hidden="true"><div className="gate-keystone" /><div className="gate-lock">⛨</div></div>
    <strong className="gate-visual-title"><b>{lang === 'pt' ? 'REQUER ::' : 'REQUIRES ::'}</b> {lang === 'pt' ? 'aceitação humana · nenhuma mutação de código-fonte' : 'human acceptance · no source-code mutation'}</strong>
    <p>{lang === 'pt' ? 'Discovery e refinamento rodam autonomamente. No Approval Gate, o usuário aceita a análise, adiciona itens faltantes ou rejeita o escopo. Só então Sniper pode materializar documentação ou preparar o executor handoff.' : 'Discovery and refinement run autonomously. At the Approval Gate, the user accepts the analysis, adds missing items, or rejects the scope. Only then can Sniper materialize documentation or prepare the executor handoff.'}</p>
  </div>;
}

const ROLE_DETAILS: Record<string, { glyph: string; klass: { pt: string; en: string }; description: { pt: string; en: string } }> = {
  Human: {
    glyph: '☉',
    klass: { pt: 'Decide', en: 'Decides' },
    description: { pt: 'Aceita, complementa ou rejeita o escopo antes da materialização.', en: 'Accepts, completes, or rejects scope before materialization.' },
  },
  Strategist: {
    glyph: '🧠',
    klass: { pt: 'Mestre · Orquestrador', en: 'Master · Orchestrator' },
    description: { pt: 'Coordena a missão e guarda as fronteiras do fluxo.', en: 'Coordinates the mission and guards the flow boundaries.' },
  },
  Ranger: {
    glyph: '⌕',
    klass: { pt: 'Batedor · Discovery', en: 'Scout · Discovery' },
    description: { pt: 'Levanta requisitos, mapeia o contexto e devolve um relatório.', en: 'Gathers requirements, maps context, and returns a discovery report.' },
  },
  Archivist: {
    glyph: '✎',
    klass: { pt: 'Arquivista · Refinamento', en: 'Archivist · Refinement' },
    description: { pt: 'Refina o relatório em documentação acionável.', en: 'Refines the report into actionable documentation.' },
  },
  Sniper: {
    glyph: '⌖',
    klass: { pt: 'Executor Documental · Materialização', en: 'Documentation Executor · Materialization' },
    description: { pt: 'Materializa documentação aprovada sem implementar código-fonte.', en: 'Materializes approved documentation without implementing source code.' },
  },
  'External executor': {
    glyph: '⚙',
    klass: { pt: 'Executor externo', en: 'External executor' },
    description: { pt: 'Recebe o handoff aprovado para executar a mudança.', en: 'Receives the approved handoff to execute the change.' },
  },
};

function RoleDetail({ values, lang }: { values: string[]; lang: Lang }) {
  return <div className="phase-role-detail"><h4>{lang === 'pt' ? 'papéis' : 'roles'}</h4>{values.length ? <div className="phase-role-list">{values.map((value) => {
    const detail = ROLE_DETAILS[value];
    return <div className="phase-role" key={value}><span className="phase-role-glyph" aria-hidden="true">{detail?.glyph || '•'}</span><div><strong>{value === 'Human' ? (lang === 'pt' ? 'Humano' : 'Human') : value === 'External executor' ? (lang === 'pt' ? 'Executor externo' : 'External executor') : value}</strong>{detail && <><small>{detail.klass[lang]}</small><p>{detail.description[lang]}</p></>}</div></div>;
  })}</div> : <p className="empty-note">—</p>}</div>;
}

function Detail({ title, values, kind }: { title: string; values: string[]; kind?: 'tools' | 'artifacts' | 'rules' }) { return <div className={`phase-detail-group${kind ? ` ${kind}` : ''}`}><h4>{title}</h4>{values.length ? <ul>{values.map((value) => <li key={value}>{value}</li>)}</ul> : <p className="empty-note">—</p>}</div>; }
