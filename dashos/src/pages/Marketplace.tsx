import { useEffect, useState, useCallback } from "react";
import {
  Store,
  Search,
  Package,
  Loader2,
  AlertTriangle,
  Download,
  Shield,
  Sparkles,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import {
  getMarketplaceSearch,
  getMarketplaceMySkills,
  postMarketplaceInstall,
  type MarketplaceSkill,
} from "@/lib/api";

type Tab = "discover" | "my-skills";

export default function Marketplace() {
  const { hasWriteAccess } = useWalletAuth();
  const [tab, setTab] = useState<Tab>("discover");

  // Discover state
  const [searchQuery, setSearchQuery] = useState("");
  const [discoverSkills, setDiscoverSkills] = useState<MarketplaceSkill[]>([]);
  const [discoverLoading, setDiscoverLoading] = useState(false);
  const [discoverError, setDiscoverError] = useState<string | null>(null);
  const [installing, setInstalling] = useState<Record<string, boolean>>({});

  // My Skills state
  const [mySkills, setMySkills] = useState<MarketplaceSkill[]>([]);
  const [mySkillsLoading, setMySkillsLoading] = useState(false);
  const [mySkillsError, setMySkillsError] = useState<string | null>(null);

  const loadDiscover = useCallback(() => {
    setDiscoverLoading(true);
    setDiscoverError(null);
    getMarketplaceSearch({ q: searchQuery.trim() || undefined })
      .then((res) => setDiscoverSkills(res.skills || []))
      .catch((e) => setDiscoverError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setDiscoverLoading(false));
  }, [searchQuery]);

  const loadMySkills = useCallback(() => {
    setMySkillsLoading(true);
    setMySkillsError(null);
    getMarketplaceMySkills()
      .then((res) => setMySkills(res.skills || []))
      .catch((e) => setMySkillsError(e instanceof Error ? e.message : "Load failed"))
      .finally(() => setMySkillsLoading(false));
  }, []);

  useEffect(() => {
    if (tab === "discover") {
      const t = setTimeout(loadDiscover, searchQuery ? 350 : 0);
      return () => clearTimeout(t);
    }
  }, [tab, searchQuery, loadDiscover]);

  useEffect(() => {
    if (tab === "my-skills") loadMySkills();
  }, [tab, loadMySkills]);

  const handleInstall = async (skill: MarketplaceSkill) => {
    if (!hasWriteAccess) return;
    setInstalling((prev) => ({ ...prev, [skill.id]: true }));
    try {
      await postMarketplaceInstall({
        skill_id: skill.id,
        agent_card_sig: "signed", // Placeholder; real flow would use wallet sign
      });
      loadMySkills();
    } catch (e) {
      setDiscoverError(e instanceof Error ? e.message : "Install failed");
    } finally {
      setInstalling((prev) => ({ ...prev, [skill.id]: false }));
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <div className="electric-icon h-10 w-10 rounded-xl flex items-center justify-center">
          <Store className="h-5 w-5 text-[#9bc3ff]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Mormaegis Marketplace</h2>
          <p className="text-sm text-[#8aa8df]">
            Discover skills from ClawHub, install with MORM rewards, and manage your published skills.
          </p>
        </div>
      </div>

      {!hasWriteAccess && (
        <div className="electric-card p-4 border-amber-500/30 bg-amber-950/20">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-amber-200">Write access required</p>
              <p className="text-sm text-amber-300/80 mt-1">
                Connect your wallet and sign to install skills and claim MORM rewards.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-[#1a3670]">
        <button
          type="button"
          onClick={() => setTab("discover")}
          className={[
            "group flex items-center gap-2 rounded-t-lg border px-4 py-2.5 text-sm font-medium transition-all",
            tab === "discover"
              ? "border-[#3a6de0] border-b-transparent bg-[#0b2f80]/55 text-white shadow-[0_0_20px_-10px_rgba(72,140,255,0.8)]"
              : "border-transparent text-[#9bb7eb] hover:border-[#294a8d] hover:bg-[#07132f]/80 hover:text-white",
          ].join(" ")}
        >
          <Search className="h-4 w-4" />
          Discover
        </button>
        <button
          type="button"
          onClick={() => setTab("my-skills")}
          className={[
            "group flex items-center gap-2 rounded-t-lg border px-4 py-2.5 text-sm font-medium transition-all",
            tab === "my-skills"
              ? "border-[#3a6de0] border-b-transparent bg-[#0b2f80]/55 text-white shadow-[0_0_20px_-10px_rgba(72,140,255,0.8)]"
              : "border-transparent text-[#9bb7eb] hover:border-[#294a8d] hover:bg-[#07132f]/80 hover:text-white",
          ].join(" ")}
        >
          <Package className="h-4 w-4" />
          My Skills
        </button>
      </div>

      {/* Discover tab */}
      {tab === "discover" && (
        <>
          {discoverError && (
            <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
              <span className="text-sm text-rose-300">{discoverError}</span>
            </div>
          )}

          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[#5f84cc]" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search ClawHub skills..."
              className="w-full rounded-xl border border-[#1a3670] bg-[#071328]/80 pl-10 pr-4 py-2.5 text-sm text-white placeholder-[#5f84cc] focus:outline-none focus:ring-2 focus:ring-[#4f83ff] focus:border-transparent"
            />
          </div>

          {discoverLoading ? (
            <div className="flex h-64 items-center justify-center">
              <div className="electric-loader h-12 w-12 rounded-full" />
            </div>
          ) : discoverSkills.length === 0 ? (
            <div className="electric-card p-8 text-center">
              <p className="text-sm text-[#8aa8df]">
                {searchQuery
                  ? "No skills match your search. Try a different query."
                  : "Enter a search term to discover skills from ClawHub."}
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {discoverSkills.map((skill) => {
                const isInstalling = !!installing[skill.id];
                return (
                  <div key={skill.id} className="electric-card overflow-hidden">
                    <div className="p-4">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2 flex-wrap">
                            <h3 className="text-sm font-semibold text-white truncate">{skill.name}</h3>
                            {skill.badges?.map((b) => (
                              <span
                                key={b}
                                className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-900/50 text-blue-300 border border-blue-700/50"
                              >
                                {b}
                              </span>
                            ))}
                          </div>
                          {skill.description && (
                            <p className="mt-1 text-xs text-[#8aa8df] line-clamp-2">{skill.description}</p>
                          )}
                          {skill.price_morm != null && skill.price_morm > 0 && (
                            <p className="mt-1 text-xs text-emerald-400">{skill.price_morm} MORM</p>
                          )}
                        </div>
                      </div>
                      {hasWriteAccess && (
                        <button
                          type="button"
                          onClick={() => handleInstall(skill)}
                          disabled={isInstalling}
                          className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg border border-[#3a6de0] bg-[#0b2f80]/40 px-3 py-2 text-xs font-medium text-[#9bc3ff] transition hover:border-[#4f83ff] hover:bg-[#0b2f80]/60 disabled:opacity-50"
                        >
                          {isInstalling ? (
                            <Loader2 className="h-3.5 w-3.5 animate-spin" />
                          ) : (
                            <Download className="h-3.5 w-3.5" />
                          )}
                          {isInstalling ? "Installing…" : "Install"}
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}

      {/* My Skills tab */}
      {tab === "my-skills" && (
        <>
          {mySkillsError && (
            <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
              <span className="text-sm text-rose-300">{mySkillsError}</span>
            </div>
          )}

          {mySkillsLoading ? (
            <div className="flex h-64 items-center justify-center">
              <div className="electric-loader h-12 w-12 rounded-full" />
            </div>
          ) : mySkills.length === 0 ? (
            <div className="electric-card p-8 text-center">
              <Sparkles className="h-10 w-10 text-[#5f84cc] mx-auto mb-3 opacity-70" />
              <p className="text-sm text-[#8aa8df]">
                No published skills yet. Install skills from Discover to build your collection.
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {mySkills.map((skill) => (
                <div key={skill.id} className="electric-card overflow-hidden">
                  <div className="p-4">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <h3 className="text-sm font-semibold text-white truncate">{skill.name}</h3>
                          {skill.badges?.map((b) => (
                            <span
                              key={b}
                              className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-900/50 text-emerald-300 border border-emerald-700/50"
                            >
                              <Shield className="h-2.5 w-2.5" />
                              {b}
                            </span>
                          ))}
                        </div>
                        {skill.description && (
                          <p className="mt-1 text-xs text-[#8aa8df] line-clamp-2">{skill.description}</p>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}
