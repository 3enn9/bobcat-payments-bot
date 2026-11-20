import { useEffect, useState } from "react";
import {
  listBunkerHaulMessages,
  saveBunkerHaulAnnotation,
  type BunkerHaulAnnotationKind,
  type BunkerHaulMessage,
} from "../api/bunkerHauls";

const annotationMethods: { kind: BunkerHaulAnnotationKind; label: string }[] = [
  { kind: "vector", label: "Вектор" },
  { kind: "dubki", label: "Дубки" },
  { kind: "cash", label: "Нал" },
];

function annotationLabel(message: BunkerHaulMessage): string {
  if (!message.annotationKind || message.annotationAmount === undefined) return "";
  const amount = new Intl.NumberFormat("ru-RU", {
    maximumFractionDigits: 3,
  }).format(message.annotationAmount);
  if (message.annotationKind === "vector") return `Вектор — вес: ${amount}`;
  if (message.annotationKind === "dubki") return `Дубки — сумма: ${amount} ₽`;
  return `Нал — сумма: ${amount} ₽`;
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
  const [selectedMethod, setSelectedMethod] = useState<BunkerHaulAnnotationKind | null>(null);
  const [annotationValue, setAnnotationValue] = useState("");
  const [savingAnnotation, setSavingAnnotation] = useState(false);
  const [annotationError, setAnnotationError] = useState("");

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
    setActiveMessageID((current) => (current === id ? null : id));
    setSelectedMethod(null);
    setAnnotationValue("");
    setAnnotationError("");
  };

  const saveAnnotation = async (message: BunkerHaulMessage) => {
    const value = Number(annotationValue.replace(",", "."));
    if (!selectedMethod || !Number.isFinite(value) || value <= 0) return;

    setSavingAnnotation(true);
    setAnnotationError("");
    try {
      await saveBunkerHaulAnnotation(message.id, selectedMethod, value);
      setMessages((current) =>
        current.map((item) =>
          item.id === message.id
            ? { ...item, annotationKind: selectedMethod, annotationAmount: value }
            : item,
        ),
      );
      setActiveMessageID(null);
      setSelectedMethod(null);
      setAnnotationValue("");
    } catch (err) {
      setAnnotationError(err instanceof Error ? err.message : "Не удалось сохранить отметку");
    } finally {
      setSavingAnnotation(false);
    }
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
        />
      </label>

      <label className="invoice-field">
        <span>Дата</span>
        <input
          type="date"
          value={date}
          onChange={(event) => setDate(event.target.value)}
        />
      </label>

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
                {message.annotationKind && (
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
                        className={`fuels-filter-btn${selectedMethod === method.kind ? " active" : ""}`}
                        onClick={() => {
                          setSelectedMethod(method.kind);
                          setAnnotationValue("");
                          setAnnotationError("");
                        }}
                      >
                        {method.label}
                      </button>
                    ))}
                  </div>
                  {selectedMethod && (
                    <>
                      <label className="invoice-field">
                        <span>{selectedMethod === "vector" ? "Вес" : "Сумма"}</span>
                        <input
                          type="number"
                          min="0.001"
                          step={selectedMethod === "vector" ? "0.001" : "0.01"}
                          inputMode="decimal"
                          value={annotationValue}
                          onChange={(event) => setAnnotationValue(event.target.value)}
                          placeholder={selectedMethod === "vector" ? "Введите вес" : "Введите сумму"}
                        />
                      </label>
                      {annotationError && <div className="error">{annotationError}</div>}
                      <button
                        type="button"
                        className="fuels-search-btn bunker-save-annotation"
                        disabled={savingAnnotation || !annotationValue || Number(annotationValue.replace(",", ".")) <= 0}
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
          onChange={(event) => setComments(event.target.value)}
          placeholder="Введите комментарий"
          maxLength={2000}
          rows={3}
        />
      </label>

      <button type="button" className="fuels-search-btn" disabled>
        Отправить
      </button>
      <small className="invoice-hint">Отправка пока не настроена.</small>
    </div>
  );
}
