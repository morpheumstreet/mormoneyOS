import { Routes, Route, Navigate } from "react-router-dom";
import { WalletAuthProvider, useWalletAuth } from "@/contexts/WalletAuthContext";
import { WalletConnectScreen } from "@/components/auth/WalletConnectScreen";
import Layout from "@/components/layout/Layout";
import Dashboard from "@/pages/Dashboard";
import Reports from "@/pages/Reports";
import ConfigLayout from "@/components/config/ConfigLayout";
import ConfigAuth from "@/pages/config/ConfigAuth";
import ConfigGeneral from "@/pages/config/ConfigGeneral";
import ConfigTools from "@/pages/config/ConfigTools";
import ConfigSocial from "@/pages/config/ConfigSocial";
import ConfigHeartbeat from "@/pages/config/ConfigHeartbeat";
import ConfigTunnel from "@/pages/config/ConfigTunnel";
import ConfigMiroFish from "@/pages/config/ConfigMiroFish";
import ConfigModelList from "@/pages/config/ConfigModelList";
import ConfigEconomic from "@/pages/config/ConfigEconomic";
import ConfigWallet from "@/pages/config/ConfigWallet";
import ConfigSoul from "@/pages/config/ConfigSoul";
import ConfigLayoutSettings from "@/pages/config/ConfigLayoutSettings";
import Skills from "@/pages/Skills";

function AppContent() {
  const { isAuthenticated } = useWalletAuth();

  if (!isAuthenticated) {
    return <WalletConnectScreen />;
  }

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/reports" element={<Reports />} />
        <Route path="/skills" element={<Skills />} />
        <Route path="/config" element={<ConfigLayout />}>
          <Route index element={<Navigate to="/config/general" replace />} />
          <Route path="layout" element={<ConfigLayoutSettings />} />
          <Route path="general" element={<ConfigGeneral />} />
          <Route path="auth" element={<ConfigAuth />} />
          <Route path="tools" element={<ConfigTools />} />
          <Route path="social" element={<ConfigSocial />} />
          <Route path="heartbeat" element={<ConfigHeartbeat />} />
          <Route path="tunnel" element={<ConfigTunnel />} />
          <Route path="mirofish" element={<ConfigMiroFish />} />
          <Route path="model-list" element={<ConfigModelList />} />
          <Route path="economic" element={<ConfigEconomic />} />
          <Route path="wallet/:index" element={<ConfigWallet />} />
          <Route path="wallet" element={<ConfigWallet />} />
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
