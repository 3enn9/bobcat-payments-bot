import { useEffect, useState } from "react";
import {
  listBunkerHaulMessages,
  sendBunkerHaulReport,
  saveBunkerHaulAnnotation,
  type BunkerHaulAnnotationKind,
  type BunkerHaulMessage,
} from "../api/bunkerHauls";
import { createCashEntry, deleteCashEntry, type CashEntryType } from "../api/cash";

type ReportCashOperation = {
  localID: number;
  entryType: CashEntryType;
  amount: string;
  description: string;
  cashEntryID?: number;
  saving: boolean;
  error: string;
};

const annotationMethods: { kind: BunkerHaulAnnotationKind; label: string }[] = [
  { kind: "vector", label: "Вектор" },
  { kind: "dubki", label: "Дубки" },
  { kind: "cash", label: "Нал" },
];

function annotationLabel(message: BunkerHaulMessage): string {
  return message.annotations.map((annotation) => {
    const amount = new Intl.NumberFormat("ru-RU", {
      maximumFractionDigits: 3,
    }).format(annotation.amount);
    if (annotation.kind === "vector") return `Вектор — вес: ${amount}`;
    if (annotation.kind === "dubki") return `Дубки — ${amount} ₽`;
    return `Нал — ${amount} ₽`;
  }).join(" · ");
}

function todayISO(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export default function WorkerMessages() {
  const [surname, setSurname] = useState("");
  const [date, setDate] = useState(todayISO);
  const [messages, setMessages] = useState<BunkerHaulMessage[]>([]);
  const [comments, setComments] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [activeMessageID, setActiveMessageID] = useState<number | null>(null);
  const [selectedMethods, setSelectedMethods] = useState<BunkerHaulAnnotationKind[]>([]);
  const [annotationValues, setAnnotationValues] = useState<Partial<Record<BunkerHaulAnnotationKind, string>>>({});
  const [savingAnnotation, setSavingAnnotation] = useState(false);
  const [annotationError, setAnnotationError] = useState("");
  const [sendingReport, setSendingReport] = useState(false);
  const [reportError, setReportError] = useState("");
  const [reportSent, setReportSent] = useState(false);
  const [cashOperations, setCashOperations] = useState<ReportCashOperation[]>([]);
  const [nextCashOperationID, setNextCashOperationID] = useState(1);

  const hasSavedCashOperations = cashOperations.some((operation) => operation.cashEntryID !== undefined);
  const hasUnsavedCashOperations = cashOperations.some((operation) => operation.cashEntryID === undefined);

  useEffect(() => {
    const normalizedSurname = surname.trim();
    if (!normalizedSurname) {
      setMessages([]);
      setLoading(false);
      setError("");
      setActiveMessageID(null);
      return;
    }

    let cancelled = false;
    setMessages([]);
    setLoading(true);
    setError("");
    setActiveMessageID(null);
    const timer = window.setTimeout(() => {
      void listBunkerHaulMessages(normalizedSurname, date)
        .then((items) => {
          if (!cancelled) {
            setMessages(items);
          }
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setMessages([]);
            setError(err instanceof Error ? err.message : "Ошибка загрузки");
          }
        })
        .finally(() => {
          if (!cancelled) {
            setLoading(false);
          }
        });
    }, 250);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [surname, date]);

  const chooseMessage = (id: number) => {
    const current = messages.find((item) => item.id === id);
    setActiveMessageID((activeID) => (activeID === id ? null : id));
    setSelectedMethods(current?.annotations.map((annotation) => annotation.kind) ?? []);
    setAnnotationValues(Object.fromEntries(
      (current?.annotations ?? []).map((annotation) => [annotation.kind, String(annotation.amount)]),
    ));
    setAnnotationError("");
  };

  const saveAnnotation = async (message: BunkerHaulMessage) => {
    const annotations = selectedMethods.map((kind) => ({
      kind,
      amount: Number(annotationValues[kind]?.replace(",", ".")),
    }));
    if (annotations.some((annotation) => !Number.isFinite(annotation.amount) || annotation.amount <= 0)) return;

    setSavingAnnotation(true);
    setAnnotationError("");
    try {
      await saveBunkerHaulAnnotation(message.id, surname, date, annotations);
      setMessages((current) =>
        current.map((item) =>
          item.id === message.id
            ? { ...item, annotations }
            : item,
        ),
      );
      setActiveMessageID(null);
      setSelectedMethods([]);
      setAnnotationValues({});
    } catch (err) {
      setAnnotationError(err instanceof Error ? err.message : "Не удалось сохранить отметку");
    } finally {
      setSavingAnnotation(false);
    }
  };

  const sendReport = async () => {
    if (hasUnsavedCashOperations) {
      setReportError("Сначала добавьте все операции в кассу или удалите незаполненные строки.");
      return;
    }
    setSendingReport(true);
    setReportError("");
    setReportSent(false);
    try {
      await sendBunkerHaulReport(surname, date, comments);
      setReportSent(true);
    } catch (err) {
      setReportError(err instanceof Error ? err.message : "Не удалось отправить отчёт");
    } finally {
      setSendingReport(false);
    }
  };

  const addCashOperation = (entryType: CashEntryType) => {
    setCashOperations((current) => [
      ...current,
      {
        localID: nextCashOperationID,
        entryType,
        amount: "",
        description: "",
        saving: false,
        error: "",
      },
    ]);
    setNextCashOperationID((current) => current + 1);
    setReportError("");
    setReportSent(false);
  };

  const updateCashOperation = (localID: number, changes: Partial<ReportCashOperation>) => {
    setCashOperations((current) => current.map((operation) =>
      operation.localID === localID ? { ...operation, ...changes } : operation,
    ));
    setReportError("");
    setReportSent(false);
  };

  const saveCashOperation = async (operation: ReportCashOperation) => {
    const amount = Number(operation.amount.replace(",", "."));
    if (!surname.trim() || !Number.isFinite(amount) || amount <= 0) {
      updateCashOperation(operation.localID, { error: "Укажите фамилию и сумму больше нуля." });
      return;
    }

    updateCashOperation(operation.localID, { saving: true, error: "" });
    try {
      const result = await createCashEntry({
        workerName: surname.trim(),
        entryType: operation.entryType,
        amount,
        description: operation.description.trim(),
        entryDate: date,
      });
      updateCashOperation(operation.localID, { cashEntryID: result.id, saving: false });
    } catch (err) {
      updateCashOperation(operation.localID, {
        saving: false,
        error: err instanceof Error ? err.message : "Не удалось добавить операцию в кассу.",
      });
    }
  };

  const removeCashOperation = async (operation: ReportCashOperation) => {
    if (operation.cashEntryID !== undefined) {
      updateCashOperation(operation.localID, { saving: true, error: "" });
      try {
        await deleteCashEntry(operation.cashEntryID, surname.trim());
      } catch (err) {
        updateCashOperation(operation.localID, {
          saving: false,
          error: err instanceof Error ? err.message : "Не удалось удалить операцию из кассы.",
        });
        return;
      }
    }
    setCashOperations((current) => current.filter((item) => item.localID !== operation.localID));
    setReportSent(false);
  };

  return (
    <div className="worker-requests bunker-worker">
      <label className="invoice-field">
        <span>Фамилия работника</span>
        <input
          value={surname}
          onChange={(event) => setSurname(event.target.value)}
          placeholder="Введите фамилию"
          autoComplete="off"
          maxLength={128}
          disabled={hasSavedCashOperations}
        />
      </label>

      <label className="invoice-field">
        <span>Дата</span>
        <input
          type="date"
          value={date}
          onChange={(event) => setDate(event.target.value)}
          disabled={hasSavedCashOperations}
        />
      </label>

      <section className="bunker-cash-operations">
        <strong>Операции по кассе</strong>
        <div className="bunker-cash-operation-actions">
          <button
            type="button"
            className="fuels-filter-btn"
            disabled={!surname.trim()}
            onClick={() => addCashOperation("income")}
          >
            + Приход
          </button>
          <button
            type="button"
            className="fuels-filter-btn"
            disabled={!surname.trim()}
            onClick={() => addCashOperation("expense")}
          >
            + Расход
          </button>
        </div>
        {cashOperations.map((operation) => (
          <div className="bunker-cash-operation" key={operation.localID}>
            <strong>{operation.entryType === "income" ? "Приход" : "Расход"}</strong>
            {operation.cashEntryID === undefined ? (
              <>
                <label className="invoice-field">
                  <span>Сумма</span>
                  <input
                    type="number"
                    min="0.01"
                    step="0.01"
                    inputMode="decimal"
                    value={operation.amount}
                    onChange={(event) => updateCashOperation(operation.localID, { amount: event.target.value })}
                    placeholder="Введите сумму"
                  />
                </label>
                <label className="invoice-field">
                  <span>Описание</span>
                  <input
                    value={operation.description}
                    maxLength={500}
                    onChange={(event) => updateCashOperation(operation.localID, { description: event.target.value })}
                    placeholder="Введите описание"
                  />
                </label>
              </>
            ) : (
              <p className="bunker-cash-operation-saved">
                {new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 }).format(Number(operation.amount.replace(",", ".")))} ₽
                {operation.description.trim() && ` — ${operation.description.trim()}`}
                <small> добавлено в кассу</small>
              </p>
            )}
            {operation.error && <div className="error">{operation.error}</div>}
            <div className="bunker-cash-operation-actions">
              {operation.cashEntryID === undefined && (
                <button
                  type="button"
                  className="fuels-search-btn bunker-save-annotation"
                  disabled={operation.saving || Number(operation.amount.replace(",", ".")) <= 0}
                  onClick={() => void saveCashOperation(operation)}
                >
                  {operation.saving ? "Добавляю…" : "Добавить в кассу"}
                </button>
              )}
              <button
                type="button"
                className="fuels-filter-btn"
                disabled={operation.saving}
                onClick={() => void removeCashOperation(operation)}
              >
                {operation.cashEntryID === undefined ? "Убрать" : "Удалить из кассы"}
              </button>
            </div>
          </div>
        ))}
      </section>

      {loading && <p className="placeholder">Загружаю сообщения…</p>}
      {!loading && error && <div className="error">{error}</div>}
      {!loading && !error && surname.trim() !== "" && messages.length === 0 && (
        <p className="placeholder">Сообщений за эту дату нет.</p>
      )}

      {messages.length > 0 && (
        <ul className="requests-list bunker-message-list">
          {messages.map((message) => (
            <li key={message.id} className="request-item">
              <div className="request-item-meta">
                <time>{message.sentAt}</time>
              </div>
              <button
                type="button"
                className="bunker-message-body"
                aria-expanded={activeMessageID === message.id}
                onClick={() => chooseMessage(message.id)}
              >
                <span className="request-item-message bunker-message-text">{message.text}</span>
                {message.annotations.length > 0 && (
                  <span className="bunker-annotation">{annotationLabel(message)}</span>
                )}
              </button>
              {activeMessageID === message.id && (
                <div className="bunker-annotation-editor">
                  <div className="bunker-method-buttons">
                    {annotationMethods.map((method) => (
                      <button
                        key={method.kind}
                        type="button"
                        className={`fuels-filter-btn${selectedMethods.includes(method.kind) ? " active" : ""}`}
                        onClick={() => {
                          setSelectedMethods((current) => current.includes(method.kind)
                            ? current.filter((kind) => kind !== method.kind)
                            : [...current, method.kind]);
                          if (!selectedMethods.includes(method.kind)) {
                            const existing = messages.find((item) => item.id === message.id)
                              ?.annotations.find((annotation) => annotation.kind === method.kind);
                            setAnnotationValues((current) => ({
                              ...current,
                              [method.kind]: existing ? String(existing.amount) : "",
                            }));
                          }
                          setAnnotationError("");
                        }}
                      >
                        {method.label}
                      </button>
                    ))}
                  </div>
                  {selectedMethods.map((kind) => (
                    <label className="invoice-field" key={kind}>
                      <span>{kind === "vector" ? "Вес Вектор" : `Сумма ${kind === "dubki" ? "Дубки" : "Нал"}`}</span>
                      <input
                        type="number"
                        min="0.001"
                        step={kind === "vector" ? "0.001" : "0.01"}
                        inputMode="decimal"
                        value={annotationValues[kind] ?? ""}
                        onChange={(event) => setAnnotationValues((current) => ({
                          ...current,
                          [kind]: event.target.value,
                        }))}
                        placeholder={kind === "vector" ? "Введите вес" : "Введите сумму"}
                      />
                    </label>
                  ))}
                  {(selectedMethods.length > 0 || message.annotations.length > 0) && (
                    <>
                      {annotationError && <div className="error">{annotationError}</div>}
                      <button
                        type="button"
                        className="fuels-search-btn bunker-save-annotation"
                        disabled={savingAnnotation || selectedMethods.some((kind) => {
                          const value = Number(annotationValues[kind]?.replace(",", "."));
                          return !Number.isFinite(value) || value <= 0;
                        })}
                        onClick={() => void saveAnnotation(message)}
                      >
                        {savingAnnotation ? "Сохраняю…" : "✓ Сохранить"}
                      </button>
                    </>
                  )}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      <label className="worker-comment">
        <span>Комментарии</span>
        <textarea
          value={comments}
          onChange={(event) => {
            setComments(event.target.value);
            setReportError("");
            setReportSent(false);
          }}
          placeholder="Введите комментарий"
          maxLength={2000}
          rows={3}
        />
      </label>

      {reportError && <div className="error">{reportError}</div>}
      {reportSent && <small className="bunker-report-success">Отчёт отправлен в группу MAX.</small>}
      <button
        type="button"
        className="fuels-search-btn"
        disabled={sendingReport || loading || messages.length === 0 || hasUnsavedCashOperations}
        onClick={() => void sendReport()}
      >
        {sendingReport ? "Отправляю…" : "Отправить"}
      </button>
    </div>
  );
}
