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

export function logMiniappOpen(): void {
  const app = getMaxWebApp();
  app?.ready?.();
  const user = getMaxUser();
  const maxUserId = String(user?.user_id ?? "").trim();
  const name = (user?.name ?? "").trim();
  const maxUsername = (user?.username ?? "").trim();
  if (!maxUserId && !name && !maxUsername) {
    return;
  }
  void fetch("/api/miniapp/open", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ maxUserId, maxUsername, name }),
  }).catch(() => {});
}