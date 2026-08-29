export type EquipmentItem = {
  id: number;
  number: string;
};

type ListResponse = {
  success: boolean;
  items?: EquipmentItem[];
  error?: string;
};

export async function fetchEquipment(): Promise<EquipmentItem[]> {
  const response = await fetch("/api/miniapp/equipment");
  const data = (await response.json()) as ListResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить технику");
  }
  return data.items ?? [];
}
