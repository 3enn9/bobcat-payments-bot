export type CashEntryType = "income" | "expense";

export type CashEntry = {
  id: number;
  workerName: string;
  entryType: CashEntryType;
  amount: number;
  description: string;
  entryDate: string;
  createdAt: string;
};

type ListCashResponse = {
  success: boolean;
  balance?: number;
  entries?: CashEntry[];
  error?: string;
};

type CreateCashResponse = {
  success: boolean;
  id?: number;
  balance?: number;
  error?: string;
};

export async function fetchWorkerCash(workerName: string): Promise<{
  balance: number;
  entries: CashEntry[];
}> {
  const params = new URLSearchParams({ worker: workerName });
  const response = await fetch(`/api/miniapp/cash?${params.toString()}`);
  const data = (await response.json()) as ListCashResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить кассу");
  }
  return {
    balance: data.balance ?? 0,
    entries: data.entries ?? [],
  };
}

export async function createCashEntry(payload: {
  workerName: string;
  entryType: CashEntryType;
  amount: number;
  description: string;
  entryDate: string;
}): Promise<{ id: number; balance: number }> {
  const response = await fetch("/api/miniapp/cash", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data = (await response.json()) as CreateCashResponse;
  if (!response.ok || !data.success || data.id === undefined) {
    throw new Error(data.error || "Не удалось сохранить запись");
  }
  return {
    id: data.id,
    balance: data.balance ?? 0,
  };
}
