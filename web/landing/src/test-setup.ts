import { vi } from 'vitest';

// Node 26 does not expose localStorage without --localstorage-file.
// Provide a simple in-memory implementation for all tests.
const makeLocalStorage = () => {
  let store: Record<string, string> = {};
  return {
    getItem: (k: string) => store[k] ?? null,
    setItem: (k: string, v: string) => { store[k] = String(v); },
    removeItem: (k: string) => { delete store[k]; },
    clear: () => { store = {}; },
    key: (i: number) => Object.keys(store)[i] ?? null,
    get length() { return Object.keys(store).length; },
  };
};

vi.stubGlobal('localStorage', makeLocalStorage());

// jsdom does not implement scrollTo — silence the warning
vi.stubGlobal('scrollTo', vi.fn());
