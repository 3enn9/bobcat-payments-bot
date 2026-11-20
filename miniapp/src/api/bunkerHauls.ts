export type BunkerHaulMessage = {
  id: number;
  sentAt: string;
  text: string;
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
