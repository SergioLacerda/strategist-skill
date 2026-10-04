import { useEffect, useState } from 'react';
import { FEATURES, FEATURE_ORDER, FEATURE_UI, FAMILY_LABELS, type FeatureDefinition, type Family, type Lang } from '../data/features';

function read(key: string, fallback: string): string { try { return localStorage.getItem(key) || fallback; } catch { return fallback; } }
function save(key: string, value: string): void { try { localStorage.setItem(key, value); } catch { /* storage is optional */ } }
function localized(value: { pt: string; en: string }, lang: Lang): string { return value[lang]; }
function familyClass(family: Family): string { return `family-${family.toLowerCase()}`; }

export default function FeaturesPanel() {
  const [lang, setLang] = useState<Lang>(() => read('strategist_console_lang', 'pt') === 'en' ? 'en' : 'pt');
  const [selected, setSelected] = useState<(typeof FEATURE_ORDER)[number]>(() => {
    const value = read('strategist_console_feature', FEATURE_ORDER[0]);
    return (FEATURE_ORDER as readonly string[]).includes(value) ? value as (typeof FEATURE_ORDER)[number] : FEATURE_ORDER[0];
  });
  const feature = FEATURES[selected];
  const ui = FEATURE_UI[lang];

  useEffect(() => {
    const onLang = (event: Event) => setLang((event as CustomEvent<string>).detail === 'en' ? 'en' : 'pt');
    const onFeature = (event: Event) => {
      const value = (event as CustomEvent<string>).detail;
      if ((FEATURE_ORDER as readonly string[]).includes(value)) setSelected(value as (typeof FEATURE_ORDER)[number]);
    };
    window.addEventListener('strategist:lang', onLang);
    window.addEventListener('strategist:feature', onFeature);
    return () => {
      window.removeEventListener('strategist:lang', onLang);
      window.removeEventListener('strategist:feature', onFeature);
    };
  }, []);

  const choose = (id: (typeof FEATURE_ORDER)[number]) => { setSelected(id); save('strategist_console_feature', id); };
  const evolving = (item: FeatureDefinition) => item.planned || item.flow.some((node) => node.planned);

  return <div className="feature-console">
    <div className="feature-legend" aria-label={ui.legend}>
      <span className="legend-title">{ui.legend}</span>
      {(Object.keys(FAMILY_LABELS[lang]) as Family[]).map((family) => <span className={`legend-item ${familyClass(family)}`} key={family}><b>{family}</b><small>{FAMILY_LABELS[lang][family]}</small></span>)}
    </div>
    <div className="feature-grid" role="list" aria-label={lang === 'pt' ? 'Funcionalidades' : 'Features'}>
      {FEATURE_ORDER.map((id) => {
        const item = FEATURES[id];
        const itemEvolving = evolving(item);
        return <button type="button" className={`feature-tile${id === selected ? ' selected' : ''}`} onClick={() => choose(id)} key={id} aria-pressed={id === selected}>
          <span className="feature-tile-glyph" aria-hidden="true">{item.glyph}</span>
          <span className="feature-tile-copy"><strong>{item.title[lang]}</strong><small>{item.tag[lang]}</small></span>
          <span className="feature-tile-footer"><span className={`feature-status ${itemEvolving ? 'evolving' : 'published'}`}>{itemEvolving ? ui.evolving : ui.live}</span></span>
        </button>;
      })}
    </div>
    <article className="feature-sheet" key={selected}>
      <header className="feature-sheet-head">
        <span className="feature-sheet-glyph" aria-hidden="true">{feature.glyph}</span>
        <div><div className="detail-kicker">{feature.tag[lang]}</div><h2>{feature.title[lang]}</h2><p>{feature.how[0]?.[lang]}</p></div>
        <span className={`feature-status ${evolving(feature) ? 'evolving' : 'published'}`}>{evolving(feature) ? ui.evolving : ui.live}</span>
      </header>
      <div className="feature-sheet-grid">
        <section className="feature-composition">
          <h3>{ui.comp}</h3>
          {feature.flow.length === 0 ? <p className="empty-note">{ui.tbd}</p> : <div className="feature-node-stack">
            {feature.flow.map((node, index) => <div key={node.key}>
              <div className={`feature-node ${familyClass(node.family)}`}>
                <div className="feature-node-top"><span className="family">{node.family}</span>{node.planned && <em className="planned-marker">{ui.planned}</em>}</div>
                <b>{localized(node.name, lang)}</b>
                {node.note && <small>{localized(node.note, lang)}</small>}
                {node.sub?.length ? <div className="feature-node-subs">{node.sub.map((chip, chipIndex) => <span className={chip.kind ? `sub-${chip.kind}` : 'sub-family'} key={`${node.key}-${chipIndex}`}>{chip.kind === 'ext' ? '⇩ ' : chip.kind === 'opt' ? '◇ ' : ''}{localized(chip.name, lang)}</span>)}</div> : null}
              </div>
              {index < feature.flow.length - 1 && <div className="feature-relation">▾ {localized(feature.relations[index] || { pt: 'conecta', en: 'connects' }, lang)}</div>}
            </div>)}
          </div>}
        </section>
        <section className="feature-explanation">
          <h3>{ui.how}</h3>
          {feature.how.map((line) => <p key={line[lang]}>{line[lang]}</p>)}
          <h3>{ui.origin}</h3>
          <dl>{feature.origin.map(([key, value]) => <div className="feature-origin-row" key={key[lang]}><dt>{key[lang]}</dt><dd>{value[lang]}</dd></div>)}</dl>
          <h3>{ui.rules}</h3>
          <ul className="feature-rules">{feature.rules.map((rule) => <li key={rule[lang]}>⛨ {rule[lang]}</li>)}</ul>
        </section>
      </div>
      {selected === 'treasure' && <details className="jewel-disclosure"><summary>{lang === 'pt' ? 'Joias/Jewels' : 'Jewels'}</summary><p>{lang === 'pt' ? <><code>index</code> descobre e propõe joias a partir dos documentos completos; <code>mine</code> é a curadoria humana.</> : <><code>index</code> discovers and proposes jewels from full documents; <code>mine</code> is human curation.</>}</p><div className="wchips">{(lang === 'pt' ? ['documentos', 'source cards', 'evidence packs', 'proposta', 'aceita', 'verificada', 'joias/jewels'] : ['documents', 'source cards', 'evidence packs', 'proposed', 'accepted', 'verified', 'jewels']).map((chip) => <span className="wchip" key={chip}>◆ {chip}</span>)}</div></details>}
    </article>
  </div>;
}
