import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";
import {
  listFuels,
  splitFuel,
  suggestFuelHolders,
  updateFuelEquipment,
  type FuelEntry,
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

function fuelLabel(kind: string | undefined): string {
  if (kind === "petrol") {
    return "Бензин";
  }
  if (kind === "dt") {
    return "ДТ";
  }
  return (kind ?? "").trim();
}

function splitHolders(holder: string | undefined, holders?: string[]): string[] {
  if (holders && holders.length > 0) {
    return holders.map((name) => name.trim()).filter(Boolean);
  }
  return (holder ?? "")
    .split(";")
    .map((name) => name.trim())
    .filter(Boolean);
}

function HolderPicker({
  names,
  value,
  disabled,
  onChange,
}: {
  names: string[];
  value: string;
  disabled?: boolean;
  onChange: (name: string) => void;
}) {
  return (
    <div className="fuels-holders">
      <span>Кто заправлял</span>
      <div className="fuels-holder-btns">
        {names.map((name) => (
          <button
            key={name}
            type="button"
            className={
              value === name ? "fuels-holder-btn active" : "fuels-holder-btn"
            }
            disabled={disabled}
            onClick={() => onChange(name)}
          >
            {name}
          </button>
        ))}
      </div>
    </div>
  );
}

type SplitRow = {
  key: string;
  equipment: string;
  amount: string;
  holder: string;
};

function newSplitRow(equipment = "", amount = "", holder = ""): SplitRow {
  return {
    key: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
    equipment,
    amount,
    holder,
  };
}

type EntryFilter = "all" | "empty";

function isEntryFilled(item: FuelEntry): boolean {
  if (!(item.equipmentNumber ?? "").trim()) {
    return false;
  }
  const names = splitHolders(item.holder, item.holders);
  if (names.length >= 2 && !(item.holderPicked ?? "").trim()) {
    return false;
  }
  return true;
}

export default function FuelsForm() {
  const [holderQuery, setHolderQuery] = useState("");
  const [appliedHolder, setAppliedHolder] = useState("");
  const [entries, setEntries] = useState<FuelEntry[]>([]);
  const [numbers, setNumbers] = useState<Record<number, string>>({});
  const [picked, setPicked] = useState<Record<number, string>>({});
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [entryFilter, setEntryFilter] = useState<EntryFilter>("all");
  const [savingId, setSavingId] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [savedId, setSavedId] = useState<number | null>(null);
  const [splitId, setSplitId] = useState<number | null>(null);
  const [splitRows, setSplitRows] = useState<SplitRow[]>([]);
  const [splitSaving, setSplitSaving] = useState(false);
  const [hints, setHints] = useState<string[]>([]);
  const [hintsOpen, setHintsOpen] = useState(false);
  const hintTimer = useRef<number | null>(null);

  async function reload(holder: string) {
    const items = await listFuels(holder);
    setEntries(items);
    const nextNumbers: Record<number, string> = {};
    const nextPicked: Record<number, string> = {};
    for (const item of items) {
      nextNumbers[item.id] = item.equipmentNumber ?? "";
      nextPicked[item.id] = item.holderPicked ?? "";
    }
    setNumbers(nextNumbers);
    setPicked(nextPicked);
  }

  useEffect(() => {
    if (hintTimer.current) {
      window.clearTimeout(hintTimer.current);
    }
    const q = holderQuery.trim();
    if (q.length < 2) {
      setHints([]);
      setHintsOpen(false);
      return;
    }
    hintTimer.current = window.setTimeout(() => {
      void suggestFuelHolders(q)
        .then((items) => {
          setHints(items);
          setHintsOpen(items.length > 0);
        })
        .catch(() => {
          setHints([]);
          setHintsOpen(false);
        });
    }, 200);
    return () => {
      if (hintTimer.current) {
        window.clearTimeout(hintTimer.current);
      }
    };
  }, [holderQuery]);

  async function searchHolder(holder: string) {
    if (!holder) {
      setError("Укажите носителя карты");
      return;
    }
    setAppliedHolder(holder);
    setLoading(true);
    setError("");
    setSearched(true);
    setEntryFilter("all");
    setSplitId(null);
    setHintsOpen(false);
    try {
      await reload(holder);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка загрузки");
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }

  async function saveEntry(id: number, nextNumber?: string, nextHolder?: string) {
    const value = (nextNumber ?? numbers[id] ?? "").trim();
    const holderName = nextHolder ?? picked[id] ?? "";
    const current = entries.find((item) => item.id === id);
    const names = splitHolders(current?.holder, current?.holders);
    if (names.length >= 2 && value && !holderName) {
      setError("Выберите, кто заправлял");
      return;
    }
    if (
      (current?.equipmentNumber ?? "") === value &&
      (current?.holderPicked ?? "") === holderName
    ) {
      return;
    }
    setSavingId(id);
    setError("");
    try {
      await updateFuelEquipment(id, value, holderName);
      setEntries((prev) =>
        prev.map((item) =>
          item.id === id
            ? { ...item, equipmentNumber: value, holderPicked: holderName }
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
    const chosen = picked[item.id] || item.holderPicked || "";
    setSplitId(item.id);
    setSplitRows([
      newSplitRow(item.equipmentNumber || "", "", chosen),
      newSplitRow("", "", chosen),
    ]);
    setError("");
  }

  async function saveSplit(item: FuelEntry) {
    const names = splitHolders(item.holder, item.holders);
    const dual = names.length >= 2;
    const parts = splitRows.map((row) => ({
      equipmentNumber: row.equipment.trim(),
      amount: parseAmount(row.amount),
      holder: dual ? row.holder : "",
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
      <form
        className="fuels-search"
        onSubmit={(e: FormEvent) => {
          e.preventDefault();
          void searchHolder(holderQuery.trim());
        }}
      >
        <label className="invoice-field">
          <span>Носитель карты</span>
          <div className="autocomplete">
            <input
              value={holderQuery}
              placeholder="Фамилия или имя на карте"
              autoComplete="off"
              onChange={(e) => setHolderQuery(e.target.value)}
              onFocus={() => {
                if (hints.length > 0) {
                  setHintsOpen(true);
                }
              }}
              onBlur={() => {
                window.setTimeout(() => setHintsOpen(false), 150);
              }}
            />
            {hintsOpen && hints.length > 0 && (
              <ul className="autocomplete-list">
                {hints.map((name) => (
                  <li key={name}>
                    <button
                      type="button"
                      onMouseDown={(event) => event.preventDefault()}
                      onClick={() => {
                        setHolderQuery(name);
                        setHintsOpen(false);
                        void searchHolder(name);
                      }}
                    >
                      {name}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
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
        <div className="fuels-filter">
          <button
            type="button"
            className={
              entryFilter === "all" ? "fuels-filter-btn active" : "fuels-filter-btn"
            }
            onClick={() => setEntryFilter("all")}
          >
            Все
          </button>
          <button
            type="button"
            className={
              entryFilter === "empty"
                ? "fuels-filter-btn active"
                : "fuels-filter-btn"
            }
            onClick={() => setEntryFilter("empty")}
          >
            Пустые
          </button>
        </div>
      )}

      {entries.length > 0 &&
        entryFilter === "empty" &&
        entries.every((item) => isEntryFilled(item)) && (
          <p className="invoice-hint">Нет пустых заправок</p>
        )}

      {entries.length > 0 && (
        <ul className="fuels-list">
          {entries
            .filter((item) => entryFilter === "all" || !isEntryFilled(item))
            .map((item) => {
            const splitting = splitId === item.id;
            const names = splitHolders(item.holder, item.holders);
            const dual = names.length >= 2;
            const chosen = picked[item.id] ?? "";
            const filled = isEntryFilled(item);
            const splitTotal = splitRows.reduce(
              (sum, row) => sum + parseAmount(row.amount),
              0,
            );
            const leftover = rublesOnly(item.amount) - splitTotal;
            return (
              <li
                key={item.id}
                className={
                  filled ? "fuels-item fuels-item-filled" : "fuels-item fuels-item-empty"
                }
              >
                <div className="fuels-item-top">
                  <time>{formatWhen(item.fueledAt)}</time>
                  <strong>{money(item.amount)}</strong>
                </div>
                <div className="fuels-meta">
                  {!dual && <span>{item.holder || "—"}</span>}
                  {dual && chosen ? <span>{chosen}</span> : null}
                  {fuelLabel(item.fuelKind) ? (
                    <span>{fuelLabel(item.fuelKind)}</span>
                  ) : null}
                  {item.cardNumber ? <span>карта {item.cardNumber}</span> : null}
                </div>
                {dual && !splitting && (
                  <HolderPicker
                    names={names}
                    value={chosen}
                    disabled={savingId === item.id}
                    onChange={(name) => {
                      setPicked((prev) => ({ ...prev, [item.id]: name }));
                      void saveEntry(item.id, undefined, name);
                    }}
                  />
                )}
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
                {savedId === item.id && !splitting && (
                  <small className="fuels-saved">Сохранено</small>
                )}

                {splitting ? (
                  <div className="fuels-split">
                    {splitRows.map((row, idx) => (
                      <div key={row.key} className="fuels-split-block">
                        <div className="fuels-split-row">
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
                        {dual && (
                          <HolderPicker
                            names={names}
                            value={row.holder}
                            onChange={(name) =>
                              setSplitRows((prev) =>
                                prev.map((itemRow) =>
                                  itemRow.key === row.key
                                    ? { ...itemRow, holder: name }
                                    : itemRow,
                                ),
                              )
                            }
                          />
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
                          setSplitRows((prev) => [
                            ...prev,
                            newSplitRow("", "", chosen),
                          ])
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
                              !row.equipment.trim() ||
                              parseAmount(row.amount) <= 0 ||
                              (dual && !row.holder),
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
