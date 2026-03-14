import { useEffect, useState } from "react";
import { FileText, RefreshCw, AlertTriangle, CheckCircle } from "lucide-react";
import { getReports } from "@/lib/api";

interface ReportSnapshot {
  id: string;
  snapshot_at: string;
  metrics: Record<string, unknown>;
  alerts: unknown[];
}

export default function Reports() {
  const [lastReport, setLastReport] = useState<{
    status?: string;
    checkedAt?: string;
    alerts?: number;
    error?: string;
  } | null>(null);
  const [snapshots, setSnapshots] = useState<ReportSnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getReports();
      setLastReport(data.last_report ?? null);
      setSnapshots((data.snapshots ?? []) as ReportSnapshot[]);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Load failed");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  if (loading && !lastReport && snapshots.length === 0) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="electric-loader h-12 w-12 rounded-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <FileText className="h-5 w-5 text-[#9bc3ff]" />
          <h2 className="text-base font-semibold text-white">Reports</h2>
        </div>
        <button
          onClick={refresh}
          disabled={loading}
          className="electric-button flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Refresh
        </button>
      </div>

      {error && (
        <div className="electric-card p-4 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
          <AlertTriangle className="h-5 w-5 text-rose-400 flex-shrink-0" />
          <span className="text-sm text-rose-300">{error}</span>
        </div>
      )}

      {/* Last metrics report status */}
      {lastReport && (
        <div className="electric-card p-4">
          <h3 className="text-sm font-medium text-[#8bb9ff] mb-2">Last Report</h3>
          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span
              className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg ${
                lastReport.status === "ok"
                  ? "bg-emerald-500/20 text-emerald-300"
                  : lastReport.status === "error"
                  ? "bg-rose-500/20 text-rose-300"
                  : "bg-amber-500/20 text-amber-300"
              }`}
            >
              {lastReport.status === "ok" ? (
                <CheckCircle className="h-4 w-4" />
              ) : (
                <AlertTriangle className="h-4 w-4" />
              )}
              {lastReport.status ?? "unknown"}
            </span>
            {lastReport.checkedAt && (
              <span className="text-[#8aa8df]">
                Checked: {new Date(lastReport.checkedAt).toLocaleString()}
              </span>
            )}
            {lastReport.alerts !== undefined && lastReport.alerts > 0 && (
              <span className="text-amber-300">{lastReport.alerts} alert(s)</span>
            )}
            {lastReport.error && (
              <span className="text-rose-300">{lastReport.error}</span>
            )}
          </div>
        </div>
      )}

      {/* Metric snapshots */}
      <div className="electric-card overflow-hidden">
        <h3 className="text-sm font-medium text-[#8bb9ff] px-4 py-3 border-b border-[#18356f]">
          Metric Snapshots
        </h3>
        {snapshots.length === 0 ? (
          <p className="p-4 text-sm text-[#8aa8df]">No snapshots yet.</p>
        ) : (
          <div className="divide-y divide-[#18356f] max-h-[400px] overflow-y-auto">
            {snapshots.map((s) => (
              <div
                key={s.id}
                className="px-4 py-3 hover:bg-[#071328]/50 transition"
              >
                <div className="flex items-center justify-between text-sm">
                  <span className="text-[#9bc3ff] font-mono">{s.id}</span>
                  <span className="text-[#6b8fcc]">
                    {new Date(s.snapshot_at).toLocaleString()}
                  </span>
                </div>
                <div className="mt-2 flex flex-wrap gap-4 text-xs">
                  {s.metrics &&
                    Object.entries(s.metrics).map(([k, v]) => (
                      <span key={k}>
                        <span className="text-[#7ea5eb]">{k}:</span>{" "}
                        <span className="text-white">{String(v)}</span>
                      </span>
                    ))}
                </div>
                {Array.isArray(s.alerts) && s.alerts.length > 0 && (
                  <div className="mt-2 text-xs text-amber-300">
                    {s.alerts.length} alert(s)
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
