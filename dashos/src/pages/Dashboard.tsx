import { useEffect, useState } from "react";
import ReactMarkdown from "react-markdown";
import {
  Cpu,
  DollarSign,
  Shield,
  Activity,
  ChevronDown,
  Sparkles,
  Send,
} from "lucide-react";
import {
  getStatus,
  getStrategies,
  getCost,
  getRisk,
  postPause,
  postResume,
  postChat,
} from "@/lib/api";

interface Status {
  agent_state?: string;
  today_pnl?: number;
  paused?: boolean;
  tick?: number;
  name?: string;
  address?: string;
  chain?: string;
}

interface Strategy {
  name?: string;
  risk_level?: string;
}

type SectionKey = "strategies" | "control" | "chat";

function formatAddr(addr: string): string {
  if (!addr) return "—";
  return addr.length > 12 ? `${addr.slice(0, 6)}…${addr.slice(-4)}` : addr;
}

export default function Dashboard() {
  const [status, setStatus] = useState<Status | null>(null);
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [cost, setCost] = useState<{ today_cost?: number } | null>(null);
  const [risk, setRisk] = useState<{ risk_level?: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState(0);
  const [sections, setSections] = useState<Record<SectionKey, boolean>>({
    strategies: true,
    control: true,
    chat: true,
  });
  const [chatInput, setChatInput] = useState("");
  const [chatMessages, setChatMessages] = useState<string[]>([
    "System is online. Web dashboard active.",
  ]);
  const [chatSending, setChatSending] = useState(false);

  const refresh = async () => {
    try {
      const [s, strat, c, r] = await Promise.all([
        getStatus(),
        getStrategies(),
        getCost(),
        getRisk(),
      ]);
      setStatus(s as Status);
      setStrategies((strat || []) as Strategy[]);
      setCost(c);
      setRisk(r);
      setLastUpdated(Date.now());
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Connection failed");
    }
  };

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, 5000);
    return () => clearInterval(id);
  }, []);

  const toggleSection = (k: SectionKey) => {
    setSections((prev) => ({ ...prev, [k]: !prev[k] }));
  };

  const handlePause = async () => {
    try {
      await postPause();
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Pause failed");
    }
  };

  const handleResume = async () => {
    try {
      await postResume();
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Resume failed");
    }
  };

  const handleSendChat = async () => {
    const msg = chatInput.trim();
    if (!msg) return;
    setChatInput("");
    setChatMessages((prev) => [...prev, `You: ${msg}`]);
    setChatSending(true);
    try {
      const res = await postChat(msg);
      setChatMessages((prev) => [
        ...prev,
        res.response || "(No response)",
      ]);
    } catch (e) {
      setChatMessages((prev) => [
        ...prev,
        `Error: ${e instanceof Error ? e.message : "Request failed"}`,
      ]);
    } finally {
      setChatSending(false);
    }
  };

  const updatedText =
    lastUpdated === 0
      ? ""
      : Math.floor((Date.now() - lastUpdated) / 1000) < 5
      ? "Updated just now"
      : `Updated ${Math.floor((Date.now() - lastUpdated) / 1000)}s ago`;

  if (error && !status) {
    return (
      <div className="electric-card p-5 text-rose-200">
        <h2 className="text-lg font-semibold text-rose-100">Dashboard load failed</h2>
        <p className="mt-2 text-sm text-rose-200/90">{error}</p>
        <button
          onClick={refresh}
          className="mt-3 electric-button px-4 py-2 rounded-lg text-sm"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-5 md:space-y-6">
      <section className="hero-panel motion-rise">
        <div className="relative z-10 flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.22em] text-[#8fb8ff]">
              MoneyClaw Command Center
            </p>
            <h1 className="mt-2 text-2xl font-semibold tracking-[0.03em] text-white md:text-3xl">
              Working Task Dashboard
            </h1>
            <p className="mt-2 max-w-2xl text-sm text-[#b3cbf8] md:text-base">
              Agent status, strategies, control, and comm link in a single surface.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <span
              className={`status-pill ${
                status?.paused ? "border-amber-500/60" : "border-emerald-500/60"
              }`}
            >
              <Sparkles className="h-3.5 w-3.5" />
              {status?.paused ? "PAUSED" : "SYSTEM ONLINE"}
            </span>
            {status?.tick !== undefined && (
              <span className="status-pill">Tick #{status.tick}</span>
            )}
          </div>
        </div>
      </section>

      {/* Identity bar */}
      <div className="electric-card px-4 py-3 flex flex-wrap gap-4 text-sm">
        <span>
          <span className="text-[#7ea5eb]">Agent:</span>{" "}
          <span className="text-white font-medium">{status?.name || "—"}</span>
        </span>
        <span>
          <span className="text-[#7ea5eb]">Wallet:</span>{" "}
          <span className="font-mono text-[#9bc3ff]">
            {formatAddr(status?.address || "")}
          </span>
        </span>
        <span>
          <span className="text-[#7ea5eb]">Chain:</span>{" "}
          <span className="text-white">{status?.chain || "—"}</span>
        </span>
        {updatedText && (
          <span className="text-[#6b8fcc] ml-auto">{updatedText}</span>
        )}
      </div>

      {/* Metrics grid */}
      <section className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <article className="electric-card motion-rise motion-delay-1 p-4">
          <div className="metric-head">
            <Cpu className="h-4 w-4" />
            <span>Agent State</span>
          </div>
          <p className="metric-value mt-3">{status?.agent_state || "—"}</p>
        </article>
        <article className="electric-card motion-rise motion-delay-2 p-4">
          <div className="metric-head">
            <DollarSign className="h-4 w-4" />
            <span>P&L Today</span>
          </div>
          <p className="metric-value mt-3">
            ${(status?.today_pnl ?? 0).toFixed(2)}
          </p>
        </article>
        <article className="electric-card motion-rise motion-delay-3 p-4">
          <div className="metric-head">
            <Shield className="h-4 w-4" />
            <span>Risk Level</span>
          </div>
          <p className="metric-value mt-3">{risk?.risk_level || "LOW"}</p>
        </article>
        <article className="electric-card motion-rise motion-delay-4 p-4">
          <div className="metric-head">
            <Activity className="h-4 w-4" />
            <span>LLM Cost Today</span>
          </div>
          <p className="metric-value mt-3">
            ${(cost?.today_cost ?? 0).toFixed(2)}
          </p>
        </article>
      </section>

      {/* Active Strategies */}
      <section className="electric-card motion-rise">
        <button
          type="button"
          onClick={() => toggleSection("strategies")}
          className="group flex w-full items-center justify-between gap-4 rounded-t-xl px-4 py-4 text-left md:px-5"
        >
          <div className="flex items-center gap-3">
            <div className="electric-icon h-10 w-10 rounded-xl">
              <Activity className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-base font-semibold text-white">
                Active Strategies
              </h2>
              <p className="text-xs uppercase tracking-[0.13em] text-[#7ea5eb]">
                Skills and risk levels
              </p>
            </div>
          </div>
          <ChevronDown
            className={`h-5 w-5 text-[#7ea5eb] transition-transform duration-300 ${
              sections.strategies ? "rotate-180" : ""
            }`}
          />
        </button>
        <div
          className={`grid overflow-hidden transition-all duration-300 ${
            sections.strategies ? "grid-rows-[1fr]" : "grid-rows-[0fr]"
          }`}
        >
          <div className="min-h-0 border-t border-[#18356f] px-4 pb-4 pt-4 md:px-5">
            {strategies.length === 0 ? (
              <p className="text-sm text-[#8aa8df]">No strategies configured.</p>
            ) : (
              <div className="space-y-2">
                {strategies.map((s) => (
                  <div
                    key={s.name || ""}
                    className="flex items-center justify-between rounded-xl border border-[#1d3770] bg-[#05112c]/90 px-3 py-2.5"
                  >
                    <span className="text-sm font-medium text-white">
                      {s.name || "—"}
                    </span>
                    <span className="text-xs text-[#8baee7]">
                      {s.risk_level || "—"}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </section>

      {/* Control */}
      <section className="electric-card motion-rise">
        <button
          type="button"
          onClick={() => toggleSection("control")}
          className="group flex w-full items-center justify-between gap-4 rounded-t-xl px-4 py-4 text-left md:px-5"
        >
          <h2 className="text-base font-semibold text-white">Control</h2>
          <ChevronDown
            className={`h-5 w-5 text-[#7ea5eb] transition-transform duration-300 ${
              sections.control ? "rotate-180" : ""
            }`}
          />
        </button>
        <div
          className={`grid overflow-hidden transition-all duration-300 ${
            sections.control ? "grid-rows-[1fr]" : "grid-rows-[0fr]"
          }`}
        >
          <div className="min-h-0 border-t border-[#18356f] px-4 pb-4 pt-4 md:px-5 flex gap-3">
            <button
              onClick={handlePause}
              className="electric-button px-4 py-2 rounded-lg text-sm font-medium"
            >
              Pause
            </button>
            <button
              onClick={handleResume}
              className="rounded-lg border border-[#2c4e97] bg-[#0a1b3f]/60 px-4 py-2 text-sm font-medium text-[#8bb9ff] hover:border-[#4f83ff] hover:text-white transition"
            >
              Resume
            </button>
            <button
              onClick={refresh}
              className="rounded-lg border border-[#2c4e97] bg-[#0a1b3f]/60 px-4 py-2 text-sm font-medium text-[#8bb9ff] hover:border-[#4f83ff] hover:text-white transition"
            >
              Refresh
            </button>
          </div>
        </div>
      </section>

      {/* Agent Comm Link */}
      <section className="electric-card motion-rise">
        <button
          type="button"
          onClick={() => toggleSection("chat")}
          className="group flex w-full items-center justify-between gap-4 rounded-t-xl px-4 py-4 text-left md:px-5"
        >
          <div className="flex items-center gap-3">
            <Send className="h-5 w-5 text-[#9bc3ff]" />
            <h2 className="text-base font-semibold text-white">
              Agent Comm Link
            </h2>
          </div>
          <ChevronDown
            className={`h-5 w-5 text-[#7ea5eb] transition-transform duration-300 ${
              sections.chat ? "rotate-180" : ""
            }`}
          />
        </button>
        <div
          className={`grid overflow-hidden transition-all duration-300 ${
            sections.chat ? "grid-rows-[1fr]" : "grid-rows-[0fr]"
          }`}
        >
          <div className="min-h-0 border-t border-[#18356f] px-4 pb-4 pt-4 md:px-5 space-y-3">
            <div className="min-h-[80px] max-h-48 overflow-y-auto space-y-1 text-sm text-[#94a3b8] prose prose-invert prose-sm max-w-none prose-p:my-1 prose-ul:my-1 prose-ol:my-1 prose-li:my-0">
              {chatMessages.map((m, i) => (
                <div key={i} className="[&>*:last-child]:mb-0">
                  <ReactMarkdown>{m}</ReactMarkdown>
                </div>
              ))}
            </div>
            <div className="flex gap-2">
              <input
                type="text"
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSendChat()}
                placeholder="Ask the agent..."
                className="flex-1 rounded-xl border border-[#29509c] bg-[#071228]/90 px-4 py-2.5 text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none"
              />
              <button
                onClick={handleSendChat}
                disabled={chatSending}
                className="electric-button px-4 py-2.5 rounded-xl font-medium disabled:opacity-50"
              >
                {chatSending ? "…" : "Send"}
              </button>
            </div>
          </div>
        </div>
      </section>

      {error && (
        <div className="fixed bottom-4 right-4 px-4 py-2 rounded-lg bg-rose-600/90 text-white text-sm z-50 animate-fade-in">
          {error}
        </div>
      )}
    </div>
  );
}
