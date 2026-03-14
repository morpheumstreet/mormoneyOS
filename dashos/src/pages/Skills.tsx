import { useEffect, useState, useCallback, useRef } from "react";
import {
  Puzzle,
  Search,
  ChevronDown,
  ChevronRight,
  Package,
  Loader2,
  AlertTriangle,
  Trash2,
  Check,
  X,
  Download,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import {
  getSkills,
  getSkillsDiscovery,
  getSkillsRecommended,
  postSkillInstall,
  patchSkillActivate,
  patchSkillDeactivate,
  deleteSkill,
  type SkillItem,
  type DiscoveryResult,
} from "@/lib/api";

const PAGE_LIMIT = 20;

type Tab = "manage" | "discovery";

function isSearchResponse(
  r: { results?: DiscoveryResult[]; items?: DiscoveryResult[]; nextCursor?: string }
): r is { results: DiscoveryResult[] } {
  return "results" in r && Array.isArray(r.results);
}

function isListResponse(
  r: { results?: DiscoveryResult[]; items?: DiscoveryResult[]; nextCursor?: string }
): r is { items: DiscoveryResult[]; nextCursor?: string } {
  return "items" in r && Array.isArray(r.items);
}

export default function Skills() {
  const { hasWriteAccess } = useWalletAuth();
  const [tab, setTab] = useState<Tab>("manage");

  // Manage state
  const [installed, setInstalled] = useState<SkillItem[]>([]);
  const [manageSearch, setManageSearch] = useState("");
  const [manageLoading, setManageLoading] = useState(true);
  const [manageError, setManageError] = useState<string | null>(null);
  const [toggling, setToggling] = useState<Record<string, boolean>>({});
  const [deleting, setDeleting] = useState<Record<string, boolean>>({});

  // Discovery state
  const [discoverySearch, setDiscoverySearch] = useState("");
  const [discoveryItems, setDiscoveryItems] = useState<DiscoveryResult[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [discoveryLoading, setDiscoveryLoading] = useState(false);
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);
  const [installing, setInstalling] = useState<Record<string, boolean>>({});
  const [hasMore, setHasMore] = useState(true);
  const discoverySentinelRef = useRef<HTMLDivElement>(null);

  const isInstalledSlug = useCallback(
    (slug: string) =>
      installed.some(
        (s) => s.name.toLowerCase() === slug.toLowerCase()
      ),
    [installed]
  );

  const refreshInstalled = useCallback(() => {
    setManageLoading(true);
    getSkills()
      .then((res) => setInstalled(res.skills || []))
      .catch((e) => setManageError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setManageLoading(false));
  }, []);

  useEffect(() => {
    if (tab === "manage") refreshInstalled();
  }, [tab, refreshInstalled]);

  const loadDiscovery = useCallback(
    (cursor?: string | null, append = false) => {
      const isSearch = discoverySearch.trim().length > 0;
      setDiscoveryLoading(true);
      setDiscoveryError(null);
      getSkillsDiscovery({
        q: isSearch ? discoverySearch.trim() : undefined,
        limit: PAGE_LIMIT,
        cursor: cursor || undefined,
      })
        .then((res) => {
          if (isSearchResponse(res)) {
            setDiscoveryItems(append ? [] : res.results);
            setNextCursor(null);
            setHasMore(false);
          } else if (isListResponse(res)) {
            setDiscoveryItems((prev) =>
              append ? [...prev, ...res.items] : res.items
            );
            setNextCursor(res.nextCursor || null);
            setHasMore(!!res.nextCursor);
          } else {
            setDiscoveryItems([]);
            setNextCursor(null);
            setHasMore(false);
          }
        })
        .catch((e) => {
          setDiscoveryError(e instanceof Error ? e.message : "Load failed");
          setDiscoveryItems([]);
        })
        .finally(() => setDiscoveryLoading(false));
    },
    [discoverySearch]
  );

  useEffect(() => {
    if (tab !== "discovery") return;
    const t = setTimeout(() => {
      setDiscoveryItems([]);
      setNextCursor(null);
      loadDiscovery(null, false);
    }, discoverySearch ? 350 : 0);
    return () => clearTimeout(t);
  }, [tab, discoverySearch, loadDiscovery]);

  const loadMoreDiscovery = useCallback(() => {
    if (!nextCursor || discoveryLoading) return;
    loadDiscovery(nextCursor, true);
  }, [nextCursor, discoveryLoading, loadDiscovery]);

  useEffect(() => {
    if (tab !== "discovery" || !hasMore || discoveryLoading) return;
    const el = discoverySentinelRef.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) loadMoreDiscovery();
      },
      { rootMargin: "200px", threshold: 0.1 }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [tab, hasMore, discoveryLoading, loadMoreDiscovery]);

  const handleToggle = async (skill: SkillItem) => {
    if (!hasWriteAccess) return;
    const nextEnabled = !skill.enabled;
    setToggling((prev) => ({ ...prev, [skill.name]: true }));
    try {
      if (nextEnabled) {
        await patchSkillActivate(skill.name);
      } else {
        await patchSkillDeactivate(skill.name);
      }
      setInstalled((prev) =>
        prev.map((s) =>
          s.name === skill.name ? { ...s, enabled: nextEnabled } : s
        )
      );
    } catch (e) {
      setManageError(e instanceof Error ? e.message : "Update failed");
    } finally {
      setToggling((prev) => ({ ...prev, [skill.name]: false }));
    }
  };

  const handleDelete = async (skill: SkillItem) => {
    if (!hasWriteAccess) return;
    setDeleting((prev) => ({ ...prev, [skill.name]: true }));
    try {
      await deleteSkill(skill.name);
      setInstalled((prev) => prev.filter((s) => s.name !== skill.name));
    } catch (e) {
      setManageError(e instanceof Error ? e.message : "Delete failed");
    } finally {
      setDeleting((prev) => ({ ...prev, [skill.name]: false }));
    }
  };

  const handleInstall = async (item: DiscoveryResult) => {
    if (!hasWriteAccess) return;
    const slug = item.slug;
    setInstalling((prev) => ({ ...prev, [slug]: true }));
    try {
      await postSkillInstall({
        source: "clawhub",
        id: slug,
        name: item.displayName,
        description: item.summary,
      });
      refreshInstalled();
    } catch (e) {
      setDiscoveryError(e instanceof Error ? e.message : "Install failed");
    } finally {
      setInstalling((prev) => ({ ...prev, [slug]: false }));
    }
  };

  const filteredInstalled = installed.filter(
    (s) =>
      s.name.toLowerCase().includes(manageSearch.toLowerCase()) ||
      (s.description || "").toLowerCase().includes(manageSearch.toLowerCase())
  );


  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <div className="electric-icon h-10 w-10 rounded-xl flex items-center justify-center">
          <Puzzle className="h-5 w-5 text-[#9bc3ff]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Skills</h2>
          <p className="text-sm text-[#8aa8df]">
            Manage installed skills or discover new ones from ClawHub.
          </p>
        </div>
      </div>

      {!hasWriteAccess && (
        <div className="electric-card p-4 border-amber-500/30 bg-amber-950/20">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-amber-200">
                Write access required
              </p>
              <p className="text-sm text-amber-300/80 mt-1">
                Connect your wallet and sign to install, enable, or remove skills.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-[#1a3670]">
        <button
          type="button"
          onClick={() => setTab("manage")}
          className={[
            "group flex items-center gap-2 rounded-t-lg border px-4 py-2.5 text-sm font-medium transition-all",
            tab === "manage"
              ? "border-[#3a6de0] border-b-transparent bg-[#0b2f80]/55 text-white shadow-[0_0_20px_-10px_rgba(72,140,255,0.8)]"
              : "border-transparent text-[#9bb7eb] hover:border-[#294a8d] hover:bg-[#07132f]/80 hover:text-white",
          ].join(" ")}
        >
          <Package className="h-4 w-4" />
          Manage
        </button>
        <button
          type="button"
          onClick={() => setTab("discovery")}
          className={[
            "group flex items-center gap-2 rounded-t-lg border px-4 py-2.5 text-sm font-medium transition-all",
            tab === "discovery"
              ? "border-[#3a6de0] border-b-transparent bg-[#0b2f80]/55 text-white shadow-[0_0_20px_-10px_rgba(72,140,255,0.8)]"
              : "border-transparent text-[#9bb7eb] hover:border-[#294a8d] hover:bg-[#07132f]/80 hover:text-white",
          ].join(" ")}
        >
          <Search className="h-4 w-4" />
          Discovery
        </button>
      </div>

      {/* Manage tab */}
      {tab === "manage" && (
        <>
          {manageError && (
            <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
              <span className="text-sm text-rose-300">{manageError}</span>
            </div>
          )}

          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[#5f84cc]" />
            <input
              type="text"
              value={manageSearch}
              onChange={(e) => setManageSearch(e.target.value)}
              placeholder="Search installed skills..."
              className="w-full rounded-xl border border-[#1a3670] bg-[#071328]/80 pl-10 pr-4 py-2.5 text-sm text-white placeholder-[#5f84cc] focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:border-transparent"
            />
          </div>

          {manageLoading ? (
            <div className="flex h-64 items-center justify-center">
              <div className="electric-loader h-12 w-12 rounded-full" />
            </div>
          ) : filteredInstalled.length === 0 ? (
            <div className="electric-card p-8 text-center">
              <p className="text-sm text-[#8aa8df]">
                {manageSearch
                  ? "No installed skills match your search."
                  : "No skills installed yet. Go to Discovery to browse and install from ClawHub."}
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredInstalled.map((skill) => {
                const isToggling = !!toggling[skill.name];
                const isDeleting = !!deleting[skill.name];
                return (
                  <div
                    key={skill.name}
                    className="electric-card overflow-hidden"
                  >
                    <div className="p-4">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2 flex-wrap">
                            <h3 className="text-sm font-semibold text-white truncate">
                              {skill.name}
                            </h3>
                            <span
                              className={[
                                "inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-medium flex-shrink-0",
                                skill.enabled
                                  ? "bg-emerald-900/50 text-emerald-300 border border-emerald-700/50"
                                  : "bg-gray-800 text-gray-400 border border-gray-700",
                              ].join(" ")}
                            >
                              {skill.enabled ? (
                                <>
                                  <Check className="h-2.5 w-2.5" />
                                  Enabled
                                </>
                              ) : (
                                <>
                                  <X className="h-2.5 w-2.5" />
                                  Disabled
                                </>
                              )}
                            </span>
                            {skill.trusted && (
                              <span className="inline-flex px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-900/50 text-blue-300 border border-blue-700/50">
                                Trusted
                              </span>
                            )}
                          </div>
                          {skill.description && (
                            <p className="mt-1 text-xs text-[#8aa8df] line-clamp-2">
                              {skill.description}
                            </p>
                          )}
                          <p className="mt-1 text-[10px] text-[#5f84cc]">
                            {skill.source}
                          </p>
                        </div>
                      </div>
                      <div className="mt-3 flex items-center gap-2">
                        <button
                          type="button"
                          onClick={() => handleToggle(skill)}
                          disabled={!hasWriteAccess || isToggling}
                          className={[
                            "relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors duration-200",
                            "focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:ring-offset-2 focus:ring-offset-[#050d1f]",
                            "disabled:opacity-50 disabled:cursor-not-allowed",
                            skill.enabled ? "bg-[#2f8fff]/60" : "bg-[#1a3670]",
                          ].join(" ")}
                          role="switch"
                          aria-checked={skill.enabled}
                        >
                          <span
                            className={[
                              "inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform duration-200",
                              skill.enabled ? "translate-x-6" : "translate-x-1",
                            ].join(" ")}
                          />
                          {isToggling && (
                            <span className="absolute inset-0 flex items-center justify-center">
                              <Loader2 className="h-4 w-4 animate-spin text-[#9bc3ff]" />
                            </span>
                          )}
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDelete(skill)}
                          disabled={!hasWriteAccess || isDeleting}
                          className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium text-rose-300 hover:bg-rose-900/30 hover:text-rose-200 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                          {isDeleting ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <Trash2 className="h-3 w-3" />
                          )}
                          Remove
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}

      {/* Discovery tab */}
      {tab === "discovery" && (
        <>
          {discoveryError && (
            <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
              <span className="text-sm text-rose-300">{discoveryError}</span>
            </div>
          )}

          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[#5f84cc]" />
            <input
              type="text"
              value={discoverySearch}
              onChange={(e) => setDiscoverySearch(e.target.value)}
              placeholder="Search ClawHub skills..."
              className="w-full rounded-xl border border-[#1a3670] bg-[#071328]/80 pl-10 pr-4 py-2.5 text-sm text-white placeholder-[#5f84cc] focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:border-transparent"
            />
          </div>

          {discoveryLoading && discoveryItems.length === 0 ? (
            <div className="flex h-64 items-center justify-center">
              <div className="electric-loader h-12 w-12 rounded-full" />
            </div>
          ) : discoveryItems.length === 0 ? (
            <div className="electric-card p-8 text-center">
              <p className="text-sm text-[#8aa8df]">
                {discoverySearch
                  ? "No skills found. Try a different search."
                  : "No skills available from ClawHub yet."}
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {discoveryItems.map((item) => {
                const slug = item.slug;
                const alreadyInstalled = isInstalledSlug(slug);
                const isInstalling = !!installing[slug];
                return (
                  <div
                    key={slug}
                    className="electric-card overflow-hidden"
                  >
                    <div className="p-4">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2 flex-wrap">
                            <h3 className="text-sm font-semibold text-white truncate">
                              {item.displayName || item.slug}
                            </h3>
                            {alreadyInstalled ? (
                              <span className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-900/50 text-emerald-300 border border-emerald-700/50 flex-shrink-0">
                                <Check className="h-2.5 w-2.5" />
                                Installed
                              </span>
                            ) : (
                              <span className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-medium bg-gray-800 text-gray-400 border border-gray-700 flex-shrink-0">
                                Available
                              </span>
                            )}
                          </div>
                          {item.summary && (
                            <p className="mt-1 text-xs text-[#8aa8df] line-clamp-2">
                              {item.summary}
                            </p>
                          )}
                          {item.version && (
                            <p className="mt-1 text-[10px] text-[#5f84cc]">
                              v{item.version}
                            </p>
                          )}
                        </div>
                      </div>
                      {!alreadyInstalled && hasWriteAccess && (
                        <div className="mt-3">
                          <button
                            type="button"
                            onClick={() => handleInstall(item)}
                            disabled={isInstalling}
                            className="inline-flex items-center gap-2 rounded-lg px-3 py-2 text-xs font-medium electric-button text-white disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            {isInstalling ? (
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                            ) : (
                              <Download className="h-3.5 w-3.5" />
                            )}
                            Install
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {hasMore && (
            <div
              ref={discoverySentinelRef}
              className="flex h-16 items-center justify-center"
            >
              {discoveryLoading && (
                <Loader2 className="h-6 w-6 animate-spin text-[#5f84cc]" />
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
