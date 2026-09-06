import { useEffect, useMemo, useState } from "react";
import {
  listUnpaidFirms,
  listUnpaidInvoices,
  type UnpaidFirm,
  type UnpaidInvoice,
} from "../api/invoices";

function money(value: number): string {
  return value.toFixed(2).replace(".", ",");
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

export default function UnpaidInvoicesTable() {
  const [firms, setFirms] = useState<UnpaidFirm[]>([]);
  const [supplierId, setSupplierId] = useState<number | null>(null);
  const [invoices, setInvoices] = useState<UnpaidInvoice[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

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
      return;
    }
    setLoading(true);
    setError("");
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
              {invoices.map((inv) => (
                <tr key={inv.id}>
                  <td>{inv.number}</td>
                  <td className="date">{shortDate(String(inv.invoiceDate))}</td>
                  <td className="buyer" title={inv.buyerName}>
                    {shortBuyerName(inv.buyerName)}
                  </td>
                  <td className="num">{money(inv.total)}</td>
                  <td className="num">{money(inv.paidAmount)}</td>
                  <td className="num">{money(inv.remainingAmount)}</td>
                </tr>
              ))}
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
