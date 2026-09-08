export type MaxUser = {
  id?: number | string;
  user_id?: string | number;
  name?: string;
  first_name?: string;
  last_name?: string;
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

function userFromInitData(initData: string | undefined): MaxUser | null {
  if (!initData) {
    return null;
  }
  try {
    const params = new URLSearchParams(initData);
    const raw = params.get("user");
    if (!raw) {
      return null;
    }
    return JSON.parse(raw) as MaxUser;
  } catch {
    return null;
  }
}

export function getMaxUser(): MaxUser | null {
  const app = getMaxWebApp();
  return app?.initDataUnsafe?.user ?? userFromInitData(app?.initData) ?? null;
}

function userFields(user: MaxUser | null): {
  maxUserId: string;
  name: string;
  maxUsername: string;
} {
  const maxUserId = String(user?.id ?? user?.user_id ?? "").trim();
  const fromParts = [user?.first_name, user?.last_name]
    .map((part) => (part ?? "").trim())
    .filter(Boolean)
    .join(" ");
  const name = (user?.name ?? fromParts).trim();
  const maxUsername = (user?.username ?? "").trim();
  return { maxUserId, name, maxUsername };
}

function sendOpen(maxUserId: string, maxUsername: string, name: string): void {
  void fetch("/api/miniapp/open", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ maxUserId, maxUsername, name }),
  }).catch(() => {});
}

export function logMiniappOpen(): void {
  const send = (): boolean => {
    const app = getMaxWebApp();
    app?.ready?.();
    const fields = userFields(getMaxUser());
    if (!fields.maxUserId && !fields.name && !fields.maxUsername) {
      return false;
    }
    sendOpen(fields.maxUserId, fields.maxUsername, fields.name);
    return true;
  };

  if (send()) {
    return;
  }

  let attempts = 0;
  const timer = window.setInterval(() => {
    attempts += 1;
    if (send() || attempts >= 25) {
      window.clearInterval(timer);
    }
  }, 200);
}
