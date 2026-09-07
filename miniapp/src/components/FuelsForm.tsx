import { useState } from "react";
import type { FormEvent } from "react";
import { listFuels, updateFuelEquipment, type FuelEntry } from "../api/fuels";

function money(value: number): string {
  const n = Math.round(value * 100) / 100;
  if (Number.isInteger(n)) {
    return String(n);
  }
  return String(n).replace(".", ",");
}

function formatWhen(value: string): string {
  const m = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/);
  if (!m) {
    return "—";
  }
  return `${m[3]}.${m[2]}.${m[1].slice(2)} ${m[4]}:${m[5]}`;
}

export default function FuelsForm() {
  const [holderQuery, setHolderQuery] = useState("");
  const [appliedHolder, setAppliedHolder] = useState("");
  const [entries, setEntries] = useState<FuelEntry[]>([]);
  const [numbers, setNumbers] = useState<Record<number, string>>({});
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [savingId, setSavingId] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [savedId, setSavedId] = useState<number | null>(null);

  async function search(event: FormEvent) {
    event.preventDefault();
    const holder = holderQuery.trim();
    if (!holder) {
      setError("Укажите носителя карты");
      return;
    }
    setAppliedHolder(holder);
    setLoading(true);
    setError("");
    setSearched(true);
    try {
      const items = await listFuels(holder);
      setEntries(items);
      const next: Record<number, string> = {};
      for (const item of items) {
        next[item.id] = item.equipmentNumber ?? "";
      }
      setNumbers(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка загрузки");
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }

  async function saveNumber(id: number) {
    const value = (numbers[id] ?? "").trim();
    if (!value) {
      return;
    }
    setSavingId(id);
    setError("");
    try {
      await updateFuelEquipment(id, value);
      setEntries((prev) =>
        prev.map((item) =>
          item.id === id ? { ...item, equipmentNumber: value } : item,
        ),
      );
      setSavedId(id);
      window.setTimeout(() => {
        setSavedId((current) => (current === id ? null : current));
      }, 1200);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка сохранения");
    } finally {
      setSavingId(null);
    }
  }

  return (
    <div className="fuels-form">
      <form className="fuels-search" onSubmit={(e) => void search(e)}>
        <label className="invoice-field">
          <span>Носитель карты</span>
          <input
            value={holderQuery}
            placeholder="Фамилия или имя на карте"
            onChange={(e) => setHolderQuery(e.target.value)}
          />
        </label>
        <button type="submit" className="fuels-search-btn" disabled={loading}>
          {loading ? "…" : "Найти"}
        </button>
      </form>

      {loading && <p className="invoice-hint">Загрузка…</p>}

      {searched && !loading && entries.length === 0 && !error && (
        <p className="invoice-hint">
          Нет заправок за 30 дней для «{appliedHolder}»
        </p>
      )}

      {entries.length > 0 && (
        <ul className="fuels-list">
          {entries.map((item) => {
            const filled = (item.equipmentNumber || "").trim() !== "";
            return (
              <li key={item.id} className="fuels-item">
                <div className="fuels-item-top">
                  <time>{formatWhen(item.fueledAt)}</time>
                  <strong>{money(item.amount)}</strong>
                </div>
                <div className="fuels-holder">{item.holder || "—"}</div>
                {filled ? (
                  <div className="fuels-eq">Техника {item.equipmentNumber}</div>
                ) : (
                  <label className="invoice-field">
                    <span>Номер техники</span>
                    <input
                      type="text"
                      inputMode="text"
                      autoCapitalize="characters"
                      autoCorrect="off"
                      spellCheck={false}
                      value={numbers[item.id] ?? ""}
                      placeholder="Буквы и цифры"
                      disabled={savingId === item.id}
                      onChange={(e) =>
                        setNumbers((prev) => ({
                          ...prev,
                          [item.id]: e.target.value,
                        }))
                      }
                      onBlur={() => void saveNumber(item.id)}
                    />
                  </label>
                )}
                {savedId === item.id && (
                  <small className="fuels-saved">Сохранено</small>
                )}
              </li>
            );
          })}
        </ul>
      )}

      {error && <p className="error">{error}</p>}
    </div>
  );
}
