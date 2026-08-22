export type CreateRequestPayload = {
  name: string;
  contact: string;
  message: string;
};

type CreateRequestResponse = {
  success: boolean;
  id?: number;
  error?: string;
};

export async function createRequest(
  payload: CreateRequestPayload,
): Promise<CreateRequestResponse> {
  const response = await fetch("/api/miniapp/requests", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  const data = (await response.json()) as CreateRequestResponse;

  if (!response.ok) {
    throw new Error(data.error || "Не удалось отправить заявку");
  }

  return data;
}