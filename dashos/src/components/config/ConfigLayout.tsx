import { Outlet } from "react-router-dom";
import ConfigSubNav from "./ConfigSubNav";

export default function ConfigLayout() {
  return (
    <div className="flex flex-col -mx-4 -mt-5 md:-mx-8 md:-mt-8">
      <ConfigSubNav />
      <div className="flex-1 px-4 pt-6 md:px-8">
        <Outlet />
      </div>
    </div>
  );
}
