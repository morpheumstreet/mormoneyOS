import { Routes, Route, Navigate } from "react-router-dom";
import { WalletAuthProvider, useWalletAuth } from "@/contexts/WalletAuthContext";
import { WalletConnectScreen } from "@/components/auth/WalletConnectScreen";
import Layout from "@/components/layout/Layout";
import Dashboard from "@/pages/Dashboard";
import Reports from "@/pages/Reports";
import ConfigLayout from "@/components/config/ConfigLayout";
import ConfigGeneral from "@/pages/config/ConfigGeneral";
import ConfigTools from "@/pages/config/ConfigTools";
import ConfigSocial from "@/pages/config/ConfigSocial";
import ConfigTunnel from "@/pages/config/ConfigTunnel";
import ConfigModelList from "@/pages/config/ConfigModelList";
import ConfigSoul from "@/pages/config/ConfigSoul";

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
        <Route path="/reports" element={<Reports />} />
        <Route path="/config" element={<ConfigLayout />}>
          <Route index element={<Navigate to="/config/general" replace />} />
          <Route path="general" element={<ConfigGeneral />} />
          <Route path="tools" element={<ConfigTools />} />
          <Route path="social" element={<ConfigSocial />} />
          <Route path="tunnel" element={<ConfigTunnel />} />
          <Route path="model-list" element={<ConfigModelList />} />
          <Route path="soul" element={<ConfigSoul />} />
        </Route>
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
