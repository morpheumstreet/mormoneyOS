import { Outlet, useLocation, Navigate } from "react-router-dom";
import ConfigSubNav from "./ConfigSubNav";
import { useWalletAuth } from "@/contexts/WalletAuthContext";
import { isGuestHiddenPath } from "@/lib/configNav";

export default function ConfigLayout() {
  const location = useLocation();
  const { isGuest } = useWalletAuth();
  const isIdentityDetail = /^\/config\/wallet\/\d+$/.test(location.pathname);

  if (isGuest && isGuestHiddenPath(location.pathname)) {
    return <Navigate to="/config/layout" replace />;
  }

  return (
    <div className="flex flex-col -mx-4 -mt-5 md:-mx-8 md:-mt-8">
      {!isIdentityDetail && <ConfigSubNav />}
      <div className="flex-1 px-4 pt-6 md:px-8">
        <Outlet />
      </div>
    </div>
  );
}
