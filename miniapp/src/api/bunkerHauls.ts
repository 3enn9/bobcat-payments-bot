export type BunkerHaulMessage = {
  id: number;
  sentAt: string;
  text: string;
  annotations: BunkerHaulAnnotation[];
};

export type BunkerHaulAnnotationKind = "vector" | "dubki" | "cash" | "base" | "rsm";

export type BunkerHaulAnnotation = {
  kind: BunkerHaulAnnotationKind;
  amount: number;
};

type BunkerHaulMessagesResponse = {
  success: boolean;
  items?: BunkerHaulMessage[];
  error?: string;
};

type BunkerHaulSurnamesResponse = {
  success: boolean;
  surnames?: string[];
  error?: string;
};

type BunkerHaulReportCommentResponse = {
  success: boolean;
  comments?: string;
  error?: string;
};

export async function suggestBunkerHaulSurnames(query: string): Promise<string[]> {
  const params = new URLSearchParams({ q: query.trim() });
  const response = await fetch(`/api/miniapp/bunker-hauls/surnames?${params}`);
  const data = (await response.json()) as BunkerHaulSurnamesResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить фамилии");
  }
  return data.surnames ?? [];
}

export async function listBunkerHaulMessages(
  surname: string,
  date: string,
): Promise<BunkerHaulMessage[]> {
  const params = new URLSearchParams({ surname: surname.trim(), date });
  const response = await fetch(`/api/miniapp/bunker-hauls?${params}`);
  const data = (await response.json()) as BunkerHaulMessagesResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить сообщения");
  }
  return data.items ?? [];
}

export async function getBunkerHaulReportComment(
  surname: string,
  date: string,
): Promise<string> {
  const params = new URLSearchParams({ surname: surname.trim(), date });
  const response = await fetch(`/api/miniapp/bunker-hauls/comment?${params}`);
  const data = (await response.json()) as BunkerHaulReportCommentResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить комментарий");
  }
  return data.comments ?? "";
}

export async function saveBunkerHaulReportComment(
  surname: string,
  date: string,
  comments: string,
): Promise<void> {
  const response = await fetch("/api/miniapp/bunker-hauls/comment", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ surname: surname.trim(), date, comments }),
  });
  const data = (await response.json()) as { success: boolean; error?: string };
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось сохранить комментарий");
  }
}

export async function saveBunkerHaulAnnotation(
  id: number,
  surname: string,
  date: string,
  annotations: BunkerHaulAnnotation[],
): Promise<void> {
  const response = await fetch(`/api/miniapp/bunker-hauls/${id}/annotation`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ surname: surname.trim(), date, annotations }),
  });
  const data = (await response.json()) as { success: boolean; error?: string };
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось сохранить отметку");
  }
}

export async function sendBunkerHaulReport(
  surname: string,
  date: string,
  comments: string,
  cashEntryIds: number[] = [],
): Promise<void> {
  const response = await fetch("/api/miniapp/bunker-hauls/send", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ surname: surname.trim(), date, comments, cashEntryIds }),
  });
  const data = (await response.json()) as { success: boolean; error?: string };
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось отправить отчёт");
  }
}
