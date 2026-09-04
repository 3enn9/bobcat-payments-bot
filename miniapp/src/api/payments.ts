export type MatchFirm = {
  id: number;
  name: string;
  inn: string;
  unmatchedCount: number;
  unpaidInvoiceCount: number;
};

export type MatchPayment = {
  id: number;
  source: string;
  executedAt: string;
  amount: number;
  remainingAmount: number;
  payerName: string;
  payerInn: string;
  purpose: string;
  account: string;
};

export type MatchInvoice = {
  id: number;
  number: number;
  invoiceDate: string;
  buyerName: string;
  buyerInn: string;
  total: number;
  paidAmount: number;
  remainingAmount: number;
};

type FirmsResponse = {
  success: boolean;
  items?: MatchFirm[];
  error?: string;
};

type MatchDataResponse = {
  success: boolean;
  payments?: MatchPayment[];
  invoices?: MatchInvoice[];
  error?: string;
};

type MatchResponse = {
  success: boolean;
  error?: string;
};

export async function listMatchFirms(): Promise<MatchFirm[]> {
  const response = await fetch("/api/miniapp/payments/match/firms");
  const data = (await response.json()) as FirmsResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить фирмы");
  }
  return data.items ?? [];
}

export async function listMatchData(
  supplierId: number,
  paymentId?: number | null,
): Promise<{
  payments: MatchPayment[];
  invoices: MatchInvoice[];
}> {
  const params = new URLSearchParams({ supplierId: String(supplierId) });
  if (paymentId) {
    params.set("paymentId", String(paymentId));
  }
  const response = await fetch(`/api/miniapp/payments/match?${params}`);
  const data = (await response.json()) as MatchDataResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось загрузить данные");
  }
  return {
    payments: data.payments ?? [],
    invoices: data.invoices ?? [],
  };
}

export async function matchPayment(
  paymentId: number,
  invoiceIds: number[],
): Promise<void> {
  const response = await fetch("/api/miniapp/payments/match", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ paymentId, invoiceIds }),
  });
  const data = (await response.json()) as MatchResponse;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Не удалось сопоставить");
  }
}
