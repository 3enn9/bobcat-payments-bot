export type InvoiceParty = {
  id?: number | null;
  name: string;
  inn: string;
  kpp: string;
  addressText: string;
};

export type InvoiceBank = {
  id?: number | null;
  supplierId?: number | null;
  name: string;
  bik: string;
  account: string;
  corrAccount: string;
};

export type InvoiceSupplier = InvoiceParty & {
  lastInvoiceNumber?: number;
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

async function search<T>(
  path: string,
  q: string,
  extra?: Record<string, string>,
): Promise<T[]> {
  const params = new URLSearchParams();
  if (q) {
    params.set("q", q);
  }
  if (extra) {
    for (const [key, value] of Object.entries(extra)) {
      if (value) {
        params.set(key, value);
      }
    }
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

export function searchBanks(q: string, supplierId?: number | null) {
  if (!supplierId) {
    return Promise.resolve([] as InvoiceBank[]);
  }
  return search<InvoiceBank>("/api/miniapp/invoices/banks", q, {
    supplierId: String(supplierId),
  });
}

export async function createInvoice(payload: {
  number: number;
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
