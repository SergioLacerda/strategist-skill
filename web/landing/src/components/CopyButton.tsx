import { useState, useEffect } from 'react';

export default function CopyButtons() {
  const [copied, setCopied] = useState<string | null>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      const btn = (e.target as HTMLElement).closest<HTMLButtonElement>('.copy-btn');
      if (!btn) return;
      const cmd = btn.dataset.command ?? '';
      navigator.clipboard.writeText(cmd).then(() => {
        setCopied(cmd);
        setTimeout(() => setCopied(null), 1800);
      }).catch(() => {});
    }
    document.addEventListener('click', handleClick);
    return () => document.removeEventListener('click', handleClick);
  }, []);

  if (!copied) return null;
  return (
    <div
      aria-live="polite"
      style={{
        position: 'fixed', bottom: '24px', right: '24px', zIndex: 999,
        fontFamily: 'var(--font-mono)', fontSize: '12px',
        background: 'rgba(20,15,9,.92)', border: '1px solid var(--line-2)',
        color: 'var(--green)', padding: '8px 14px', borderRadius: 'var(--radius-sm)',
        pointerEvents: 'none',
      }}
    >
      ✓ copied
    </div>
  );
}
