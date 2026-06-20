import { useEffect, useState } from 'react';

function read(k: string, d: string) {
  try { return localStorage.getItem(k) || d; } catch { return d; }
}
function save(k: string, v: string) {
  try { localStorage.setItem(k, v); } catch {}
}

export default function LangToggle() {
  const [lang, setLang] = useState(() => read('strategist_console_lang', 'pt'));

  useEffect(() => {
    save('strategist_console_lang', lang);
    document.documentElement.lang = lang === 'en' ? 'en' : 'pt-BR';
    // Swap visible text in data-pt / data-en elements
    document.querySelectorAll<HTMLElement>('[data-pt]').forEach(el => {
      const text = lang === 'en' ? el.dataset.en : el.dataset.pt;
      if (text !== undefined) el.innerHTML = text;
    });
    window.dispatchEvent(new CustomEvent('strategist:lang', { detail: lang }));
  }, [lang]);

  return (
    <div className="langbar">
      <button
        className={lang === 'pt' ? 'active' : ''}
        onClick={() => setLang('pt')}
        aria-pressed={lang === 'pt'}
      >🇧🇷 PT</button>
      <button
        className={lang === 'en' ? 'active' : ''}
        onClick={() => setLang('en')}
        aria-pressed={lang === 'en'}
      >🇺🇸 EN</button>
    </div>
  );
}
