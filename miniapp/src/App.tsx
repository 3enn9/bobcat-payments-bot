import { useEffect, useState } from "react";
import GarageForm from "./components/GarageForm";
import WorkerMessages from "./components/WorkerMessages";
import WorkerRequests from "./components/WorkerRequests";
import InvoiceForm from "./components/InvoiceForm";
import PaymentMatchForm from "./components/PaymentMatchForm";
import UnpaidInvoicesTable from "./components/UnpaidInvoicesTable";
import InvoiceEmailManager from "./components/InvoiceEmailManager";
import DaysOffForm from "./components/DaysOffForm";
import KopytenkovForm from "./components/KopytenkovForm";
import CashForm from "./components/CashForm";
import FuelsForm from "./components/FuelsForm";
import { getMaxUser, logMiniappOpen } from "./max";

type Screen = "home" | "garage" | "fuels" | "worker" | "kopytenkov" | "kosenko" | "invoices" | "daysoff" | "cash";
type InvoiceTab = "create" | "match" | "table" | "email";
const invoiceOwnerMaxId = "242575328";

export default function App() {
  const [screen, setScreen] = useState<Screen>("home");
  const [invoiceTab, setInvoiceTab] = useState<InvoiceTab>("create");
  const [canSeeInvoices, setCanSeeInvoices] = useState(false);

  useEffect(() => {
    logMiniappOpen();

    const checkUser = () => {
      const user = getMaxUser();
      const userId = String(user?.id ?? user?.user_id ?? "").trim();
      if (!userId) {
        return false;
      }
      setCanSeeInvoices(userId === invoiceOwnerMaxId);
      return true;
    };

    if (checkUser()) {
      return;
    }

    let attempts = 0;
    const timer = window.setInterval(() => {
      attempts += 1;
      if (checkUser() || attempts >= 25) {
        window.clearInterval(timer);
      }
    }, 200);

    return () => window.clearInterval(timer);
  }, []);

  if (screen === "garage") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Гараж</h1>
          </div>

          <div className="requests-scroll">
            <GarageForm />
          </div>
        </div>
      </div>
    );
  }

  if (screen === "fuels") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Заправки</h1>
          </div>

          <div className="requests-scroll">
            <FuelsForm />
          </div>
        </div>
      </div>
    );
  }

  if (screen === "worker") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Работник</h1>
          </div>

          <div className="requests-scroll">
            <WorkerMessages />
          </div>
        </div>
      </div>
    );
  }

  if (screen === "kopytenkov") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Копытенков</h1>
          </div>

          <div className="requests-scroll">
            <KopytenkovForm />
          </div>
        </div>
      </div>
    );
  }

  if (screen === "kosenko") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Косенко</h1>
          </div>

          <WorkerRequests />
        </div>
      </div>
    );
  }

  if (screen === "invoices") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Счета</h1>
          </div>

          <div className="invoice-tabs">
            <button
              type="button"
              className={invoiceTab === "create" ? "tab active" : "tab"}
              onClick={() => setInvoiceTab("create")}
            >
              Создать
            </button>
            <button
              type="button"
              className={invoiceTab === "match" ? "tab active" : "tab"}
              onClick={() => setInvoiceTab("match")}
            >
              Сопоставить
            </button>
            <button
              type="button"
              className={invoiceTab === "table" ? "tab active" : "tab"}
              onClick={() => setInvoiceTab("table")}
            >
              Таблица
            </button>
            <button
              type="button"
              className={invoiceTab === "email" ? "tab active" : "tab"}
              onClick={() => setInvoiceTab("email")}
            >
              Почта
            </button>
          </div>

          <div className="requests-scroll">
            {invoiceTab === "create" ? (
              <InvoiceForm />
            ) : invoiceTab === "match" ? (
              <PaymentMatchForm />
            ) : invoiceTab === "table" ? (
              <UnpaidInvoicesTable />
            ) : (
              <InvoiceEmailManager />
            )}
          </div>
        </div>
      </div>
    );
  }

  if (screen === "daysoff") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Выходные</h1>
          </div>

          <div className="requests-scroll">
            <DaysOffForm />
          </div>
        </div>
      </div>
    );
  }

  if (screen === "cash") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Касса</h1>
          </div>

          <div className="requests-scroll">
            <CashForm />
          </div>
        </div>
      </div>
    );
  }

  return (
    <main className="home-screen">
      <div className="role-buttons">
        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("garage")}
        >
          <span className="role-icon">🔧</span>

          <span>
            <strong>Гараж</strong>
            <small>Учёт работ</small>
          </span>
        </button>

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("fuels")}
        >
          <span className="role-icon">⛽</span>

          <span>
            <strong>Заправки</strong>
            <small>РН Карт и номера техники</small>
          </span>
        </button>

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("worker")}
        >
          <span className="role-icon">👷</span>

          <span>
            <strong>Работник</strong>
            <small>Перейти в кабинет</small>
          </span>
        </button>

        {canSeeInvoices && (
          <button
            type="button"
            className="role-button"
            onClick={() => setScreen("invoices")}
          >
            <span className="role-icon">🧾</span>

            <span>
              <strong>Счета</strong>
              <small>Конструктор счетов</small>
            </span>
          </button>
        )}

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("daysoff")}
        >
          <span className="role-icon">🏖️</span>

          <span>
            <strong>Выходные</strong>
            <small>График отдыха</small>
          </span>
        </button>

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("cash")}
        >
          <span className="role-icon">💵</span>

          <span>
            <strong>Касса</strong>
            <small>Личные приходы и расходы</small>
          </span>
        </button>
      </div>
    </main>
  );
}
