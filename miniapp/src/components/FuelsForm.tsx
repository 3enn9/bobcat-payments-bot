import { useState } from "react";
import type { FormEvent } from "react";
import {
  listFuels,
  splitFuel,
  updateFuelEquipment,
  type FuelEntry,
  type FuelKind,
} from "../api/fuels";

function money(value: number): string {
  return String(rublesOnly(value));
}

function rublesOnly(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return 0;
  }
  return Math.trunc(Math.round(value * 100) / 100);
}

function parseAmount(value: string): number {
  const n = Number(value.replace(",", ".").replace(/[^\d.]/g, "").trim());
  return rublesOnly(n);
}

function formatWhen(value: string): string {
  const m = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/);
  if (!m) {
    return "—";
  }
  return `${m[3]}.${m[2]}.${m[1].slice(2)} ${m[4]}:${m[5]}`;
}

function asFuelKind(value: string | undefined): FuelKind {
  return value === "petrol" || value === "dt" ? value : "";
}

function FuelKindToggle({
  value,
  disabled,
  onChange,
}: {
  value: FuelKind;
  disabled?: boolean;
  onChange: (kind: FuelKind) => void;
}) {
  return (
    <div className="fuels-kind">
      <button
        type="button"
        className={value === "petrol" ? "fuels-kind-btn active" : "fuels-kind-btn"}
        disabled={disabled}
        onClick={() => onChange("petrol")}
      >
        Бензин
      </button>
      <button
        type="button"
        className={value === "dt" ? "fuels-kind-btn active" : "fuels-kind-btn"}
        disabled={disabled}
        onClick={() => onChange("dt")}
      >
        ДТ
      </button>
    </div>
  );
}

type SplitRow = {
  key: string;
  equipment: string;
  amount: string;
};

function newSplitRow(equipment = "", amount = ""): SplitRow {
  return {
    key: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
    equipment,
    amount,
  };
}

export default function FuelsForm() {
  const [holderQuery, setHolderQuery] = useState("");
  const [appliedHolder, setAppliedHolder] = useState("");
  const [entries, setEntries] = useState<FuelEntry[]>([]);
  const [numbers, setNumbers] = useState<Record<number, string>>({});
  const [kinds, setKinds] = useState<Record<number, FuelKind>>({});
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [savingId, setSavingId] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [savedId, setSavedId] = useState<number | null>(null);
  const [splitId, setSplitId] = useState<number | null>(null);
  const [splitRows, setSplitRows] = useState<SplitRow[]>([]);
  const [splitSaving, setSplitSaving] = useState(false);

  async function reload(holder: string) {
    const items = await listFuels(holder);
    setEntries(items);
    const nextNumbers: Record<number, string> = {};
    const nextKinds: Record<number, FuelKind> = {};
    for (const item of items) {
      nextNumbers[item.id] = item.equipmentNumber ?? "";
      nextKinds[item.id] = asFuelKind(item.fuelKind);
    }
    setNumbers(nextNumbers);
    setKinds(nextKinds);
  }

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
    setSplitId(null);
    try {
      await reload(holder);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка загрузки");
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }

  async function saveEntry(id: number, nextNumber?: string, nextKind?: FuelKind) {
    const value = (nextNumber ?? numbers[id] ?? "").trim();
    const kind = nextKind ?? kinds[id] ?? "";
    const current = entries.find((item) => item.id === id);
    if (
      (current?.equipmentNumber ?? "") === value &&
      asFuelKind(current?.fuelKind) === kind
    ) {
      return;
    }
    setSavingId(id);
    setError("");
    try {
      await updateFuelEquipment(id, value, kind);
      setEntries((prev) =>
        prev.map((item) =>
          item.id === id
            ? { ...item, equipmentNumber: value, fuelKind: kind }
            : item,
        ),
      );
      setSavedId(id);
      window.setTimeout(() => {
        setSavedId((cur) => (cur === id ? null : cur));
      }, 1200);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка сохранения");
    } finally {
      setSavingId(null);
    }
  }

  function openSplit(item: FuelEntry) {
    setSplitId(item.id);
    setSplitRows([
      newSplitRow(item.equipmentNumber || "", ""),
      newSplitRow(),
    ]);
    setError("");
  }

  async function saveSplit(item: FuelEntry) {
    const parts = splitRows.map((row) => ({
      equipmentNumber: row.equipment.trim(),
      amount: parseAmount(row.amount),
    }));
    setSplitSaving(true);
    setError("");
    try {
      await splitFuel(item.id, parts);
      setSplitId(null);
      if (appliedHolder) {
        await reload(appliedHolder);
      }
      setSavedId(item.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка разделения");
    } finally {
      setSplitSaving(false);
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
            const splitting = splitId === item.id;
            const splitTotal = splitRows.reduce(
              (sum, row) => sum + parseAmount(row.amount),
              0,
            );
            const leftover = rublesOnly(item.amount) - splitTotal;
            return (
              <li key={item.id} className="fuels-item">
                <div className="fuels-item-top">
                  <time>{formatWhen(item.fueledAt)}</time>
                  <strong>{money(item.amount)}</strong>
                </div>
                <div className="fuels-holder">{item.holder || "—"}</div>
                {item.cardNumber ? (
                  <div className="fuels-card">Карта {item.cardNumber}</div>
                ) : null}
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
                    disabled={savingId === item.id || splitting}
                    onChange={(e) =>
                      setNumbers((prev) => ({
                        ...prev,
                        [item.id]: e.target.value,
                      }))
                    }
                    onBlur={() => void saveEntry(item.id)}
                  />
                </label>
                <div className="invoice-field">
                  <span>Вид</span>
                  <FuelKindToggle
                    value={kinds[item.id] ?? ""}
                    disabled={savingId === item.id || splitting}
                    onChange={(kind) => {
                      setKinds((prev) => ({ ...prev, [item.id]: kind }));
                      void saveEntry(item.id, undefined, kind);
                    }}
                  />
                </div>
                {savedId === item.id && !splitting && (
                  <small className="fuels-saved">Сохранено</small>
                )}

                {splitting ? (
                  <div className="fuels-split">
                    {splitRows.map((row, idx) => (
                      <div key={row.key} className="fuels-split-row">
                        <input
                          type="text"
                          inputMode="text"
                          autoCapitalize="characters"
                          autoCorrect="off"
                          spellCheck={false}
                          placeholder="Техника"
                          value={row.equipment}
                          onChange={(e) =>
                            setSplitRows((prev) =>
                              prev.map((itemRow) =>
                                itemRow.key === row.key
                                  ? { ...itemRow, equipment: e.target.value }
                                  : itemRow,
                              ),
                            )
                          }
                        />
                        <input
                          inputMode="numeric"
                          placeholder="Рубли"
                          value={row.amount}
                          onChange={(e) =>
                            setSplitRows((prev) =>
                              prev.map((itemRow) =>
                                itemRow.key === row.key
                                  ? { ...itemRow, amount: e.target.value }
                                  : itemRow,
                              ),
                            )
                          }
                        />
                        {splitRows.length > 2 && (
                          <button
                            type="button"
                            className="fuels-split-remove"
                            onClick={() =>
                              setSplitRows((prev) =>
                                prev.filter((_, i) => i !== idx),
                              )
                            }
                          >
                            ×
                          </button>
                        )}
                      </div>
                    ))}
                    <small>
                      {leftover === 0
                        ? "Суммы сходятся"
                        : leftover > 0
                          ? `Осталось распределить ${money(leftover)}`
                          : `На ${money(-leftover)} больше исходной суммы`}
                    </small>
                    <div className="fuels-split-actions">
                      <button
                        type="button"
                        className="fuels-search-btn"
                        onClick={() =>
                          setSplitRows((prev) => [...prev, newSplitRow()])
                        }
                      >
                        Ещё машина
                      </button>
                      <button
                        type="button"
                        className="fuels-search-btn"
                        disabled={
                          splitSaving ||
                          leftover !== 0 ||
                          splitRows.some(
                            (row) =>
                              !row.equipment.trim() || parseAmount(row.amount) <= 0,
                          )
                        }
                        onClick={() => void saveSplit(item)}
                      >
                        {splitSaving ? "…" : "Сохранить"}
                      </button>
                      <button
                        type="button"
                        className="fuels-search-btn"
                        onClick={() => setSplitId(null)}
                      >
                        Отмена
                      </button>
                    </div>
                  </div>
                ) : (
                  <button
                    type="button"
                    className="fuels-split-open"
                    onClick={() => openSplit(item)}
                  >
                    Разделить на машины
                  </button>
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
