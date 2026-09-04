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

function round2(v: number): number {
  return Math.round(v * 100) / 100;
}

export default function PaymentMatchForm() {
  const [firms, setFirms] = useState<MatchFirm[]>([]);
  const [supplierId, setSupplierId] = useState<number | null>(null);
  const [payments, setPayments] = useState<MatchPayment[]>([]);
  const [invoices, setInvoices] = useState<MatchInvoice[]>([]);
  const [paymentId, setPaymentId] = useState<number | null>(null);
  const [selectedInvoices, setSelectedInvoices] = useState<number[]>([]);
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
      setSelectedInvoices([]);
      return;
    }
    setLoading(true);
    setError("");
    setSuccess("");
    void listMatchData(supplierId, paymentId)
      .then(({ payments: p, invoices: i }) => {
        setPayments(p);
        setInvoices(i);
        setSelectedInvoices((prev) =>
          prev.filter((id) => i.some((inv) => inv.id === id)),
        );
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [supplierId, paymentId]);

  const selectedPayment = useMemo(
    () => payments.find((p) => p.id === paymentId) ?? null,
    [payments, paymentId],
  );

  const selectedInvoiceRows = useMemo(
    () =>
      selectedInvoices
        .map((id) => invoices.find((inv) => inv.id === id))
        .filter((x): x is MatchInvoice => !!x),
    [invoices, selectedInvoices],
  );

  const selectedRemaining = useMemo(() => {
    let sum = 0;
    for (const inv of selectedInvoiceRows) {
      sum += inv.remainingAmount;
    }
    return round2(sum);
  }, [selectedInvoiceRows]);

  const paymentRemaining = selectedPayment?.remainingAmount ?? 0;

  const preview = useMemo(() => {
    let payLeft = paymentRemaining;
    let covered = 0;
    const leftovers: Array<{ id: number; leftover: number }> = [];
    for (const inv of selectedInvoiceRows) {
      const take = Math.min(inv.remainingAmount, payLeft);
      covered = round2(covered + take);
      payLeft = round2(payLeft - take);
      leftovers.push({
        id: inv.id,
        leftover: round2(inv.remainingAmount - take),
      });
    }
    return {
      covered,
      paymentLeftover: payLeft,
      invoiceLeftovers: leftovers,
    };
  }, [selectedInvoiceRows, paymentRemaining]);

  const canMatch =
    selectedPayment != null &&
    selectedInvoices.length > 0 &&
    paymentRemaining > 0;

  function toggleInvoice(id: number) {
    setSelectedInvoices((prev) => {
      if (prev.includes(id)) {
        return prev.filter((x) => x !== id);
      }
      return [...prev, id];
    });
  }

  async function onMatch() {
    if (!canMatch || !selectedPayment || !supplierId) return;
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await matchPayment(selectedPayment.id, selectedInvoices);
      setSuccess("Сопоставлено");
      const refreshedFirms = await listMatchFirms();
      setFirms(refreshedFirms);
      const stillHere = refreshedFirms.some((f) => f.id === supplierId);
      if (!stillHere) {
        setSupplierId(refreshedFirms[0]?.id ?? null);
        setPaymentId(null);
        return;
      }
      const data = await listMatchData(supplierId, null);
      setPayments(data.payments);
      setInvoices([]);
      setPaymentId(null);
      setSelectedInvoices([]);
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
            setPaymentId(null);
            setSelectedInvoices([]);
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
                {payments.map((p) => {
                  const partial = p.remainingAmount < p.amount;
                  return (
                    <li key={p.id}>
                      <label className="match-row">
                        <input
                          type="radio"
                          name="payment"
                          checked={paymentId === p.id}
                          onChange={() => {
                            setPaymentId(p.id);
                            setSelectedInvoices([]);
                          }}
                        />
                        <span className="match-row-body">
                          <strong>
                            {shortDate(String(p.executedAt))} · остаток{" "}
                            {money(p.remainingAmount)} ₽
                            {partial ? ` из ${money(p.amount)}` : ""}
                          </strong>
                          <small>{p.payerName || "—"}</small>
                          <small className="match-purpose">
                            {truncate(p.purpose || "—")}
                          </small>
                        </span>
                      </label>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>

          <section className="match-section">
            <h3>Счета плательщика</h3>
            {!paymentId ? (
              <p className="invoice-hint">Сначала выберите платёж</p>
            ) : invoices.length === 0 ? (
              <p className="invoice-hint">
                Нет открытых счетов для этого плательщика
              </p>
            ) : (
              <ul className="match-list">
                {invoices.map((inv) => {
                  const partial = inv.paidAmount > 0;
                  const leftover = preview.invoiceLeftovers.find(
                    (x) => x.id === inv.id,
                  );
                  return (
                    <li key={inv.id}>
                      <label className="match-row">
                        <input
                          type="checkbox"
                          checked={selectedInvoices.includes(inv.id)}
                          onChange={() => toggleInvoice(inv.id)}
                        />
                        <span className="match-row-body">
                          <strong>
                            №{inv.number} · {shortDate(String(inv.invoiceDate))}{" "}
                            · к оплате {money(inv.remainingAmount)} ₽
                          </strong>
                          <small>{inv.buyerName}</small>
                          {partial && (
                            <small>
                              Уже оплачено {money(inv.paidAmount)} из{" "}
                              {money(inv.total)}
                            </small>
                          )}
                          {leftover &&
                            selectedInvoices.includes(inv.id) &&
                            leftover.leftover > 0 && (
                              <small className="match-leftover">
                                После сопоставления останется доплатить{" "}
                                {money(leftover.leftover)} ₽
                              </small>
                            )}
                        </span>
                      </label>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>

          <div
            className={`match-summary ${
              canMatch
                ? preview.paymentLeftover === 0 &&
                  preview.invoiceLeftovers.every((x) => x.leftover === 0)
                  ? "match-ok"
                  : "match-partial"
                : ""
            }`}
          >
            <div>
              К оплате по счетам: <strong>{money(selectedRemaining)}</strong>
            </div>
            <div>
              Остаток платежа:{" "}
              <strong>
                {selectedPayment ? money(paymentRemaining) : "—"}
              </strong>
            </div>
            {canMatch && (
              <>
                <div>
                  Будет списано: <strong>{money(preview.covered)}</strong>
                </div>
                {preview.paymentLeftover > 0 && (
                  <small>
                    После сопоставления у платежа останется{" "}
                    {money(preview.paymentLeftover)} ₽
                  </small>
                )}
              </>
            )}
          </div>

          <button
            type="button"
            className="submit"
            disabled={!canMatch || saving}
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
