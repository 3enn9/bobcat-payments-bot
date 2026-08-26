import { useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import {
  createInvoice,
  searchBanks,
  searchBuyers,
  searchSuppliers,
  type InvoiceBank,
  type InvoiceItem,
  type InvoiceParty,
  type InvoiceSupplier,
} from "../api/invoices";

type PartyDraft = InvoiceParty & { id: number | null };
type BankDraft = InvoiceBank & { id: number | null };

type LineDraft = {
  key: string;
  title: string;
  quantity: string;
  unit: string;
  price: string;
};

function todayISO(): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function emptyParty(): PartyDraft {
  return { id: null, name: "", inn: "", kpp: "", addressText: "" };
}

function emptyBank(): BankDraft {
  return { id: null, name: "", bik: "", account: "", corrAccount: "" };
}

function newLine(): LineDraft {
  return {
    key: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
    title: "",
    quantity: "1",
    unit: "ч",
    price: "",
  };
}

function parseNum(value: string): number {
  const normalized = value.replace(",", ".").trim();
  const n = Number(normalized);
  return Number.isFinite(n) ? n : 0;
}

function money(value: number): string {
  return value.toFixed(2).replace(".", ",");
}

type SuggestKind = "suppliers" | "buyers" | "banks";

type SuggestProps = {
  label: string;
  value: string;
  kind: SuggestKind;
  onChange: (value: string) => void;
  onPick: (item: InvoiceSupplier | InvoiceParty | InvoiceBank) => void;
  renderItem: (item: InvoiceSupplier | InvoiceParty | InvoiceBank) => string;
  disabled?: boolean;
};

function AutocompleteField({
  label,
  value,
  kind,
  onChange,
  onPick,
  renderItem,
  disabled,
}: SuggestProps) {
  const [items, setItems] = useState<
    Array<InvoiceSupplier | InvoiceParty | InvoiceBank>
  >([]);
  const [open, setOpen] = useState(false);
  const timer = useRef<number | null>(null);

  useEffect(() => {
    if (timer.current) {
      window.clearTimeout(timer.current);
    }
    if (value.trim().length < 2) {
      setItems([]);
      setOpen(false);
      return;
    }

    timer.current = window.setTimeout(() => {
      const loader =
        kind === "suppliers"
          ? searchSuppliers
          : kind === "buyers"
            ? searchBuyers
            : searchBanks;

      void loader(value.trim())
        .then((result) => {
          setItems(result);
          setOpen(result.length > 0);
        })
        .catch(() => {
          setItems([]);
          setOpen(false);
        });
    }, 250);

    return () => {
      if (timer.current) {
        window.clearTimeout(timer.current);
      }
    };
  }, [value, kind]);

  return (
    <label className="invoice-field">
      <span>{label}</span>
      <div className="autocomplete">
        <input
          value={value}
          disabled={disabled}
          onChange={(event) => onChange(event.target.value)}
          onFocus={() => {
            if (items.length > 0) {
              setOpen(true);
            }
          }}
          onBlur={() => {
            window.setTimeout(() => setOpen(false), 150);
          }}
        />
        {open && (
          <ul className="autocomplete-list">
            {items.map((item, index) => (
              <li key={index}>
                <button
                  type="button"
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => {
                    onPick(item);
                    setOpen(false);
                  }}
                >
                  {renderItem(item)}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </label>
  );
}

export default function InvoiceForm() {
  const [supplier, setSupplier] = useState<PartyDraft>(emptyParty);
  const [buyer, setBuyer] = useState<PartyDraft>(emptyParty);
  const [bank, setBank] = useState<BankDraft>(emptyBank);
  const [invoiceDate, setInvoiceDate] = useState(todayISO);
  const [basis, setBasis] = useState("");
  const [lines, setLines] = useState<LineDraft[]>([newLine()]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const totals = useMemo(() => {
    const items = lines.map((line, index) => {
      const quantity = parseNum(line.quantity);
      const price = parseNum(line.price);
      const amount = Math.round(quantity * price * 100) / 100;
      return {
        position: index + 1,
        title: line.title.trim(),
        quantity,
        unit: line.unit.trim(),
        price,
        amount,
      } satisfies InvoiceItem;
    });
    const total = Math.round(items.reduce((sum, item) => sum + item.amount, 0) * 100) / 100;
    const vat = Math.round(((total * 22) / 122) * 100) / 100;
    return { items, total, vat };
  }, [lines]);

  function updateLine(key: string, patch: Partial<LineDraft>) {
    setLines((prev) =>
      prev.map((line) => (line.key === key ? { ...line, ...patch } : line)),
    );
  }

  function pickSupplier(item: InvoiceSupplier) {
    setSupplier({
      id: item.id ?? null,
      name: item.name,
      inn: item.inn,
      kpp: item.kpp,
      addressText: item.addressText,
    });
    if (item.bank) {
      setBank({
        id: item.bank.id ?? null,
        name: item.bank.name,
        bik: item.bank.bik,
        account: item.bank.account,
        corrAccount: item.bank.corrAccount,
      });
    }
  }

  function pickBuyer(item: InvoiceParty) {
    setBuyer({
      id: item.id ?? null,
      name: item.name,
      inn: item.inn,
      kpp: item.kpp,
      addressText: item.addressText,
    });
  }

  function pickBank(item: InvoiceBank) {
    setBank({
      id: item.id ?? null,
      name: item.name,
      bik: item.bik,
      account: item.account,
      corrAccount: item.corrAccount,
    });
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSuccess("");

    if (!supplier.name.trim() || !supplier.inn.trim() || !supplier.addressText.trim()) {
      setError("Заполните поставщика");
      return;
    }
    if (!buyer.name.trim() || !buyer.inn.trim() || !buyer.addressText.trim()) {
      setError("Заполните покупателя");
      return;
    }
    if (!bank.name.trim() || !bank.bik.trim() || !bank.account.trim() || !bank.corrAccount.trim()) {
      setError("Заполните банк");
      return;
    }
    if (totals.items.some((item) => !item.title || !item.unit || item.quantity <= 0)) {
      setError("Проверьте позиции счёта");
      return;
    }

    setSaving(true);
    try {
      const result = await createInvoice({
        invoiceDate,
        basis: basis.trim(),
        supplier,
        buyer,
        bank,
        items: totals.items,
      });
      setSuccess(`Счёт № ${result.number} создан и отправлен`);
      setLines([newLine()]);
      setBasis("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка создания счёта");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form className="invoice-form" onSubmit={submit}>
      <section className="invoice-section">
        <h2>Поставщик</h2>
        <AutocompleteField
          label="Наименование"
          kind="suppliers"
          value={supplier.name}
          disabled={saving}
          onChange={(name) => setSupplier((prev) => ({ ...prev, id: null, name }))}
          onPick={(item) => pickSupplier(item as InvoiceSupplier)}
          renderItem={(item) =>
            `${(item as InvoiceSupplier).name} · ИНН ${(item as InvoiceSupplier).inn}`
          }
        />
        <div className="invoice-grid-2">
          <label className="invoice-field">
            <span>ИНН</span>
            <input
              value={supplier.inn}
              disabled={saving}
              onChange={(event) =>
                setSupplier((prev) => ({ ...prev, inn: event.target.value }))
              }
            />
          </label>
          <label className="invoice-field">
            <span>КПП</span>
            <input
              value={supplier.kpp}
              disabled={saving}
              onChange={(event) =>
                setSupplier((prev) => ({ ...prev, kpp: event.target.value }))
              }
            />
          </label>
        </div>
        <label className="invoice-field">
          <span>Индекс и адрес</span>
          <textarea
            rows={2}
            value={supplier.addressText}
            disabled={saving}
            onChange={(event) =>
              setSupplier((prev) => ({
                ...prev,
                addressText: event.target.value,
              }))
            }
          />
        </label>
      </section>

      <section className="invoice-section">
        <h2>Покупатель</h2>
        <AutocompleteField
          label="Наименование"
          kind="buyers"
          value={buyer.name}
          disabled={saving}
          onChange={(name) => setBuyer((prev) => ({ ...prev, id: null, name }))}
          onPick={(item) => pickBuyer(item as InvoiceParty)}
          renderItem={(item) =>
            `${(item as InvoiceParty).name} · ИНН ${(item as InvoiceParty).inn}`
          }
        />
        <div className="invoice-grid-2">
          <label className="invoice-field">
            <span>ИНН</span>
            <input
              value={buyer.inn}
              disabled={saving}
              onChange={(event) =>
                setBuyer((prev) => ({ ...prev, inn: event.target.value }))
              }
            />
          </label>
          <label className="invoice-field">
            <span>КПП</span>
            <input
              value={buyer.kpp}
              disabled={saving}
              onChange={(event) =>
                setBuyer((prev) => ({ ...prev, kpp: event.target.value }))
              }
            />
          </label>
        </div>
        <label className="invoice-field">
          <span>Индекс и адрес</span>
          <textarea
            rows={2}
            value={buyer.addressText}
            disabled={saving}
            onChange={(event) =>
              setBuyer((prev) => ({
                ...prev,
                addressText: event.target.value,
              }))
            }
          />
        </label>
      </section>

      <section className="invoice-section">
        <h2>Банк</h2>
        <AutocompleteField
          label="Наименование банка"
          kind="banks"
          value={bank.name}
          disabled={saving}
          onChange={(name) => setBank((prev) => ({ ...prev, id: null, name }))}
          onPick={(item) => pickBank(item as InvoiceBank)}
          renderItem={(item) =>
            `${(item as InvoiceBank).name} · БИК ${(item as InvoiceBank).bik}`
          }
        />
        <div className="invoice-grid-2">
          <label className="invoice-field">
            <span>БИК</span>
            <input
              value={bank.bik}
              disabled={saving}
              onChange={(event) =>
                setBank((prev) => ({ ...prev, bik: event.target.value }))
              }
            />
          </label>
          <label className="invoice-field">
            <span>Р/с</span>
            <input
              value={bank.account}
              disabled={saving}
              onChange={(event) =>
                setBank((prev) => ({ ...prev, account: event.target.value }))
              }
            />
          </label>
        </div>
        <label className="invoice-field">
          <span>Корр. счёт</span>
          <input
            value={bank.corrAccount}
            disabled={saving}
            onChange={(event) =>
              setBank((prev) => ({
                ...prev,
                corrAccount: event.target.value,
              }))
            }
          />
        </label>
      </section>

      <section className="invoice-section">
        <div className="invoice-grid-2">
          <label className="invoice-field">
            <span>Дата</span>
            <input
              type="date"
              value={invoiceDate}
              disabled={saving}
              onChange={(event) => setInvoiceDate(event.target.value)}
            />
          </label>
          <label className="invoice-field">
            <span>Основание</span>
            <input
              value={basis}
              disabled={saving}
              onChange={(event) => setBasis(event.target.value)}
            />
          </label>
        </div>
      </section>

      <section className="invoice-section">
        <div className="invoice-section-head">
          <h2>Услуги</h2>
          <button
            type="button"
            className="icon-action icon-action-plus"
            disabled={saving}
            aria-label="Добавить позицию"
            onClick={() => setLines((prev) => [...prev, newLine()])}
          >
            +
          </button>
        </div>

        <div className="invoice-lines">
          {lines.map((line, index) => {
            const amount =
              Math.round(parseNum(line.quantity) * parseNum(line.price) * 100) /
              100;
            return (
              <div key={line.key} className="invoice-line">
                <div className="invoice-line-num">{index + 1}</div>
                <input
                  className="invoice-line-title"
                  placeholder="Наименование услуги"
                  value={line.title}
                  disabled={saving}
                  onChange={(event) =>
                    updateLine(line.key, { title: event.target.value })
                  }
                />
                <input
                  placeholder="Кол-во"
                  value={line.quantity}
                  disabled={saving}
                  onChange={(event) =>
                    updateLine(line.key, { quantity: event.target.value })
                  }
                />
                <input
                  placeholder="Ед."
                  value={line.unit}
                  disabled={saving}
                  onChange={(event) =>
                    updateLine(line.key, { unit: event.target.value })
                  }
                />
                <input
                  placeholder="Цена"
                  value={line.price}
                  disabled={saving}
                  onChange={(event) =>
                    updateLine(line.key, { price: event.target.value })
                  }
                />
                <div className="invoice-line-sum">{money(amount)}</div>
                {lines.length > 1 && (
                  <button
                    type="button"
                    className="icon-action icon-action-minus"
                    disabled={saving}
                    aria-label="Удалить позицию"
                    onClick={() =>
                      setLines((prev) =>
                        prev.filter((item) => item.key !== line.key),
                      )
                    }
                  >
                    −
                  </button>
                )}
              </div>
            );
          })}
        </div>

        <div className="invoice-totals">
          <div>
            <span>Итого</span>
            <strong>{money(totals.total)}</strong>
          </div>
          <div>
            <span>НДС 22%</span>
            <strong>{money(totals.vat)}</strong>
          </div>
          <div>
            <span>Всего к оплате</span>
            <strong>{money(totals.total)}</strong>
          </div>
        </div>
      </section>

      {error && <div className="error">{error}</div>}
      {success && <div className="invoice-success">{success}</div>}

      <button type="submit" disabled={saving}>
        {saving ? "Создаём..." : "Создать"}
      </button>
    </form>
  );
}
