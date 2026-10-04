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
    <div className="interaction-hint" aria-hidden={touched}>{lang === 'pt' ? '☝ clique numa fase' : '☝ click a phase'}</div>
    <div className="phases accordion">
      {MISSION_PHASES.map((phase, index) => <article className={`phase-card${phase.gate ? ' gate' : ''}${openPhase === index ? ' open' : ''}`} key={phase.id}>
        <button type="button" className="phase-trigger" onClick={() => toggle(index)} aria-expanded={openPhase === index}>
          <span className="pico">{phase.glyph}</span><span><small>{lang === 'pt' ? 'fase' : 'phase'} {phase.number}</small><strong>{phase.title[lang]}</strong><span>{phase.description[lang]}</span></span><em>{label(phase.who[lang])}</em><b aria-hidden="true">{openPhase === index ? '⌃' : '⌄'}</b>
        </button>
        {openPhase === index && <div className="phase-detail">
          {phase.gate && <div className="gate-card"><div className="gate-lock">⛨</div><strong>Approval Gate</strong><p>{lang === 'pt' ? 'Aceitação humana antes de qualquer materialização; nenhuma mutação de código-fonte.' : 'Human acceptance before any materialization; no source-code mutation.'}</p></div>}
          <div className="phase-columns"><Detail title={lang === 'pt' ? 'papéis' : 'roles'} values={phase.roles.map(label)} /><Detail title="tools" values={phase.tools} /><div><h4>{lang === 'pt' ? 'funcionalidades' : 'features'}</h4>{phase.features.length ? <div className="detail-links">{phase.features.map((id) => <button type="button" onClick={() => openFeature(id)} key={id}>{FEATURES[id as keyof typeof FEATURES]?.title[lang] || id}</button>)}</div> : <p className="empty-note">{lang === 'pt' ? 'nenhum nesta fase' : 'none in this phase'}</p>}</div><Detail title={lang === 'pt' ? 'artefatos' : 'artifacts'} values={phase.artifacts} /><Detail title={lang === 'pt' ? 'regras' : 'rules'} values={phase.rules} /></div>
        </div>}
      </article>)}
    </div>
  </div>;
}

function Detail({ title, values }: { title: string; values: string[] }) { return <div><h4>{title}</h4>{values.length ? <ul>{values.map((value) => <li key={value}>{value}</li>)}</ul> : <p className="empty-note">—</p>}</div>; }
