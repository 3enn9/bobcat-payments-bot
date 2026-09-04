import { useState } from "react";
import GarageForm from "./components/GarageForm";
import WorkerCabinet from "./components/WorkerCabinet";
import WorkerRequests from "./components/WorkerRequests";
import InvoiceForm from "./components/InvoiceForm";
import PaymentMatchForm from "./components/PaymentMatchForm";
import DaysOffForm from "./components/DaysOffForm";
import KopytenkovForm from "./components/KopytenkovForm";
import CashForm from "./components/CashForm";

type Screen = "home" | "garage" | "worker" | "kopytenkov" | "kosenko" | "invoices" | "daysoff" | "cash";
type InvoiceTab = "create" | "match";

export default function App() {
  const [screen, setScreen] = useState<Screen>("home");
  const [invoiceTab, setInvoiceTab] = useState<InvoiceTab>("create");

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

          <WorkerCabinet />
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
          </div>

          <div className="requests-scroll">
            {invoiceTab === "create" ? <InvoiceForm /> : <PaymentMatchForm />}
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
          onClick={() => setScreen("worker")}
        >
          <span className="role-icon">👷</span>

          <span>
            <strong>Работник</strong>
            <small>Перейти в кабинет</small>
          </span>
        </button>

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("kopytenkov")}
        >
          <span className="role-icon">🚜</span>

          <span>
            <strong>Копытенков</strong>
            <small>Техника и водители</small>
          </span>
        </button>

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("kosenko")}
        >
          <span className="role-icon">📋</span>

          <span>
            <strong>Косенко</strong>
            <small>Заявки Рогатка</small>
          </span>
        </button>

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
