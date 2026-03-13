import { Routes, Route, Navigate } from "react-router-dom";
import { WalletAuthProvider, useWalletAuth } from "@/contexts/WalletAuthContext";
import { WalletConnectScreen } from "@/components/auth/WalletConnectScreen";
import Layout from "@/components/layout/Layout";
import Dashboard from "@/pages/Dashboard";
import Config from "@/pages/Config";

function AppContent() {
  const { address, hasWriteAccess } = useWalletAuth();
  const isAuthenticated = !!address && hasWriteAccess;

  if (!isAuthenticated) {
    return <WalletConnectScreen />;
  }

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/config" element={<Config />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}

export default function App() {
  return (
    <WalletAuthProvider>
      <AppContent />
    </WalletAuthProvider>
  );
}
