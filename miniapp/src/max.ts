export type MaxUser = {
  user_id: string;
  name?: string;
  username?: string;
  avatar_url?: string;
};

export type MaxWebApp = {
  initData?: string;
  initDataUnsafe?: {
    user?: MaxUser;
    start_param?: string;
  };
  ready?: () => void;
  close?: () => void;
};

declare global {
  interface Window {
    WebApp?: MaxWebApp;
  }
}

export function getMaxWebApp(): MaxWebApp | null {
  if (typeof window === "undefined") {
    return null;
  }

  return window.WebApp ?? null;
}

export function getMaxUser(): MaxUser | null {
  return getMaxWebApp()?.initDataUnsafe?.user ?? null;
}