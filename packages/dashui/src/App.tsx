import { useEffect, useState } from "react";
import {
  getStatus,
  getStrategies,
  getCost,
  postPause,
  postResume,
  postChat,
  type Status,
  type Strategy,
  type Cost,
} from "./api";
import { Header } from "./components/Header";
import { StatsGrid } from "./components/StatsGrid";
import { StrategiesList } from "./components/StrategiesList";
import { ChatPanel } from "./components/ChatPanel";
import { ControlPanel } from "./components/ControlPanel";
import { StrategyModal } from "./components/StrategyModal";

function App() {
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<Status | null>(null);
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [cost, setCost] = useState<Cost | null>(null);
  const [selectedStrategy, setSelectedStrategy] = useState<Strategy | null>(null);

  const refresh = async () => {
    try {
      const [s, strat, c] = await Promise.all([
        getStatus(),
        getStrategies(),
        getCost(),
      ]);
      setStatus(s);
      setStrategies(strat);
      setCost(c);
    } catch (e) {
      console.warn("Connection lost...", e);
    }
  };

  useEffect(() => {
    setTimeout(() => setLoading(false), 1200);
  }, []);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, 3000);
    return () => clearInterval(id);
  }, []);

  const handlePause = async () => {
    await postPause();
    refresh();
  };

  const handleResume = async () => {
    await postResume();
    refresh();
  };

  const handleChat = async (message: string) => {
    const data = await postChat(message);
    return data.response;
  };

  if (loading) {
    return (
      <div className="fixed inset-0 bg-black z-50 flex items-center justify-center">
        <p className="font-orbitron text-cyan-400 animate-pulse">
          INITIALIZING NEURAL INTERFACE...
        </p>
      </div>
    );
  }

  return (
    <main className="max-w-7xl mx-auto p-4 md:p-6 min-h-screen">
      <Header status={status} />
      <StatsGrid status={status} cost={cost} />
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <StrategiesList
            strategies={strategies}
            onSelect={setSelectedStrategy}
          />
          <ChatPanel onSend={handleChat} />
        </div>
        <div className="space-y-6">
          <ControlPanel onPause={handlePause} onResume={handleResume} />
          {cost && (
            <section className="glass rounded-lg p-4">
              <h2 className="font-orbitron text-lg text-cyan-400 mb-4">
                LLM COST
              </h2>
              <p className="text-cyan-600 text-sm">
                Today: <span>${cost.today_cost.toFixed(2)}</span>
              </p>
              <p className="text-cyan-600 text-sm">
                Total: <span>${cost.total_cost.toFixed(2)}</span>
              </p>
            </section>
          )}
        </div>
      </div>
      <StrategyModal
        strategy={selectedStrategy}
        onClose={() => setSelectedStrategy(null)}
      />
    </main>
  );
}

export default App;
