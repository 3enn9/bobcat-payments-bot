export type RogatkaRequest = {
  id: number;
  maxChatId: number;
  maxUserId: number;
  maxUsername: string;
  maxUserName: string;
  maxMessageId: string;
  message: string;
  driverName: string | null;
  isCompleted: boolean;
  driverComment: string | null;
  createdAt: string;
};

export type RogatkaListStatus = "active" | "assigned";

type ListRogatkaResponse = {
  success: boolean;
  requests?: RogatkaRequest[];
  error?: string;
};

type ActionResponse = {
  success: boolean;
  error?: string;
};

export async function fetchRogatkaRequests(
  status: RogatkaListStatus = "active",
  driver?: string,
): Promise<RogatkaRequest[]> {
  const params = new URLSearchParams();
  if (driver) {
    params.set("driver", driver);
  } else {
    params.set("status", status);
  }

  const response = await fetch(
    `/api/miniapp/rogatka-requests?${params.toString()}`,
  );
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

  const data = (await response.json()) as ActionResponse;

  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось назначить водителя");
  }
}

export async function deleteRogatkaRequest(id: number): Promise<void> {
  const response = await fetch(`/api/miniapp/rogatka-requests/${id}`, {
    method: "DELETE",
  });

  const data = (await response.json()) as ActionResponse;

  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось удалить заявку");
  }
}

export async function completeRogatkaRequest(
  id: number,
  driverName: string,
  comment: string,
  photos: File[],
): Promise<void> {
  const formData = new FormData();
  formData.append("driverName", driverName);
  formData.append("comment", comment);
  for (const photo of photos) {
    formData.append("photos", photo);
  }

  const response = await fetch(`/api/miniapp/rogatka-requests/${id}/complete`, {
    method: "POST",
    body: formData,
  });

  const data = (await response.json()) as ActionResponse;

  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось отправить заявку");
  }
}
