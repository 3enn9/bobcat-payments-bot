export type RogatkaRequest = {
  id: number;
  maxChatId: number;
  maxUserId: number;
  maxUsername: string;
  maxUserName: string;
  maxMessageId: string;
  message: string;
  driverName: string | null;
  createdAt: string;
};

type ListRogatkaResponse = {
  success: boolean;
  requests?: RogatkaRequest[];
  error?: string;
};

type AssignDriverResponse = {
  success: boolean;
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

export async function assignRogatkaDriver(
  id: number,
  driverName: string,
): Promise<void> {
  const response = await fetch(`/api/miniapp/rogatka-requests/${id}/assign`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ driverName }),
  });

  const data = (await response.json()) as AssignDriverResponse;

  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось назначить водителя");
  }
}
