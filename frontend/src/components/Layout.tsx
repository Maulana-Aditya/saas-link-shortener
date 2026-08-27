import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth";

const navItems = [
  { to: "/", label: "Links" },
  { to: "/analytics", label: "Analytics" },
  { to: "/billing", label: "Billing" },
];

export default function Layout() {
  const { user, org, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b flex items-center justify-between px-6 py-3">
        <div className="flex items-center gap-6">
          <span className="font-semibold">Shrtn</span>
          <nav className="flex gap-4 text-sm">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end
                className={({ isActive }) =>
                  isActive ? "font-medium text-blue-600" : "text-gray-600 hover:text-gray-900"
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
        </div>
        <div className="flex items-center gap-4 text-sm">
          <span className="text-gray-500">
            {org?.name} · {org?.plan} plan
          </span>
          <span className="text-gray-500">{user?.email}</span>
          <button onClick={handleLogout} className="text-red-600 hover:underline">
            Log out
          </button>
        </div>
      </header>
      <main className="flex-1 p-6">
        <Outlet />
      </main>
    </div>
  );
}
