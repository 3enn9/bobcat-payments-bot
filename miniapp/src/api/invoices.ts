export type InvoiceParty = {
  id?: number | null;
  name: string;
  inn: string;
  kpp: string;
  addressText: string;
};

export type InvoiceBank = {
  id?: number | null;
  name: string;
  bik: string;
  account: string;
  corrAccount: string;
};

export type InvoiceSupplier = InvoiceParty & {
  bankId?: number | null;
  lastInvoiceNumber?: number;
  bank?: InvoiceBank | null;
};

export type InvoiceItem = {
  position: number;
  title: string;
  quantity: number;
  unit: string;
  price: number;
  amount: number;
};

type ListResponse<T> = {
  success: boolean;
  items?: T[];
  error?: string;
};

type CreateInvoiceResponse = {
  success: boolean;
  id?: number;
  number?: number;
  error?: string;
};

async function search<T>(path: string, q: string): Promise<T[]> {
  const params = new URLSearchParams();
  if (q) {
    params.set("q", q);
  }
  const response = await fetch(`${path}?${params.toString()}`);
  const data = (await response.json()) as ListResponse<T>;
  if (!response.ok || !data.success) {
    throw new Error(data.error || "Ошибка поиска");
  }
  return data.items ?? [];
}

export function searchSuppliers(q: string) {
  return search<InvoiceSupplier>("/api/miniapp/invoices/suppliers", q);
}

export function searchBuyers(q: string) {
  return search<InvoiceParty>("/api/miniapp/invoices/buyers", q);
}

export function searchBanks(q: string) {
  return search<InvoiceBank>("/api/miniapp/invoices/banks", q);
}

export async function createInvoice(payload: {
  invoiceDate: string;
  basis: string;
  supplier: InvoiceParty;
  buyer: InvoiceParty;
  bank: InvoiceBank;
  items: InvoiceItem[];
}): Promise<{ id: number; number: number }> {
  const response = await fetch("/api/miniapp/invoices", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data = (await response.json()) as CreateInvoiceResponse;
  if (!response.ok || !data.success || !data.id || !data.number) {
    throw new Error(data.error || "Не удалось создать счёт");
  }
  return { id: data.id, number: data.number };
}
