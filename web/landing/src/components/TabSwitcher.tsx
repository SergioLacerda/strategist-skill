import { useEffect, useState } from 'react';

const TABS = ['overview', 'roles', 'skills', 'mission', 'invoke'] as const;
type Tab = typeof TABS[number];

const NAV_PT: Record<Tab, string> = {
  overview: 'Visão Geral', roles: 'Papéis', skills: 'Habilidades',
  mission: 'Fluxo da Missão', invoke: 'Invocação',
};
const NAV_EN: Record<Tab, string> = {
  overview: 'Overview', roles: 'Roles', skills: 'Skills',
  mission: 'Mission Flow', invoke: 'Invoke',
};

function read(k: string, d: string) {
  try { return localStorage.getItem(k) || d; } catch { return d; }
}
function save(k: string, v: string) {
  try { localStorage.setItem(k, v); } catch {}
}

export default function TabSwitcher() {
  const [tab, setTab] = useState<Tab>(() => (read('strategist_console_tab', 'overview') as Tab));

  useEffect(() => {
    save('strategist_console_tab', tab);
    document.querySelectorAll<HTMLElement>('.panel').forEach(el => {
      el.classList.toggle('active', el.dataset.panel === tab);
    });
    window.scrollTo({ top: 0 });
  }, [tab]);

  // Listen for external tab switches (e.g. from tile clicks)
  useEffect(() => {
    function onSwitch(e: Event) {
      const t = (e as CustomEvent<Tab>).detail;
      if (TABS.includes(t)) setTab(t);
    }
    window.addEventListener('strategist:tab', onSwitch);
    return () => window.removeEventListener('strategist:tab', onSwitch);
  }, []);

  // Activate initial tab on mount
  useEffect(() => {
    document.querySelectorAll<HTMLElement>('.panel').forEach(el => {
      el.classList.toggle('active', el.dataset.panel === tab);
    });
  }, []);

  return (
    <nav className="nav" aria-label="Console tabs">
      {TABS.map(t => (
        <button
          key={t}
          className={t === tab ? 'active' : ''}
          onClick={() => setTab(t)}
          aria-current={t === tab ? 'page' : undefined}
          data-pt={NAV_PT[t]}
          data-en={NAV_EN[t]}
        >{NAV_PT[t]}</button>
      ))}
    </nav>
  );
}
