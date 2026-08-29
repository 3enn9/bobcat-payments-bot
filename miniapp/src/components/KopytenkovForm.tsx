import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { fetchEquipment, type EquipmentItem } from "../api/equipment";

type RowValues = {
  driver: string;
  description: string;
};

export default function KopytenkovForm() {
  const [equipment, setEquipment] = useState<EquipmentItem[]>([]);
  const [rows, setRows] = useState<Record<number, RowValues>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");
      try {
        const items = await fetchEquipment();
        if (cancelled) {
          return;
        }
        setEquipment(items);
        const initial: Record<number, RowValues> = {};
        for (const item of items) {
          initial[item.id] = { driver: "", description: "" };
        }
        setRows(initial);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Ошибка загрузки");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  function updateRow(id: number, field: keyof RowValues, value: string) {
    setRows((prev) => ({
      ...prev,
      [id]: {
        ...prev[id],
        [field]: value,
      },
    }));
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSuccess("");
    setSaving(true);

    const payload = equipment.map((item) => ({
      equipmentId: item.id,
      number: item.number,
      driver: rows[item.id]?.driver.trim() ?? "",
      description: rows[item.id]?.description.trim() ?? "",
    }));

    console.log("Kopytenkov submit (stub):", payload);
    setSuccess("Отправка пока не подключена — данные сохранены только для проверки.");
    setSaving(false);
  }

  return (
    <form className="kopytenkov-form" onSubmit={submit}>
      {loading && <p className="invoice-hint">Загрузка...</p>}
      {error && <div className="error">{error}</div>}

      {!loading && !error && (
        <div className="kopytenkov-table-wrap">
          <table className="kopytenkov-table">
            <thead>
              <tr>
                <th>№ техники</th>
                <th>Водитель</th>
                <th>Описание</th>
              </tr>
            </thead>
            <tbody>
              {equipment.map((item) => (
                <tr key={item.id}>
                  <td className="kopytenkov-number">{item.number}</td>
                  <td>
                    <input
                      value={rows[item.id]?.driver ?? ""}
                      disabled={saving}
                      placeholder="ФИО"
                      onChange={(event) =>
                        updateRow(item.id, "driver", event.target.value)
                      }
                    />
                  </td>
                  <td>
                    <input
                      value={rows[item.id]?.description ?? ""}
                      disabled={saving}
                      placeholder="Комментарий"
                      onChange={(event) =>
                        updateRow(item.id, "description", event.target.value)
                      }
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {success && <div className="invoice-success">{success}</div>}
      {!loading && !error && (
        <button type="submit" disabled={saving || equipment.length === 0}>
          {saving ? "Отправляем..." : "Отправить"}
        </button>
      )}
    </form>
  );
}
