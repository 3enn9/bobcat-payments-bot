import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import {
  addBuyerEmail,
  deleteBuyerEmail,
  listBuyerEmails,
  listUnpaidBuyerFirms,
  type InvoiceBuyerEmail,
  type UnpaidBuyerFirm,
} from "../api/invoices";

export default function InvoiceEmailManager() {
  const [firms, setFirms] = useState<UnpaidBuyerFirm[]>([]);
  const [selectedFirmId, setSelectedFirmId] = useState<number | null>(null);
  const [query, setQuery] = useState("");
  const [emails, setEmails] = useState<InvoiceBuyerEmail[]>([]);
  const [newEmail, setNewEmail] = useState("");
  const [loadingFirms, setLoadingFirms] = useState(true);
  const [loadingEmails, setLoadingEmails] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    void listUnpaidBuyerFirms()
      .then((items) => setFirms(items))
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoadingFirms(false));
  }, []);

  useEffect(() => {
    if (!selectedFirmId) {
      setEmails([]);
      return;
    }
    let active = true;
    setLoadingEmails(true);
    setError("");
    setSuccess("");
    void listBuyerEmails(selectedFirmId)
      .then((items) => {
        if (active) setEmails(items);
      })
      .catch((err: Error) => {
        if (active) setError(err.message);
      })
      .finally(() => {
        if (active) setLoadingEmails(false);
      });
    return () => {
      active = false;
    };
  }, [selectedFirmId]);

  const visibleFirms = useMemo(() => {
    const normalized = query.trim().toLocaleLowerCase().replace(/ё/g, "е");
    if (!normalized) return firms;
    return firms.filter((firm) =>
      `${firm.name} ${firm.inn}`.toLocaleLowerCase().replace(/ё/g, "е").includes(normalized),
    );
  }, [firms, query]);

  const selectedFirm = firms.find((firm) => firm.id === selectedFirmId) ?? null;

  async function onAddEmail(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedFirmId || !newEmail.trim()) return;
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      await addBuyerEmail(selectedFirmId, newEmail.trim());
      const nextEmails = await listBuyerEmails(selectedFirmId);
      setEmails(nextEmails);
      setNewEmail("");
      setSuccess("Адрес добавлен");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось добавить адрес");
    } finally {
      setSaving(false);
    }
  }

  async function onDeleteEmail(email: InvoiceBuyerEmail) {
    if (!selectedFirmId) return;
    setDeletingId(email.id);
    setError("");
    setSuccess("");
    try {
      await deleteBuyerEmail(selectedFirmId, email.id);
      setEmails((current) => current.filter((item) => item.id !== email.id));
      setSuccess("Адрес удалён");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось удалить адрес");
    } finally {
      setDeletingId(null);
    }
  }

  return (
    <div className="invoice-email-manager">
      <label className="invoice-field">
        <span>Поиск фирмы</span>
        <input
          type="search"
          value={query}
          placeholder="Название или ИНН"
          onChange={(event) => setQuery(event.target.value)}
        />
      </label>

      {loadingFirms && <p className="invoice-hint">Загрузка фирм…</p>}
      {!loadingFirms && firms.length === 0 && !error && (
        <p className="invoice-hint">Нет фирм с неоплаченными счетами</p>
      )}
      {!loadingFirms && firms.length > 0 && visibleFirms.length === 0 && (
        <p className="invoice-hint">Совпадений нет</p>
      )}

      {visibleFirms.length > 0 && (
        <ul className="invoice-email-firms">
          {visibleFirms.map((firm) => (
            <li key={firm.id}>
              <button
                type="button"
                className={
                  selectedFirmId === firm.id
                    ? "invoice-email-firm selected"
                    : "invoice-email-firm"
                }
                aria-pressed={selectedFirmId === firm.id}
                onClick={() => setSelectedFirmId(firm.id)}
              >
                <span className="invoice-email-firm-name">{firm.name}</span>
                <small>{firm.unpaidInvoiceCount} неоплаченных</small>
              </button>
            </li>
          ))}
        </ul>
      )}

      {selectedFirm && (
        <section className="invoice-email-details">
          <div className="invoice-email-details-heading">
            <h2>{selectedFirm.name}</h2>
            <span>ИНН {selectedFirm.inn}</span>
          </div>

          {loadingEmails ? (
            <p className="invoice-hint">Загрузка адресов…</p>
          ) : (
            <>
              {emails.length === 0 ? (
                <p className="invoice-hint">Для фирмы пока нет email</p>
              ) : (
                <ul className="invoice-email-list">
                  {emails.map((email) => (
                    <li key={email.id}>
                      <span>{email.email}</span>
                      <button
                        type="button"
                        className="invoice-email-delete"
                        aria-label={`Удалить ${email.email}`}
                        disabled={deletingId !== null}
                        onClick={() => void onDeleteEmail(email)}
                      >
                        {deletingId === email.id ? "…" : "Удалить"}
                      </button>
                    </li>
                  ))}
                </ul>
              )}

              <form className="invoice-email-add" onSubmit={onAddEmail}>
                <label className="invoice-field">
                  <span>Новый адрес</span>
                  <input
                    type="email"
                    value={newEmail}
                    placeholder="email@example.com"
                    required
                    onChange={(event) => setNewEmail(event.target.value)}
                  />
                </label>
                <button type="submit" disabled={saving || !newEmail.trim()}>
                  {saving ? "Добавляем…" : "Добавить email"}
                </button>
              </form>
            </>
          )}
        </section>
      )}

      {error && <p className="error">{error}</p>}
      {success && <p className="invoice-email-success">{success}</p>}
    </div>
  );
}
