import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../stores/auth';
import { useState, useEffect } from 'react';
import { api } from '../services/api';

export default function AppLayout() {
  const { user, token, organizations, activeOrgId, setActiveOrgId, logout } = useAuthStore();
  const location = useLocation();
  const navigate = useNavigate();

  const [credits, setCredits] = useState<number | null>(null);

  useEffect(() => {
    if (!token || !activeOrgId) return;

    // Fetch credits for the active organization
    api.get(`/credits/balance`)
      .then((res) => {
        setCredits(res.data.data?.balance ?? 0);
      })
      .catch(() => {
        setCredits(0);
      });
  }, [activeOrgId, token]);

  const activeOrg = organizations.find((o) => o.organization_id === activeOrgId);

  const menuItems = [
    { label: 'Painel', path: '/app', icon: '📊' },
    { label: 'Meus Mapas', path: '/app/maps', icon: '🧠' },
    { label: 'Faturamento', path: '/app/billing', icon: '💳' },
    { label: 'Configurações', path: '/app/settings', icon: '⚙️' },
  ];

  return (
    <div className="min-h-screen bg-slate-950 text-white flex">
      {/* Sidebar */}
      <aside className="w-64 bg-slate-900 border-r border-slate-800 flex flex-col justify-between hidden md:flex">
        <div>
          {/* Brand Logo */}
          <div className="p-6 border-b border-slate-800 flex items-center gap-3">
            <div className="h-8 w-8 rounded-lg bg-gradient-to-tr from-purple-600 to-indigo-600 flex items-center justify-center font-bold text-white shadow-lg">
              M
            </div>
            <span className="font-extrabold text-lg bg-gradient-to-r from-white to-purple-400 bg-clip-text text-transparent">
              MapaTurbo <span className="text-purple-500 text-sm">App</span>
            </span>
          </div>

          {/* User Org / Tenant Selector */}
          <div className="p-4 border-b border-slate-800">
            <label className="block text-[10px] font-bold uppercase tracking-wider text-slate-500 mb-1.5">
              Workspace Ativo
            </label>
            <select
              value={activeOrgId || ''}
              onChange={(e) => setActiveOrgId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-purple-600"
            >
              {organizations.map((o) => (
                <option key={o.organization_id} value={o.organization_id}>
                  {o.organization_name}
                </option>
              ))}
            </select>
          </div>

          {/* Navigation Links */}
          <nav className="p-4 space-y-1">
            {menuItems.map((item, idx) => {
              const isActive = location.pathname === item.path;
              return (
                <Link
                  key={idx}
                  to={item.path}
                  className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${
                    isActive
                      ? 'bg-purple-600 text-white'
                      : 'text-slate-400 hover:bg-slate-850 hover:text-white'
                  }`}
                >
                  <span>{item.icon}</span>
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </div>

        {/* User Info & Switch Panel */}
        <div className="p-4 border-t border-slate-800 space-y-4">
          {/* Credit balance display */}
          <div className="bg-slate-950 border border-slate-850 p-3 rounded-xl flex items-center justify-between text-xs">
            <span className="text-slate-400 font-medium">Créditos de IA:</span>
            <span className="font-bold text-purple-400">{credits !== null ? `${credits} 🪙` : 'Carregando...'}</span>
          </div>

          {/* Super Admin Switch */}
          {user?.global_role === 'SUPER_ADMIN' && (
            <Link
              to="/admin"
              className="w-full text-center bg-slate-950 border border-slate-800 hover:border-slate-700 py-2 rounded-lg text-xs font-semibold flex items-center justify-center gap-2 text-slate-300 hover:text-white transition-all"
            >
              🛡️ Ir para Admin
            </Link>
          )}

          <div className="flex items-center justify-between gap-2">
            <div className="min-w-0">
              <p className="text-xs font-bold truncate">{user?.name}</p>
              <p className="text-[10px] text-slate-500 truncate">{user?.email}</p>
            </div>
            <button
              onClick={() => {
                logout();
                navigate('/login');
              }}
              className="text-xs hover:text-red-400 text-slate-500 font-bold px-2 py-1 rounded hover:bg-slate-850 transition-colors"
            >
              Sair
            </button>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-h-screen overflow-x-hidden">
        {/* Mobile Header */}
        <header className="h-14 bg-slate-900 border-b border-slate-800 flex items-center justify-between px-6 md:hidden">
          <div className="flex items-center gap-3">
            <div className="h-8 w-8 rounded-lg bg-gradient-to-tr from-purple-600 to-indigo-600 flex items-center justify-center font-bold text-white shadow-lg">
              M
            </div>
            <span className="font-extrabold text-sm tracking-tight">MapaTurbo</span>
          </div>

          <div className="flex items-center gap-4">
            <select
              value={activeOrgId || ''}
              onChange={(e) => setActiveOrgId(e.target.value)}
              className="bg-slate-950 border border-slate-800 rounded px-2 py-1 text-xs"
            >
              {organizations.map((o) => (
                <option key={o.organization_id} value={o.organization_id}>
                  {o.organization_name}
                </option>
              ))}
            </select>
            <button
              onClick={() => {
                logout();
                navigate('/login');
              }}
              className="text-xs text-red-400 font-semibold"
            >
              Sair
            </button>
          </div>
        </header>

        {/* Content Outlet */}
        <main className="flex-1 p-6 md:p-10 max-w-7xl w-full mx-auto">
          <div className="mb-6 flex justify-between items-center hidden md:flex">
            <div>
              <h1 className="text-2xl font-extrabold tracking-tight">
                {menuItems.find((item) => item.path === location.pathname)?.label || 'Painel'}
              </h1>
              <p className="text-xs text-slate-400 mt-1">
                Workspace: <span className="font-semibold text-slate-300">{activeOrg?.organization_name}</span>
              </p>
            </div>
          </div>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
