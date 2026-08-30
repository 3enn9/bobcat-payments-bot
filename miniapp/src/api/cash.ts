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
  total?: number;
  limit?: number;
  error?: string;
};

type MutateCashResponse = {
  success: boolean;
  id?: number;
  balance?: number;
  error?: string;
};

export const CASH_HISTORY_LIMIT = 10;

export async function fetchWorkerCash(workerName: string): Promise<{
  balance: number;
  entries: CashEntry[];
  total: number;
}> {
  const params = new URLSearchParams({
    worker: workerName,
    limit: String(CASH_HISTORY_LIMIT),
  });
  const response = await fetch(`/api/miniapp/cash?${params.toString()}`);
  const data = (await response.json()) as ListCashResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить кассу");
  }
  return {
    balance: data.balance ?? 0,
    entries: data.entries ?? [],
    total: data.total ?? 0,
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
  const data = (await response.json()) as MutateCashResponse;
  if (!response.ok || !data.success || data.id === undefined) {
    throw new Error(data.error || "Не удалось сохранить запись");
  }
  return {
    id: data.id,
    balance: data.balance ?? 0,
  };
}

export async function updateCashEntry(
  id: number,
  payload: {
    workerName: string;
    entryType: CashEntryType;
    amount: number;
    description: string;
    entryDate: string;
  },
): Promise<{ balance: number }> {
  const response = await fetch(`/api/miniapp/cash/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data = (await response.json()) as MutateCashResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось обновить запись");
  }
  return {
    balance: data.balance ?? 0,
  };
}
