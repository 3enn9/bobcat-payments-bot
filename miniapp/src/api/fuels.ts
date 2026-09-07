export type FuelEntry = {
  id: number;
  fueledAt: string;
  equipmentNumber: string;
  amount: number;
  holder: string;
};

type ListResponse = {
  success: boolean;
  items?: FuelEntry[];
  error?: string;
};

type MutateResponse = {
  success: boolean;
  error?: string;
};

export async function listFuels(holder: string): Promise<FuelEntry[]> {
  const params = new URLSearchParams({ holder: holder.trim() });
  const response = await fetch(`/api/miniapp/fuels?${params}`);
  const data = (await response.json()) as ListResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить заправки");
  }
  return data.items ?? [];
}

export async function updateFuelEquipment(
  id: number,
  equipmentNumber: string,
): Promise<void> {
  const response = await fetch(`/api/miniapp/fuels/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ equipmentNumber }),
  });
  const data = (await response.json()) as MutateResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось сохранить номер");
  }
}
