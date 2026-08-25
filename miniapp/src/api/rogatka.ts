export type RogatkaRequest = {
  id: number;
  maxChatId: number;
  maxUserId: number;
  maxUsername: string;
  maxUserName: string;
  maxMessageId: string;
  message: string;
  createdAt: string;
};

type ListRogatkaResponse = {
  success: boolean;
  requests?: RogatkaRequest[];
  error?: string;
};

export async function fetchRogatkaRequests(): Promise<RogatkaRequest[]> {
  const response = await fetch("/api/miniapp/rogatka-requests");
  const data = (await response.json()) as ListRogatkaResponse;

  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить заявки");
  }

  return data.requests ?? [];
}
