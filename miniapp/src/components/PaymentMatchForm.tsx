import { useEffect, useMemo, useState } from "react";
import {
  listMatchData,
  listMatchFirms,
  matchPayment,
  type MatchFirm,
  type MatchInvoice,
  type MatchPayment,
} from "../api/payments";

function money(value: number): string {
  return value.toFixed(2).replace(".", ",");
}

function shortDate(value: string): string {
  if (!value) return "—";
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) return value;
  return `${d}.${m}.${y}`;
}

function truncate(text: string, max = 90): string {
  const t = text.replace(/\s+/g, " ").trim();
  if (t.length <= max) return t;
  return t.slice(0, max - 1) + "…";
}

export default function PaymentMatchForm() {
  const [firms, setFirms] = useState<MatchFirm[]>([]);
  const [supplierId, setSupplierId] = useState<number | null>(null);
  const [payments, setPayments] = useState<MatchPayment[]>([]);
  const [invoices, setInvoices] = useState<MatchInvoice[]>([]);
  const [paymentId, setPaymentId] = useState<number | null>(null);
  const [selectedInvoices, setSelectedInvoices] = useState<Set<number>>(
    () => new Set(),
  );
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    void listMatchFirms()
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
      setPayments([]);
      setInvoices([]);
      setPaymentId(null);
      setSelectedInvoices(new Set());
      return;
    }
    setLoading(true);
    setError("");
    setSuccess("");
    void listMatchData(supplierId)
      .then(({ payments: p, invoices: i }) => {
        setPayments(p);
        setInvoices(i);
        setPaymentId(null);
        setSelectedInvoices(new Set());
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [supplierId]);

  const selectedPayment = useMemo(
    () => payments.find((p) => p.id === paymentId) ?? null,
    [payments, paymentId],
  );

  const selectedTotal = useMemo(() => {
    let sum = 0;
    for (const inv of invoices) {
      if (selectedInvoices.has(inv.id)) {
        sum += inv.total;
      }
    }
    return Math.round(sum * 100) / 100;
  }, [invoices, selectedInvoices]);

  const paymentAmount = selectedPayment?.amount ?? 0;
  const amountsEqual =
    selectedPayment != null &&
    selectedInvoices.size > 0 &&
    selectedTotal === paymentAmount;

  function toggleInvoice(id: number) {
    setSelectedInvoices((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  async function onMatch() {
    if (!selectedPayment || !amountsEqual) return;
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await matchPayment(selectedPayment.id, Array.from(selectedInvoices));
      setSuccess("Сопоставлено");
      const data = await listMatchData(supplierId!);
      setPayments(data.payments);
      setInvoices(data.invoices);
      setPaymentId(null);
      setSelectedInvoices(new Set());
      const refreshed = await listMatchFirms();
      setFirms(refreshed);
      if (!refreshed.some((f) => f.id === supplierId)) {
        setSupplierId(refreshed[0]?.id ?? null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="match-form">
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
              {f.name} · платежей {f.unmatchedCount} · счетов{" "}
              {f.unpaidInvoiceCount}
            </option>
          ))}
        </select>
      </label>

      {firms.length === 0 && !error && (
        <p className="invoice-hint">Нет фирм с нераспределёнными платежами</p>
      )}

      {loading && <p className="invoice-hint">Загрузка…</p>}

      {supplierId && !loading && (
        <>
          <section className="match-section">
            <h3>Входящие платежи</h3>
            {payments.length === 0 ? (
              <p className="invoice-hint">Нет нераспределённых платежей</p>
            ) : (
              <ul className="match-list">
                {payments.map((p) => (
                  <li key={p.id}>
                    <label className="match-row">
                      <input
                        type="radio"
                        name="payment"
                        checked={paymentId === p.id}
                        onChange={() => setPaymentId(p.id)}
                      />
                      <span className="match-row-body">
                        <strong>
                          {shortDate(String(p.executedAt))} · {money(p.amount)} ₽
                        </strong>
                        <small>{p.payerName || "—"}</small>
                        <small className="match-purpose">
                          {truncate(p.purpose || "—")}
                        </small>
                      </span>
                    </label>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="match-section">
            <h3>Неоплаченные счета</h3>
            {invoices.length === 0 ? (
              <p className="invoice-hint">Нет неоплаченных счетов</p>
            ) : (
              <ul className="match-list">
                {invoices.map((inv) => (
                  <li key={inv.id}>
                    <label className="match-row">
                      <input
                        type="checkbox"
                        checked={selectedInvoices.has(inv.id)}
                        onChange={() => toggleInvoice(inv.id)}
                        disabled={!paymentId}
                      />
                      <span className="match-row-body">
                        <strong>
                          №{inv.number} · {shortDate(String(inv.invoiceDate))} ·{" "}
                          {money(inv.total)} ₽
                        </strong>
                        <small>{inv.buyerName}</small>
                      </span>
                    </label>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <div
            className={`match-summary ${
              amountsEqual ? "match-ok" : selectedPayment ? "match-bad" : ""
            }`}
          >
            <div>
              Счета: <strong>{money(selectedTotal)}</strong>
            </div>
            <div>
              Платёж:{" "}
              <strong>
                {selectedPayment ? money(paymentAmount) : "—"}
              </strong>
            </div>
            {selectedPayment && selectedInvoices.size > 0 && !amountsEqual && (
              <small>
                Разница: {money(Math.abs(selectedTotal - paymentAmount))}
              </small>
            )}
          </div>

          <button
            type="button"
            className="submit"
            disabled={!amountsEqual || saving}
            onClick={() => void onMatch()}
          >
            {saving ? "Сохраняю…" : "Сопоставить"}
          </button>
        </>
      )}

      {error && <p className="error">{error}</p>}
      {success && <p className="invoice-success">{success}</p>}
    </div>
  );
}
