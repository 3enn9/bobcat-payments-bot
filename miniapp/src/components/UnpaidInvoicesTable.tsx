import { Fragment, useEffect, useMemo, useState } from "react";
import {
  listInvoiceItems,
  listUnpaidFirms,
  listUnpaidInvoices,
  type InvoiceLine,
  type UnpaidFirm,
  type UnpaidInvoice,
} from "../api/invoices";

function money(value: number): string {
  const n = Math.round(value * 100) / 100;
  if (Number.isInteger(n)) {
    return String(n);
  }
  return String(n).replace(".", ",");
}

function shortDate(value: string): string {
  if (!value) return "—";
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) return value;
  return `${d}.${m}.${y.slice(2)}`;
}

const LEGAL_PREFIXES = [
  /^индивидуальный\s+предприниматель\s+/i,
  /^общество\s+с\s+ограниченной\s+ответственностью\s+/i,
  /^публичное\s+акционерное\s+общество\s+/i,
  /^открытое\s+акционерное\s+общество\s+/i,
  /^закрытое\s+акционерное\s+общество\s+/i,
  /^акционерное\s+общество\s+/i,
  /^(ип|ооо|оао|пао|зао|нао|ао)\s+/i,
];

function shortBuyerName(name: string): string {
  let s = name.replace(/\u00a0/g, " ").replace(/\s+/g, " ").trim();
  if (!s) return "—";
  for (let i = 0; i < 3; i++) {
    let cut = false;
    for (const re of LEGAL_PREFIXES) {
      const next = s.replace(re, "");
      if (next !== s) {
        s = next.trim().replace(/^[,.\-–—]\s*/, "");
        cut = true;
        break;
      }
    }
    if (!cut) break;
  }
  return s.replace(/^["«]+|["»]+$/g, "").trim() || name;
}

function qty(value: number): string {
  const n = Math.round(value * 1000) / 1000;
  if (Number.isInteger(n)) {
    return String(n);
  }
  return String(n).replace(".", ",");
}

export default function UnpaidInvoicesTable() {
  const [firms, setFirms] = useState<UnpaidFirm[]>([]);
  const [supplierId, setSupplierId] = useState<number | null>(null);
  const [invoices, setInvoices] = useState<UnpaidInvoice[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [itemsById, setItemsById] = useState<Record<number, InvoiceLine[]>>({});
  const [itemsLoading, setItemsLoading] = useState<number | null>(null);

  useEffect(() => {
    void listUnpaidFirms()
      .then((items) => {
        setFirms(items);
        if (items.length === 1) {
          setSupplierId(items[0].id);
        }
      })
      .catch((err: Error) => setError(err.message));
  }, []);

  useEffect(() => {
    if (!supplierId) {
      setInvoices([]);
      setExpandedId(null);
      return;
    }
    setLoading(true);
    setError("");
    setExpandedId(null);
    void listUnpaidInvoices(supplierId)
      .then(setInvoices)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [supplierId]);

  const totals = useMemo(() => {
    let remaining = 0;
    let total = 0;
    let paid = 0;
    for (const inv of invoices) {
      remaining += inv.remainingAmount;
      total += inv.total;
      paid += inv.paidAmount;
    }
    return {
      remaining: Math.round(remaining * 100) / 100,
      total: Math.round(total * 100) / 100,
      paid: Math.round(paid * 100) / 100,
    };
  }, [invoices]);

  async function toggleInvoice(id: number) {
    if (expandedId === id) {
      setExpandedId(null);
      return;
    }
    setExpandedId(id);
    if (itemsById[id]) {
      return;
    }
    setItemsLoading(id);
    try {
      const items = await listInvoiceItems(id);
      setItemsById((prev) => ({ ...prev, [id]: items }));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка загрузки позиций");
      setExpandedId(null);
    } finally {
      setItemsLoading(null);
    }
  }

  return (
    <div className="unpaid-form">
      <label className="invoice-field">
        <span>Фирма</span>
        <select
          value={supplierId ?? ""}
          onChange={(e) => {
            const v = e.target.value;
            setSupplierId(v ? Number(v) : null);
          }}
        >
          <option value="">Выберите фирму</option>
          {firms.map((f) => (
            <option key={f.id} value={f.id}>
              {f.name} · {f.unpaidInvoiceCount}
            </option>
          ))}
        </select>
      </label>

      {firms.length === 0 && !error && (
        <p className="invoice-hint">Нет фирм с неоплаченными счетами</p>
      )}

      {loading && <p className="invoice-hint">Загрузка…</p>}

      {supplierId && !loading && invoices.length === 0 && !error && (
        <p className="invoice-hint">Нет открытых счетов</p>
      )}

      {supplierId && !loading && invoices.length > 0 && (
        <div className="unpaid-table-wrap">
          <table className="unpaid-table">
            <thead>
              <tr>
                <th>№</th>
                <th className="date">Дата</th>
                <th>Покупатель</th>
                <th className="num">Сумма</th>
                <th className="num">Опл.</th>
                <th className="num">К оплате</th>
              </tr>
            </thead>
            <tbody>
              {invoices.map((inv) => {
                const open = expandedId === inv.id;
                const lines = itemsById[inv.id];
                return (
                  <Fragment key={inv.id}>
                    <tr
                      className={open ? "unpaid-row open" : "unpaid-row"}
                      onClick={() => void toggleInvoice(inv.id)}
                    >
                      <td>{inv.number}</td>
                      <td className="date">
                        {shortDate(String(inv.invoiceDate))}
                      </td>
                      <td className="buyer" title={inv.buyerName}>
                        {shortBuyerName(inv.buyerName)}
                      </td>
                      <td className="num">{money(inv.total)}</td>
                      <td className="num">{money(inv.paidAmount)}</td>
                      <td className="num">{money(inv.remainingAmount)}</td>
                    </tr>
                    {open && (
                      <tr className="unpaid-items-row">
                        <td colSpan={6}>
                          {itemsLoading === inv.id && (
                            <p className="invoice-hint">Загрузка позиций…</p>
                          )}
                          {lines && lines.length === 0 && (
                            <p className="invoice-hint">Нет позиций</p>
                          )}
                          {lines && lines.length > 0 && (
                            <ul className="unpaid-items">
                              {lines.map((line, idx) => (
                                <li key={`${inv.id}-${line.position}-${idx}`}>
                                  <span className="unpaid-item-title">
                                    {line.title}
                                  </span>
                                  <span className="unpaid-item-sum">
                                    {qty(line.quantity)} {line.unit} ×{" "}
                                    {money(line.price)} = {money(line.amount)}
                                  </span>
                                </li>
                              ))}
                            </ul>
                          )}
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
            <tfoot>
              <tr>
                <td colSpan={3}>Итого · {invoices.length} сч.</td>
                <td className="num">{money(totals.total)}</td>
                <td className="num">{money(totals.paid)}</td>
                <td className="num">{money(totals.remaining)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      )}

      {error && <p className="error">{error}</p>}
    </div>
  );
}
