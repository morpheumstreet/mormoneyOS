import { useEffect, useState, useCallback } from "react";
import {
  Wallet2,
  Plus,
  List,
  ArrowLeft,
  ChevronRight,
  CheckCircle,
  Copy,
  AlertTriangle,
} from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { useWalletConfig } from "./useWalletConfig";
import { AddressCard } from "./AddressCard";
import { SUPPORTED_CHAINS, DEFAULT_CHAIN } from "./constants";
import { truncateAddress } from "@/lib/format";
import type { ChainAddress } from "./useWalletConfig";

type Screen =
  | "welcome"
  | "create-index"
  | "create-preview"
  | "create-finalize"
  | "create-success"
  | "manage"
  | "manage-detail";

const FOOTNOTE =
  "Private keys & mnemonic are securely managed — never shown here.";

function CopyToast({ onDone }: { onDone: () => void }) {
  useEffect(() => {
    const t = setTimeout(onDone, 2000);
    return () => clearTimeout(t);
  }, [onDone]);
  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 electric-card px-4 py-2 border-[#00d4aa]/40 bg-[#00d4aa]/10 text-[#00d4aa] text-sm font-medium animate-[riseIn_0.3s_ease]">
      Copied!
    </div>
  );
}

export default function ConfigWallet() {
  const { hasWriteAccess } = useWalletAuth();
  const {
    wallet,
    loading,
    error,
    success,
    setError,
    setSuccess,
    loadWallet,
    deriveAddressesForIndex,
    rotateToIndex,
    updateConfigIdentity,
    getIdentityLabel,
  } = useWalletConfig();

  const [screen, setScreen] = useState<Screen>("welcome");
  const [createIndex, setCreateIndex] = useState(1);
  const [previewAddresses, setPreviewAddresses] = useState<ChainAddress[]>([]);
  const [selectedChain, setSelectedChain] = useState(DEFAULT_CHAIN);
  const [identityLabel, setIdentityLabel] = useState("");
  const [understoodCheck, setUnderstoodCheck] = useState(false);
  const [creating, setCreating] = useState(false);
  const [labelEdit, setLabelEdit] = useState("");
  const [savingLabel, setSavingLabel] = useState(false);
  const [copyToast, setCopyToast] = useState(false);
  const [qrModal, setQrModal] = useState<{ addr: string; label: string } | null>(null);

  useEffect(() => {
    loadWallet();
  }, [loadWallet]);

  const copyToClipboard = useCallback((addr: string) => {
    navigator.clipboard.writeText(addr);
    setCopyToast(true);
  }, []);

  const chainName = (caip2: string) =>
    SUPPORTED_CHAINS.find((c) => c.caip2 === caip2)?.name ?? caip2;

  // Create flow: reset when entering create
  useEffect(() => {
    if (screen === "create-index" && wallet?.exists && wallet.currentIndex != null) {
      setCreateIndex(wallet.currentIndex + 1);
    }
  }, [screen, wallet?.exists, wallet?.currentIndex]);

  const handlePreviewAddresses = useCallback(async () => {
    setError(null);
    try {
      const addrs = await deriveAddressesForIndex(createIndex);
      setPreviewAddresses(addrs);
      setScreen("create-preview");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to derive addresses");
    }
  }, [createIndex, deriveAddressesForIndex, setError]);

  const handleCreateAndSave = useCallback(async () => {
    if (!hasWriteAccess) {
      setError("Write access required. Connect wallet and sign.");
      return;
    }
    if (!identityLabel.trim()) {
      setError("Please enter a name for this identity.");
      return;
    }
    if (!understoodCheck) {
      setError("Please confirm you understand that private keys are never shown.");
      return;
    }
    setCreating(true);
    setError(null);
    try {
      await rotateToIndex(createIndex, true);
      await updateConfigIdentity({
        defaultChain: selectedChain,
        identityLabel: identityLabel.trim(),
        index: createIndex,
      });
      setScreen("create-success");
    } catch {
      // error already set in rotateToIndex
    } finally {
      setCreating(false);
    }
  }, [
    hasWriteAccess,
    identityLabel,
    understoodCheck,
    createIndex,
    selectedChain,
    rotateToIndex,
    updateConfigIdentity,
    setError,
  ]);

  const handleManageDetail = useCallback(async () => {
    if (!wallet?.exists || wallet.currentIndex == null) return;
    const lbl = await getIdentityLabel(wallet.currentIndex);
    setLabelEdit(lbl || `Account #${wallet.currentIndex}`);
    setScreen("manage-detail");
  }, [wallet?.exists, wallet?.currentIndex, getIdentityLabel]);

  const handleSaveLabel = useCallback(async () => {
    if (!wallet?.exists || wallet.currentIndex == null) return;
    setSavingLabel(true);
    setError(null);
    try {
      await updateConfigIdentity({
        identityLabel: labelEdit.trim(),
        index: wallet.currentIndex,
      });
      setSuccess("Label saved.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSavingLabel(false);
    }
  }, [wallet?.exists, wallet?.currentIndex, labelEdit, updateConfigIdentity, setSuccess, setError]);

  const primaryAddress = previewAddresses.find((a) => a.caip2 === selectedChain)?.address ?? "";
  const morpheumAddr = previewAddresses.find((a) => a.isMorpheum)?.address ?? "";

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="electric-loader h-12 w-12 rounded-full" />
      </div>
    );
  }

  if (!wallet?.exists) {
    return (
      <ConfigPageLayout
        icon={Wallet2}
        title="Wallet Identity"
        description="Manage agent wallet identities. One seed → many chains."
        hasWriteAccess={!!hasWriteAccess}
        error={wallet?.error ?? "No wallet found. Run 'moneyclaw init' first."}
        loading={false}
      >
        <div className="electric-card p-6 text-center text-[#8aa8df]">
          <p className="text-sm">
            Create a wallet with <code className="rounded bg-[#1a3670] px-1.5 py-0.5">moneyclaw init</code> before managing identities here.
          </p>
        </div>
      </ConfigPageLayout>
    );
  }

  // QR modal (simple overlay)
  if (qrModal) {
    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
        onClick={() => setQrModal(null)}
      >
        <div
          className="electric-card p-6 max-w-sm"
          onClick={(e) => e.stopPropagation()}
        >
          <p className="text-sm font-medium text-[#9bb7eb] mb-2">{qrModal.label}</p>
          <div className="bg-white p-4 rounded-lg inline-block">
            {/* Placeholder for QR - in production use qrcode.react or similar */}
            <div className="w-32 h-32 bg-slate-200 flex items-center justify-center text-slate-500 text-xs">
              QR
            </div>
          </div>
          <p className="mt-2 font-mono text-xs text-[#8aa8df] break-all">{qrModal.addr}</p>
          <button
            type="button"
            onClick={() => setQrModal(null)}
            className="mt-4 w-full electric-button py-2 rounded-lg text-sm"
          >
            Close
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {copyToast && <CopyToast onDone={() => setCopyToast(false)} />}

      {/* Screen: Welcome */}
      {screen === "welcome" && (
        <div className="motion-rise">
          <div className="text-center max-w-2xl mx-auto py-8">
            <h2 className="text-2xl font-bold text-white mb-2">
              Give your Automaton its next identity
            </h2>
            <p className="text-[#8aa8df] text-sm mb-8">
              One secure seed → many chains. Powered by Morpheum standards.
            </p>
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              <button
                type="button"
                onClick={() => setScreen("create-index")}
                className="electric-button flex items-center justify-center gap-2 px-6 py-4 rounded-xl text-base font-medium"
              >
                <Plus className="h-5 w-5" />
                Create New Identity
              </button>
              <button
                type="button"
                onClick={() => setScreen("manage")}
                className="flex items-center justify-center gap-2 px-6 py-4 rounded-xl text-base font-medium border border-[#3a6de0] text-[#9bb7eb] hover:bg-[#0b2f80]/40 hover:border-[#4f83ff] transition-all"
              >
                <List className="h-5 w-5" />
                Manage Existing Identities
              </button>
            </div>
            <p className="mt-8 text-xs text-[#6b8fcc]">{FOOTNOTE}</p>
          </div>
        </div>
      )}

      {/* Screen: Create - Index selection */}
      {screen === "create-index" && (
        <div className="motion-rise space-y-6">
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={() => setScreen("welcome")}
              className="flex items-center gap-1 text-[#8aa8df] hover:text-white"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>
            <span className="text-sm text-[#6b8fcc]">Create New Identity · Step 1/3</span>
          </div>
          <div className="electric-card p-6">
            <p className="text-sm text-[#9bb7eb] mb-2">
              Next available account index: <strong className="text-white">{createIndex}</strong>
            </p>
            <p className="text-xs text-[#6b8fcc] mb-4">
              {wallet.currentIndex != null && `(last used was ${wallet.currentIndex})`}
            </p>
            <div className="flex items-center gap-4">
              <button
                type="button"
                onClick={() => setCreateIndex(Math.max(1, (wallet.currentIndex ?? 0) + 1, createIndex - 1))}
                className="electric-button px-4 py-2 rounded-lg"
              >
                −
              </button>
              <span className="text-2xl font-mono font-bold text-white min-w-[3rem] text-center">
                {createIndex}
              </span>
              <button
                type="button"
                onClick={() => setCreateIndex(createIndex + 1)}
                className="electric-button px-4 py-2 rounded-lg"
              >
                +
              </button>
              <button
                type="button"
                onClick={() => setCreateIndex(10000 + Math.floor(Math.random() * 90000))}
                className="electric-button px-4 py-2 rounded-lg text-sm"
              >
                Pick random 5-digit
              </button>
            </div>
            {createIndex > 50 && (
              <p className="mt-2 text-amber-400/90 text-xs">
                Higher indices are fine — but remember to fund them.
              </p>
            )}
            <p className="mt-4 text-xs text-[#6b8fcc]">
              Derivation follows standard paths per chain. The agent will use this new identity going forward.
            </p>
            <button
              type="button"
              onClick={handlePreviewAddresses}
              className="mt-6 electric-button flex items-center gap-2 px-5 py-2.5 rounded-lg"
            >
              Preview Addresses
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {/* Screen: Create - Address preview */}
      {screen === "create-preview" && (
        <div className="motion-rise space-y-6">
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={() => setScreen("create-index")}
              className="flex items-center gap-1 text-[#8aa8df] hover:text-white"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>
            <span className="text-sm text-[#6b8fcc]">Your new multi-chain identity (index {createIndex})</span>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {previewAddresses.map((item) => (
              <AddressCard
                key={item.caip2}
                item={item}
                onCopy={copyToClipboard}
                onShowQr={(addr, label) => setQrModal({ addr, label })}
              />
            ))}
          </div>
          <div className="flex justify-end">
            <button
              type="button"
              onClick={() => setScreen("create-finalize")}
              className="electric-button flex items-center gap-2 px-5 py-2.5 rounded-lg"
            >
              Looks good!
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {/* Screen: Create - Default chain & label */}
      {screen === "create-finalize" && (
        <div className="motion-rise space-y-6">
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={() => setScreen("create-preview")}
              className="flex items-center gap-1 text-[#8aa8df] hover:text-white"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>
            <span className="text-sm text-[#6b8fcc]">Almost there · Step 2/3</span>
          </div>

          {error && (
            <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
              <span className="text-sm text-rose-300">{error}</span>
            </div>
          )}

          <div className="electric-card p-6 space-y-6">
            <div>
              <h3 className="text-sm font-medium text-[#9bb7eb] mb-2">Primary chain</h3>
              <p className="text-xs text-[#6b8fcc] mb-3">
                Used for provisioning, heartbeats, and when no chain is specified.
              </p>
              <div className="flex flex-wrap gap-2">
                {SUPPORTED_CHAINS.map((ch) => (
                  <button
                    key={ch.caip2}
                    type="button"
                    onClick={() => setSelectedChain(ch.caip2)}
                    className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-all ${
                      selectedChain === ch.caip2
                        ? ch.isMorpheum
                          ? "bg-[#00d4aa]/30 text-[#00d4aa] border border-[#00d4aa]/50"
                          : "bg-[#4f83ff]/30 text-[#9bc3ff] border border-[#4f83ff]/50"
                        : "border border-[#29509c] text-[#8aa8df] hover:bg-[#07132f]"
                    }`}
                  >
                    {ch.name}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <h3 className="text-sm font-medium text-[#9bb7eb] mb-2">Give it a name</h3>
              <input
                type="text"
                value={identityLabel}
                onChange={(e) => setIdentityLabel(e.target.value)}
                placeholder="My Trading Agent #3"
                className="w-full px-4 py-3 rounded-lg border border-[#29509c] bg-[#071228]/90 text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none"
                maxLength={64}
              />
              <p className="mt-1 text-xs text-[#6b8fcc]">
                {identityLabel.length}/64
              </p>
            </div>

            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={understoodCheck}
                onChange={(e) => setUnderstoodCheck(e.target.checked)}
                className="mt-1 rounded border-[#29509c]"
              />
              <span className="text-sm text-[#8aa8df]">
                I understand private keys are never shown and are managed securely by the backend.
              </span>
            </label>

            <button
              type="button"
              onClick={handleCreateAndSave}
              disabled={creating || !identityLabel.trim() || !understoodCheck || !hasWriteAccess}
              className="electric-button w-full py-3 rounded-lg font-medium disabled:opacity-50"
            >
              {creating ? "Creating…" : "Create & Save Identity"}
            </button>
          </div>
        </div>
      )}

      {/* Screen: Create - Success */}
      {screen === "create-success" && (
        <div className="motion-rise text-center py-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-emerald-500/20 border border-emerald-500/40 mb-4">
            <CheckCircle className="h-8 w-8 text-emerald-400" />
          </div>
          <h2 className="text-2xl font-bold text-white mb-2">Identity Created!</h2>
          <p className="text-[#8aa8df] mb-6">
            Account #{createIndex} – &quot;{identityLabel}&quot;
          </p>
          <div className="electric-card p-6 max-w-md mx-auto text-left space-y-2">
            <p><span className="text-[#6b8fcc]">Name:</span> {identityLabel}</p>
            <p><span className="text-[#6b8fcc]">Index:</span> {createIndex}</p>
            <p><span className="text-[#6b8fcc]">Default chain:</span> {chainName(selectedChain)}</p>
            <p>
              <span className="text-[#6b8fcc]">Primary address:</span>{" "}
              {truncateAddress(primaryAddress, 6, 4)}
              <button
                type="button"
                onClick={() => copyToClipboard(primaryAddress)}
                className="ml-2 inline p-1 rounded text-[#8aa8df] hover:bg-[#1a3670]"
              >
                <Copy className="h-3.5 w-3.5" />
              </button>
            </p>
            {morpheumAddr && (
              <p>
                <span className="text-[#6b8fcc]">Morpheum address:</span>{" "}
                <span className="text-[#00d4aa]">{truncateAddress(morpheumAddr, 8, 6)}</span>
                <button
                  type="button"
                  onClick={() => copyToClipboard(morpheumAddr)}
                  className="ml-2 inline p-1 rounded text-[#8aa8df] hover:bg-[#1a3670]"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </p>
            )}
          </div>
          <div className="flex flex-col sm:flex-row gap-4 justify-center mt-8">
            <button
              type="button"
              onClick={() => { setScreen("welcome"); loadWallet(); }}
              className="electric-button px-6 py-2.5 rounded-lg"
            >
              Continue to Dashboard
            </button>
            <button
              type="button"
              onClick={() => { setScreen("create-index"); setIdentityLabel(""); setUnderstoodCheck(false); loadWallet(); }}
              className="border border-[#3a6de0] text-[#9bb7eb] hover:bg-[#0b2f80]/40 px-6 py-2.5 rounded-lg"
            >
              Create Another Identity
            </button>
          </div>
        </div>
      )}

      {/* Screen: Manage list */}
      {screen === "manage" && (
        <div className="motion-rise space-y-6">
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={() => setScreen("welcome")}
              className="flex items-center gap-1 text-[#8aa8df] hover:text-white"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>
          </div>
          <div className="electric-card overflow-hidden">
            <div
              className="flex items-center gap-4 p-4 cursor-pointer hover:bg-[#0b2f80]/30 transition-colors"
              onClick={handleManageDetail}
            >
              <div
                className="w-12 h-12 rounded-full shrink-0"
                style={{
                  background: `linear-gradient(135deg, #3a6de0 0%, #00d4aa 100%)`,
                }}
              />
              <div className="min-w-0 flex-1">
                <p className="font-medium text-white">
                  {wallet.currentIndex != null ? `Account #${wallet.currentIndex}` : "Current identity"}
                </p>
                <p className="text-sm text-[#6b8fcc]">
                  {wallet.defaultChain ? chainName(wallet.defaultChain) : "—"} · {wallet.address ? truncateAddress(wallet.address, 6, 4) : "—"}
                </p>
              </div>
              <ChevronRight className="h-5 w-5 text-[#6b8fcc] shrink-0" />
            </div>
          </div>
          <p className="text-xs text-[#6b8fcc]">{FOOTNOTE}</p>
        </div>
      )}

      {/* Screen: Manage detail */}
      {screen === "manage-detail" && wallet?.exists && (
        <div className="motion-rise space-y-6">
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={() => setScreen("manage")}
              className="flex items-center gap-1 text-[#8aa8df] hover:text-white"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>
          </div>

          {success && (
            <div className="electric-card p-3 border-emerald-500/30 bg-emerald-950/20 flex items-center gap-2">
              <CheckCircle className="h-4 w-4 text-emerald-400 flex-shrink-0" />
              <span className="text-sm text-emerald-300">{success}</span>
            </div>
          )}
          {error && (
            <div className="electric-card p-3 border-rose-500/30 bg-rose-950/20 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-rose-400 flex-shrink-0" />
              <span className="text-sm text-rose-300">{error}</span>
            </div>
          )}

          <div className="electric-card p-6 space-y-4">
            <div>
              <label className="block text-sm font-medium text-[#9bb7eb] mb-2">Friendly name</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={labelEdit}
                  onChange={(e) => setLabelEdit(e.target.value)}
                  placeholder={`Account #${wallet.currentIndex}`}
                  className="flex-1 px-4 py-2 rounded-lg border border-[#29509c] bg-[#071228]/90 text-white placeholder:text-[#6b8fcc] focus:border-[#4f83ff] focus:outline-none"
                />
                <button
                  type="button"
                  onClick={handleSaveLabel}
                  disabled={savingLabel || !hasWriteAccess}
                  className="electric-button px-4 py-2 rounded-lg disabled:opacity-50"
                >
                  {savingLabel ? "Saving…" : "Save"}
                </button>
              </div>
            </div>
          </div>

          <div>
            <h3 className="text-sm font-medium text-[#9bb7eb] mb-3">Addresses by chain</h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              <ManageDetailAddresses
                index={wallet.currentIndex ?? 0}
                deriveAddressesForIndex={deriveAddressesForIndex}
                onCopy={copyToClipboard}
                onShowQr={(addr, label) => setQrModal({ addr, label })}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function ManageDetailAddresses({
  index,
  deriveAddressesForIndex,
  onCopy,
  onShowQr,
}: {
  index: number;
  deriveAddressesForIndex: (i: number) => Promise<ChainAddress[]>;
  onCopy: (a: string) => void;
  onShowQr: (addr: string, label: string) => void;
}) {
  const [addrs, setAddrs] = useState<ChainAddress[]>([]);
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    deriveAddressesForIndex(index).then(setAddrs).finally(() => setLoading(false));
  }, [index, deriveAddressesForIndex]);
  if (loading) return <div className="col-span-full py-4 text-center text-[#6b8fcc]">Loading…</div>;
  return (
    <>
      {addrs.map((item) => (
        <AddressCard
          key={item.caip2}
          item={item}
          onCopy={onCopy}
          onShowQr={onShowQr}
        />
      ))}
    </>
  );
}
