export type DaysOffItem = {
  id: number;
  workerName: string;
  dateFrom: string;
  dateTo: string;
  comment: string;
  status: "pending" | "approved" | "rejected";
  decidedByName?: string | null;
  decidedAt?: string | null;
  createdAt: string;
};

type ListResponse = {
  success: boolean;
  items?: DaysOffItem[];
  error?: string;
};

type CreateResponse = {
  success: boolean;
  id?: number;
  error?: string;
};

export async function fetchDaysOff(workerName: string): Promise<DaysOffItem[]> {
  const params = new URLSearchParams({ worker: workerName });
  const response = await fetch(`/api/miniapp/days-off?${params.toString()}`);
  const data = (await response.json()) as ListResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить выходные");
  }
  return data.items ?? [];
}

export async function createDaysOff(payload: {
  workerName: string;
  dateFrom: string;
  dateTo: string;
  comment: string;
}): Promise<number> {
  const response = await fetch("/api/miniapp/days-off", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data = (await response.json()) as CreateResponse;
  if (!response.ok || !data.success || !data.id) {
    throw new Error(data.error || "Не удалось отправить заявку");
  }
  return data.id;
}

export function statusLabel(status: DaysOffItem["status"]): string {
  switch (status) {
    case "approved":
      return "Подтверждено";
    case "rejected":
      return "Отклонено";
    default:
      return "Ожидает";
  }
}

export function formatPeriod(dateFrom: string, dateTo: string): string {
  const fmt = (iso: string) => {
    const [y, m, d] = iso.split("-");
    return `${d}.${m}.${y}`;
  };
  if (dateFrom === dateTo) {
    return fmt(dateFrom);
  }
  return `${fmt(dateFrom)} – ${fmt(dateTo)}`;
}
