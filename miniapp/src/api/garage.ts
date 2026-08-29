type CreateResponse = {
  success: boolean;
  id?: number;
  error?: string;
};

export async function createGarageWork(payload: {
  workerName: string;
  workDate: string;
  timeFrom: string;
  timeTo: string;
  description: string;
}): Promise<number> {
  const response = await fetch("/api/miniapp/garage-work", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data = (await response.json()) as CreateResponse;
  if (!response.ok || !data.success || !data.id) {
    throw new Error(data.error || "Не удалось сохранить запись");
  }
  return data.id;
}
