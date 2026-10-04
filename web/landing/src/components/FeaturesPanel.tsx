import { useEffect, useMemo, useState } from 'react';
import { FEATURES, FEATURE_ORDER, type Lang } from '../data/features';

function read(key: string, fallback: string): string { try { return localStorage.getItem(key) || fallback; } catch { return fallback; } }
function save(key: string, value: string): void { try { localStorage.setItem(key, value); } catch { /* storage is optional */ } }

export default function FeaturesPanel() {
  const [lang, setLang] = useState<Lang>(() => read('strategist_console_lang', 'pt') === 'en' ? 'en' : 'pt');
  const [selected, setSelected] = useState(() => {
    const value = read('strategist_console_feature', FEATURE_ORDER[0]);
    return (FEATURE_ORDER as readonly string[]).includes(value) ? value : FEATURE_ORDER[0];
  });
  const feature = useMemo(() => FEATURES[selected as (typeof FEATURE_ORDER)[number]], [selected]);

  useEffect(() => {
    const onLang = (event: Event) => setLang((event as CustomEvent<string>).detail === 'en' ? 'en' : 'pt');
    window.addEventListener('strategist:lang', onLang);
    return () => window.removeEventListener('strategist:lang', onLang);
  }, []);

  const choose = (id: string) => { setSelected(id); save('strategist_console_feature', id); };
  return (
    <div className="features-layout">
      <div className="feature-list" role="list" aria-label={lang === 'pt' ? 'Funcionalidades' : 'Features'}>
        {FEATURE_ORDER.map((id) => {
          const item = FEATURES[id];
          return <button type="button" className={`feature-tile${id === selected ? ' selected' : ''}`} onClick={() => choose(id)} key={id}>
            <span className="feature-glyph" aria-hidden="true">{item.glyph}</span>
            <span><strong>{item.title[lang]}</strong><small>{item.tag[lang]}</small></span>
            {item.planned && <em>{lang === 'pt' ? 'planejado' : 'planned'}</em>}
          </button>;
        })}
      </div>
      <article className="feature-detail" key={selected}>
        <div className="detail-kicker">{feature.tag[lang]}</div>
        <h2>{feature.glyph} {feature.title[lang]}</h2>
        {feature.planned && <span className="planned-badge">{lang === 'pt' ? 'evolução planejada' : 'planned evolution'}</span>}
        <div className="detail-copy">
          {feature.how.map((line) => <p key={line[lang]}>{line[lang]}</p>)}
        </div>
        <div className="detail-section"><h3>{lang === 'pt' ? 'Composição' : 'Composition'}</h3><div className="node-flow">
          {feature.flow.length === 0 && <span className="empty-note">{lang === 'pt' ? 'a definir' : 'to be defined'}</span>}
          {feature.flow.map((item, index) => <span className="flow-node" key={item.key}>
            <span className={`family family-${item.family.toLowerCase()}`}>{item.family}</span>
            <b>{item.name[lang]}</b>{item.planned && <em>{lang === 'pt' ? 'planejado' : 'planned'}</em>}
            {index < feature.flow.length - 1 && <i aria-hidden="true">→</i>}
          </span>)}
        </div></div>
        <div className="detail-grid"><div><h3>{lang === 'pt' ? 'Origem' : 'Origin'}</h3>{feature.origin.map(([key, value]) => <p className="origin-row" key={key[lang]}><span>{key[lang]}</span>{value[lang]}</p>)}</div><div><h3>{lang === 'pt' ? 'Regras' : 'Rules'}</h3><ul>{feature.rules.map((rule) => <li key={rule[lang]}>{rule[lang]}</li>)}</ul></div></div>
        {selected === 'treasure' && <details className="jewel-disclosure"><summary>{lang === 'pt' ? 'Joias/Jewels' : 'Jewels'}</summary><p>{lang === 'pt' ? <><code>index</code> descobre e propõe joias; <code>mine</code> é a curadoria humana.</> : <><code>index</code> discovers and proposes jewels; <code>mine</code> is human curation.</>}</p><div className="wchips">{(lang === 'pt' ? ['documentos', 'source cards', 'evidence packs', 'proposta', 'aceita', 'verificada', 'joias/jewels'] : ['documents', 'source cards', 'evidence packs', 'proposed', 'accepted', 'verified', 'jewels']).map((chip) => <span className="wchip" key={chip}>◆ {chip}</span>)}</div></details>}
      </article>
    </div>
  );
}
