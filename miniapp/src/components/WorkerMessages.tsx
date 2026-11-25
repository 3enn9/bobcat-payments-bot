import { useEffect, useState } from "react";
import {
  getBunkerHaulReportComment,
  listBunkerHaulMessages,
  saveBunkerHaulReportComment,
  sendBunkerHaulReport,
  saveBunkerHaulAnnotation,
  suggestBunkerHaulSurnames,
  type BunkerHaulAnnotationKind,
  type BunkerHaulMessage,
} from "../api/bunkerHauls";
import {
  createCashEntry,
  deleteCashEntry,
  fetchWorkerCashEntriesByDate,
  updateCashEntry,
  type CashEntryType,
} from "../api/cash";

type ReportCashOperation = {
  localID: number;
  entryType: CashEntryType;
  amount: string;
  description: string;
  cashEntryID?: number;
  dirty?: boolean;
  saving: boolean;
  error: string;
};

const annotationMethods: { kind: BunkerHaulAnnotationKind; label: string }[] = [
  { kind: "vector", label: "Вектор" },
  { kind: "dubki", label: "Дубки" },
  { kind: "cash", label: "Нал" },
  { kind: "base", label: "База" },
  { kind: "rsm", label: "РСМ" },
];

function annotationLabel(message: BunkerHaulMessage): string {
  return message.annotations.map((annotation) => {
    if (annotation.kind === "base") return "База";
    if (annotation.kind === "rsm") return "РСМ";
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
  const [surnameSuggestions, setSurnameSuggestions] = useState<string[]>([]);
  const [showSurnameSuggestions, setShowSurnameSuggestions] = useState(false);
  const [date, setDate] = useState(todayISO);
  const [messages, setMessages] = useState<BunkerHaulMessage[]>([]);
  const [comments, setComments] = useState("");
  const [commentStatus, setCommentStatus] = useState<"idle" | "loading" | "unsaved" | "saving" | "saved" | "error">("idle");
  const [commentSaveError, setCommentSaveError] = useState("");
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
  const [cashOperationsLoading, setCashOperationsLoading] = useState(false);
  const [cashOperationsError, setCashOperationsError] = useState("");
  const [nextCashOperationID, setNextCashOperationID] = useState(1);
  const [editCashOperationID, setEditCashOperationID] = useState<number | null>(null);

  const hasSavedCashOperations = cashOperations.some((operation) => operation.cashEntryID !== undefined);
  const hasUnsavedCashOperations = cashOperations.some((operation) => operation.cashEntryID === undefined);
  const hasSavingCashOperations = cashOperations.some((operation) => operation.saving);
  const hasReportContent = messages.length > 0 || comments.trim() !== "" || hasSavedCashOperations;

  useEffect(() => {
    const query = surname.trim();
    if (!showSurnameSuggestions || !query) {
      setSurnameSuggestions([]);
      return;
    }

    let cancelled = false;
    const timer = window.setTimeout(() => {
      void suggestBunkerHaulSurnames(query)
        .then((items) => {
          if (!cancelled) setSurnameSuggestions(items);
        })
        .catch(() => {
          if (!cancelled) setSurnameSuggestions([]);
        });
    }, 180);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [surname, showSurnameSuggestions]);

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

  useEffect(() => {
    const normalizedSurname = surname.trim();
    let cancelled = false;
    setCashOperations([]);
    setEditCashOperationID(null);
    setCashOperationsError("");
    if (!normalizedSurname || !date) {
      setCashOperationsLoading(false);
      return;
    }

    setCashOperationsLoading(true);
    void fetchWorkerCashEntriesByDate(normalizedSurname, date)
      .then((entries) => {
        if (!cancelled) {
          setCashOperations(entries.map((entry, index) => ({
            localID: index + 1,
            entryType: entry.entryType,
            amount: String(entry.amount),
            description: entry.description,
            cashEntryID: entry.id,
            dirty: false,
            saving: false,
            error: "",
          })));
          setNextCashOperationID(entries.length + 1);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setCashOperationsError(err instanceof Error ? err.message : "Не удалось загрузить операции кассы");
        }
      })
      .finally(() => {
        if (!cancelled) setCashOperationsLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [surname, date]);

  useEffect(() => {
    const normalizedSurname = surname.trim();
    let cancelled = false;
    if (!normalizedSurname) {
      setComments("");
      setCommentStatus("idle");
      setCommentSaveError("");
      return;
    }

    setComments("");
    setCommentStatus("loading");
    setCommentSaveError("");
    const timer = window.setTimeout(() => {
      void getBunkerHaulReportComment(normalizedSurname, date)
        .then((savedComments) => {
          if (!cancelled) {
            setComments(savedComments);
            setCommentStatus(savedComments ? "saved" : "idle");
          }
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setCommentStatus("error");
            setCommentSaveError(err instanceof Error ? err.message : "Не удалось загрузить комментарий");
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
      amount: kind === "base" || kind === "rsm"
        ? 1
        : Number(annotationValues[kind]?.replace(",", ".")),
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

  const saveComments = async (): Promise<boolean> => {
    const normalizedSurname = surname.trim();
    if (!normalizedSurname) {
      setCommentStatus("error");
      setCommentSaveError("Сначала укажите фамилию работника.");
      return false;
    }

    setCommentStatus("saving");
    setCommentSaveError("");
    try {
      await saveBunkerHaulReportComment(normalizedSurname, date, comments);
      setCommentStatus("saved");
      return true;
    } catch (err) {
      setCommentStatus("error");
      setCommentSaveError(err instanceof Error ? err.message : "Не удалось сохранить комментарий");
      return false;
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
      if (!(await saveComments())) {
        setReportError("Не удалось сохранить комментарий. Отчёт не отправлен.");
        return;
      }
      const cashEntryIDs = cashOperations.flatMap((operation) =>
        operation.cashEntryID === undefined ? [] : [operation.cashEntryID],
      );
      await sendBunkerHaulReport(surname, date, comments, cashEntryIDs);
      setReportSent(true);
    } catch (err) {
      setReportError(err instanceof Error ? err.message : "Не удалось отправить отчёт");
    } finally {
      setSendingReport(false);
    }
  };

  const addCashOperation = (entryType: CashEntryType) => {
    const localID = nextCashOperationID;
    setCashOperations((current) => [
      ...current,
      {
        localID,
        entryType,
        amount: "",
        description: "",
        saving: false,
        error: "",
      },
    ]);
    setEditCashOperationID(localID);
    setNextCashOperationID((current) => current + 1);
    setReportError("");
    setReportSent(false);
  };

  const updateCashOperation = (localID: number, changes: Partial<ReportCashOperation>) => {
    setCashOperations((current) => current.map((operation) =>
      operation.localID === localID
        ? {
            ...operation,
            ...changes,
            dirty: changes.dirty ?? (
              operation.cashEntryID !== undefined &&
              ("entryType" in changes || "amount" in changes || "description" in changes)
                ? true
                : operation.dirty
            ),
          }
        : operation,
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
    const payload = {
        workerName: surname.trim(),
        entryType: operation.entryType,
        amount,
        description: operation.description.trim(),
        entryDate: date,
      };
    try {
      if (operation.cashEntryID !== undefined) {
        await updateCashEntry(operation.cashEntryID, payload);
        updateCashOperation(operation.localID, { saving: false, dirty: false });
      } else {
        const result = await createCashEntry(payload);
        updateCashOperation(operation.localID, { cashEntryID: result.id, saving: false });
      }
      setEditCashOperationID(null);
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
    setEditCashOperationID((current) => current === operation.localID ? null : current);
    setReportSent(false);
  };

  return (
    <div className="worker-requests bunker-worker">
      <div className="worker-surname-autocomplete">
        <label className="invoice-field">
          <span>Фамилия работника</span>
          <input
            value={surname}
            disabled={sendingReport}
            onChange={(event) => {
              setSurname(event.target.value);
              setShowSurnameSuggestions(true);
            }}
            onFocus={() => setShowSurnameSuggestions(true)}
            onBlur={() => window.setTimeout(() => setShowSurnameSuggestions(false), 120)}
            placeholder="Введите фамилию"
            autoComplete="off"
            maxLength={128}
            role="combobox"
            aria-autocomplete="list"
            aria-expanded={showSurnameSuggestions && surnameSuggestions.length > 0}
            aria-controls="worker-surname-suggestions"
          />
        </label>
        {showSurnameSuggestions && surnameSuggestions.length > 0 && (
          <ul className="worker-surname-suggestions" id="worker-surname-suggestions" role="listbox">
            {surnameSuggestions.map((suggestion) => (
              <li key={suggestion} role="presentation">
                <button
                  type="button"
                  role="option"
                  aria-selected={suggestion === surname}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => {
                    setSurname(suggestion);
                    setSurnameSuggestions([]);
                    setShowSurnameSuggestions(false);
                  }}
                >
                  {suggestion}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <label className="invoice-field">
        <span>Дата</span>
        <input
          type="date"
          value={date}
          onChange={(event) => setDate(event.target.value)}
          disabled={sendingReport || hasUnsavedCashOperations || hasSavingCashOperations || cashOperationsLoading}
        />
      </label>

      <section className="bunker-cash-operations">
        <strong>Операции по кассе за выбранную дату</strong>
        {cashOperationsLoading && <p className="placeholder">Загружаю операции кассы…</p>}
        {cashOperationsError && <div className="error">{cashOperationsError}</div>}
        {!cashOperationsLoading && !cashOperationsError && surname.trim() !== "" && cashOperations.length === 0 && (
          <p className="placeholder">За эту дату операций нет.</p>
        )}
        <div className="bunker-cash-operation-actions">
          <button
            type="button"
            className="fuels-filter-btn"
            disabled={!surname.trim() || cashOperationsLoading}
            onClick={() => addCashOperation("income")}
          >
            + Приход
          </button>
          <button
            type="button"
            className="fuels-filter-btn"
            disabled={!surname.trim() || cashOperationsLoading}
            onClick={() => addCashOperation("expense")}
          >
            + Расход
          </button>
        </div>
        {cashOperations.map((operation) => (
          <div className="bunker-cash-operation" key={operation.localID}>
            {operation.cashEntryID !== undefined && editCashOperationID !== operation.localID ? (
              <button
                type="button"
                className="bunker-cash-operation-summary"
                aria-expanded="false"
                onClick={() => setEditCashOperationID(operation.localID)}
              >
                <strong>{operation.entryType === "income" ? "Приход" : "Расход"} · {new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 }).format(Number(operation.amount.replace(",", ".")))} ₽</strong>
                {operation.description && <span>{operation.description}</span>}
                <small>Нажмите, чтобы редактировать</small>
              </button>
            ) : (
              <>
                {operation.cashEntryID !== undefined && (
                  <button
                    type="button"
                    className="bunker-cash-operation-summary"
                    aria-expanded="true"
                    disabled={operation.dirty || operation.saving}
                    onClick={() => setEditCashOperationID(null)}
                  >
                    <strong>{operation.entryType === "income" ? "Приход" : "Расход"} · {new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 }).format(Number(operation.amount.replace(",", ".")))} ₽</strong>
                    <small>Свернуть</small>
                  </button>
                )}
                <label className="invoice-field">
                  <span>Тип операции</span>
                  <select
                    value={operation.entryType}
                    onChange={(event) => updateCashOperation(operation.localID, { entryType: event.target.value as CashEntryType })}
                  >
                    <option value="income">Приход</option>
                    <option value="expense">Расход</option>
                  </select>
                </label>
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
            )}
            {operation.error && <div className="error">{operation.error}</div>}
            <div className="bunker-cash-operation-actions">
              {(operation.cashEntryID === undefined || operation.dirty) && (
                <button
                  type="button"
                  className="fuels-search-btn bunker-save-annotation"
                  disabled={operation.saving || Number(operation.amount.replace(",", ".")) <= 0 || (operation.cashEntryID !== undefined && !operation.dirty)}
                  onClick={() => void saveCashOperation(operation)}
                >
                  {operation.saving ? "Сохраняю…" : operation.cashEntryID === undefined ? "Добавить в кассу" : "Сохранить изменения"}
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
                  {selectedMethods.filter((kind) => kind !== "base" && kind !== "rsm").map((kind) => (
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
                          if (kind === "base" || kind === "rsm") return false;
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
          disabled={!surname.trim() || commentStatus === "loading" || commentStatus === "saving" || sendingReport}
          onChange={(event) => {
            setComments(event.target.value);
            setCommentStatus("unsaved");
            setCommentSaveError("");
            setReportError("");
            setReportSent(false);
          }}
          placeholder="Введите комментарий"
          maxLength={2000}
          rows={3}
        />
      </label>

      <div className="worker-comment-actions">
        <button
          type="button"
          className="fuels-filter-btn"
          disabled={!surname.trim() || commentStatus === "loading" || commentStatus === "saving" || sendingReport}
          onClick={() => void saveComments()}
        >
          {commentStatus === "saving" ? "Сохраняю…" : "Сохранить комментарий"}
        </button>
        <small className="worker-comment-status" aria-live="polite">
          {commentStatus === "loading" && "Загружаю сохранённый комментарий…"}
          {commentStatus === "unsaved" && "Есть несохранённые изменения"}
          {commentStatus === "saved" && "Комментарий сохранён"}
          {commentStatus === "error" && commentSaveError}
        </small>
      </div>

      {reportError && <div className="error">{reportError}</div>}
      {reportSent && <small className="bunker-report-success">Отчёт отправлен в группу MAX.</small>}
      <button
        type="button"
        className="fuels-search-btn"
        disabled={sendingReport || loading || cashOperationsLoading || cashOperationsError !== "" || commentStatus === "loading" || commentStatus === "saving" || !hasReportContent || hasUnsavedCashOperations || cashOperations.some((operation) => operation.dirty) || hasSavingCashOperations}
        onClick={() => void sendReport()}
      >
        {sendingReport ? "Отправляю…" : "Отправить"}
      </button>
    </div>
  );
}
