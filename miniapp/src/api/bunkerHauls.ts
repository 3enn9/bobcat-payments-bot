export type BunkerHaulMessage = {
  id: number;
  sentAt: string;
  text: string;
  annotations: BunkerHaulAnnotation[];
};

export type BunkerHaulAnnotationKind = "vector" | "dubki" | "cash";

export type BunkerHaulAnnotation = {
  kind: BunkerHaulAnnotationKind;
  amount: number;
};

type BunkerHaulMessagesResponse = {
  success: boolean;
  items?: BunkerHaulMessage[];
  error?: string;
};

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
): Promise<void> {
  const response = await fetch("/api/miniapp/bunker-hauls/send", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ surname: surname.trim(), date, comments }),
  });
  const data = (await response.json()) as { success: boolean; error?: string };
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось отправить отчёт");
  }
}
