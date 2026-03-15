import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Wallet2 } from "lucide-react";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { ConfigPageLayout } from "@/components/config/ConfigPageLayout";
import { ConfigWalletProvider, useConfigWallet } from "./ConfigWalletContext";
import { DEFAULT_CHAIN } from "./constants";
import type { ChainAddress } from "./useWalletConfig";
import { QRModal } from "@/components/ui/QRModal";
import { CopyToast } from "@/components/ui/CopyToast";
import {
  WelcomeScreen,
  CreateIndexScreen,
  CreatePreviewScreen,
  CreateFinalizeScreen,
  CreateSuccessScreen,
  ManageScreen,
  ManageDetailScreen,
} from "./screens";

type Screen =
  | "welcome"
  | "create-index"
  | "create-preview"
  | "create-finalize"
  | "create-success"
  | "manage"
  | "manage-detail";

function ConfigWalletContent() {
  const { hasWriteAccess } = useWalletAuth();
  const { index: indexParam } = useParams<{ index: string }>();
  const navigate = useNavigate();
  const config = useConfigWallet();

  const {
    wallet,
    identityLabels,
    loading,
    error,
    success,
    setError,
    setSuccess,
    loadWallet,
    loadIdentityLabels,
    getPrimaryAddress,
    deriveAddressesForIndex,
    rotateToIndex,
    updateConfigIdentity,
    getIdentityLabel,
  } = config;

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
  const [detailIndex, setDetailIndex] = useState<number | null>(null);

  useEffect(() => {
    loadWallet();
  }, [loadWallet]);

  useEffect(() => {
    if (screen === "manage") loadIdentityLabels();
  }, [screen, loadIdentityLabels]);

  useEffect(() => {
    if (indexParam != null && wallet?.exists) {
      const idx = parseInt(indexParam, 10);
      if (!Number.isNaN(idx) && idx >= 0) {
        loadIdentityLabels();
        setDetailIndex(idx);
        setScreen("manage-detail");
        setLabelEdit(identityLabels[String(idx)] ?? `Account #${idx}`);
        getIdentityLabel(idx).then((fetched) => {
          setLabelEdit((prev) => (prev === `Account #${idx}` && fetched ? fetched : prev));
        });
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- identityLabels causes infinite loop
  }, [indexParam, wallet?.exists]);

  useEffect(() => {
    if (screen === "create-index" && wallet?.exists && wallet.currentIndex != null) {
      setCreateIndex(wallet.currentIndex + 1);
    }
  }, [screen, wallet?.exists, wallet?.currentIndex]);

  const copyToClipboard = useCallback((addr: string) => {
    navigator.clipboard.writeText(addr);
    setCopyToast(true);
  }, []);

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

  const handleSaveLabel = useCallback(
    async (newLabel: string) => {
      if (!wallet?.exists || detailIndex == null) return;
      setSavingLabel(true);
      setError(null);
      try {
        await updateConfigIdentity({ identityLabel: newLabel, index: detailIndex });
        setLabelEdit(newLabel);
        setSuccess("Label saved.");
        await loadIdentityLabels();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to save");
      } finally {
        setSavingLabel(false);
      }
    },
    [wallet?.exists, detailIndex, updateConfigIdentity, loadIdentityLabels, setSuccess, setError]
  );

  const handleSwitchToIdentity = useCallback(async () => {
    if (!hasWriteAccess || detailIndex == null) return;
    setError(null);
    try {
      await rotateToIndex(detailIndex, true);
      setSuccess(`Switched to identity #${detailIndex}.`);
      await loadWallet();
    } catch {
      // error set by rotateToIndex
    }
  }, [hasWriteAccess, detailIndex, rotateToIndex, loadWallet]);

  const goBackToManage = useCallback(() => {
    navigate("/config/wallet");
    setScreen("manage");
    setDetailIndex(null);
    setSuccess(null);
    setError(null);
  }, [navigate]);

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
        title="ID Identity"
        description="Manage agent ID identities. One seed → many chains."
        hasWriteAccess={!!hasWriteAccess}
        error={wallet?.error ?? "No ID found. Run 'moneyclaw init' first."}
        loading={false}
      >
        <div className="electric-card p-6 text-center text-[#8aa8df]">
          <p className="text-sm">
            Create an ID with <code className="rounded bg-[#1a3670] px-1.5 py-0.5">moneyclaw init</code> before managing identities here.
          </p>
        </div>
      </ConfigPageLayout>
    );
  }

  if (qrModal) {
    return (
      <QRModal
        address={qrModal.addr}
        label={qrModal.label}
        onClose={() => setQrModal(null)}
      />
    );
  }

  return (
    <div className="space-y-6">
      {copyToast && <CopyToast onDone={() => setCopyToast(false)} />}

      {screen === "welcome" && (
        <WelcomeScreen
          onCreate={() => setScreen("create-index")}
          onManage={() => setScreen("manage")}
        />
      )}

      {screen === "create-index" && (
        <CreateIndexScreen
          createIndex={createIndex}
          setCreateIndex={setCreateIndex}
          currentIndex={wallet.currentIndex ?? null}
          onPreview={handlePreviewAddresses}
          onBack={() => setScreen("welcome")}
        />
      )}

      {screen === "create-preview" && (
        <CreatePreviewScreen
          createIndex={createIndex}
          previewAddresses={previewAddresses}
          onCopy={copyToClipboard}
          onShowQr={(addr, label) => setQrModal({ addr, label })}
          onContinue={() => setScreen("create-finalize")}
          onBack={() => setScreen("create-index")}
        />
      )}

      {screen === "create-finalize" && (
        <CreateFinalizeScreen
          error={error}
          identityLabel={identityLabel}
          setIdentityLabel={setIdentityLabel}
          understoodCheck={understoodCheck}
          setUnderstoodCheck={setUnderstoodCheck}
          selectedChain={selectedChain}
          setSelectedChain={setSelectedChain}
          onCreate={handleCreateAndSave}
          creating={creating}
          hasWriteAccess={!!hasWriteAccess}
          onBack={() => setScreen("create-preview")}
        />
      )}

      {screen === "create-success" && (
        <CreateSuccessScreen
          createIndex={createIndex}
          identityLabel={identityLabel}
          selectedChain={selectedChain}
          primaryAddress={primaryAddress}
          morpheumAddr={morpheumAddr}
          onCopy={copyToClipboard}
          onContinue={() => {
            setScreen("welcome");
            loadWallet();
          }}
          onCreateAnother={() => {
            setScreen("create-index");
            setIdentityLabel("");
            setUnderstoodCheck(false);
            loadWallet();
          }}
        />
      )}

      {screen === "manage" && (
        <ManageScreen
          identityLabels={identityLabels}
          currentIndex={wallet.currentIndex}
          defaultChain={wallet.defaultChain}
          getPrimaryAddress={getPrimaryAddress}
          onDetail={(idx) => navigate(`/config/wallet/${idx}`)}
          onBack={() => setScreen("welcome")}
        />
      )}

      {screen === "manage-detail" && wallet?.exists && detailIndex != null && (
        <ManageDetailScreen
          detailIndex={detailIndex}
          currentIndex={wallet.currentIndex}
          labelEdit={labelEdit}
          success={success}
          error={error}
          hasWriteAccess={!!hasWriteAccess}
          savingLabel={savingLabel}
          onSaveLabel={handleSaveLabel}
          onSwitchIdentity={handleSwitchToIdentity}
          deriveAddressesForIndex={deriveAddressesForIndex}
          onCopy={copyToClipboard}
          onShowQr={(addr, label) => setQrModal({ addr, label })}
          onBack={goBackToManage}
        />
      )}
    </div>
  );
}

export default function ConfigWallet() {
  return (
    <ConfigWalletProvider>
      <ConfigWalletContent />
    </ConfigWalletProvider>
  );
}
