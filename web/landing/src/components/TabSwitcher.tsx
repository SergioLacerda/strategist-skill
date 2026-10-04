import { useEffect, useState } from 'react';

export const TABS = ['overview', 'features', 'structure', 'mission', 'invoke', 'allies'] as const;
export type Tab = (typeof TABS)[number];

const NAV_PT: Record<Tab, string> = {
  overview: 'Visão Geral', features: 'Funcionalidades', structure: 'Componentes', mission: 'Missão', invoke: 'Instalação', allies: 'Aliados',
};
const NAV_EN: Record<Tab, string> = {
  overview: 'Overview', features: 'Features', structure: 'Components', mission: 'Mission', invoke: 'Installation', allies: 'Allies',
};

function read(k: string, fallback: string): string { try { return localStorage.getItem(k) || fallback; } catch { return fallback; } }
function save(k: string, value: string): void { try { localStorage.setItem(k, value); } catch { /* storage is optional */ } }
function isTab(value: string | null): value is Tab { return value !== null && (TABS as readonly string[]).includes(value); }
function initialTab(): Tab {
  if (typeof window !== 'undefined' && window.location.hash === '#allies') return 'allies';
  const stored = read('strategist_console_tab', 'overview');
  return isTab(stored) ? stored : 'overview';
}

export default function TabSwitcher() {
  const [tab, setTab] = useState<Tab>(initialTab);

  useEffect(() => {
    save('strategist_console_tab', tab);
    document.querySelectorAll<HTMLElement>('.panel').forEach((element) => element.classList.toggle('active', element.dataset.panel === tab));
    if (tab === 'allies' && window.location.hash !== '#allies') history.replaceState(null, '', `${window.location.pathname}${window.location.search}#allies`);
    if (tab !== 'allies' && window.location.hash === '#allies') history.replaceState(null, '', window.location.pathname + window.location.search);
    window.scrollTo({ top: 0 });
  }, [tab]);

  useEffect(() => {
    const onSwitch = (event: Event) => { const value = (event as CustomEvent<string>).detail; if (isTab(value)) setTab(value); };
    window.addEventListener('strategist:tab', onSwitch);
    return () => window.removeEventListener('strategist:tab', onSwitch);
  }, []);

  return (
    <nav className="nav" aria-label="Console tabs">
      {TABS.map((value) => <button key={value} className={value === tab ? 'active' : ''} onClick={() => setTab(value)} aria-current={value === tab ? 'page' : undefined} data-pt={NAV_PT[value]} data-en={NAV_EN[value]}>{NAV_PT[value]}</button>)}
    </nav>
  );
}
