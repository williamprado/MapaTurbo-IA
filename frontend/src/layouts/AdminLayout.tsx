import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../stores/auth';

export default function AdminLayout() {
  const { user, logout } = useAuthStore();
  const location = useLocation();
  const navigate = useNavigate();

  const menuItems = [
    { label: 'Estatísticas', path: '/admin', icon: '📈' },
    { label: 'Empresas', path: '/admin/organizations', icon: '🏢' },
    { label: 'Planos', path: '/admin/plans', icon: '📋' },
    { label: 'Usuários', path: '/admin/users', icon: '👥' },
    { label: 'Configurações', path: '/admin/settings', icon: '⚙️' },
    { label: 'Faturamento', path: '/admin/payments', icon: '💰' },
    { label: 'Provedores IA', path: '/admin/ai-providers', icon: '🤖' },
    { label: 'Assinaturas', path: '/admin/subscriptions', icon: '💳' },
    { label: 'Auditoria', path: '/admin/audit-logs', icon: '📜' },
  ];

  return (
    <div className="min-h-screen bg-slate-950 text-white flex">
      {/* Sidebar */}
      <aside className="w-64 bg-slate-900 border-r border-red-950/20 flex flex-col justify-between hidden md:flex">
        <div>
          {/* Brand Logo */}
          <div className="p-6 border-b border-slate-800 flex items-center gap-3">
            <div className="h-8 w-8 rounded-lg bg-gradient-to-tr from-red-600 to-purple-600 flex items-center justify-center font-bold text-white shadow-lg">
              A
            </div>
            <span className="font-extrabold text-lg bg-gradient-to-r from-white to-red-400 bg-clip-text text-transparent">
              MapaTurbo <span className="text-red-500 text-xs">Admin</span>
            </span>
          </div>

          <div className="p-3 bg-red-950/20 border-b border-slate-800 text-[10px] text-center text-red-400 font-bold uppercase tracking-widest">
            Painel Global
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
                      ? 'bg-red-700 text-white'
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
          {/* App Switch */}
          <Link
            to="/app"
            className="w-full text-center bg-slate-950 border border-slate-800 hover:border-slate-700 py-2 rounded-lg text-xs font-semibold flex items-center justify-center gap-2 text-slate-300 hover:text-white transition-all"
          >
            🧠 Voltar para o App
          </Link>

          <div className="flex items-center justify-between gap-2">
            <div className="min-w-0">
              <p className="text-xs font-bold truncate text-red-400">{user?.name}</p>
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
            <div className="h-8 w-8 rounded-lg bg-gradient-to-tr from-red-650 to-purple-600 flex items-center justify-center font-bold text-white shadow-lg">
              A
            </div>
            <span className="font-extrabold text-sm tracking-tight text-red-500">Super Admin</span>
          </div>

          <div className="flex items-center gap-4">
            <Link to="/app" className="text-xs text-slate-400">App</Link>
            <button
              onClick={() => {
                logout();
                navigate('/login');
              }}
              className="text-xs text-red-450 font-semibold"
            >
              Sair
            </button>
          </div>
        </header>

        {/* Content Outlet */}
        <main className="flex-1 p-6 md:p-10 max-w-7xl w-full mx-auto">
          <div className="mb-6 flex justify-between items-center hidden md:flex">
            <div>
              <h1 className="text-2xl font-extrabold tracking-tight text-slate-100">
                {menuItems.find((item) => item.path === location.pathname)?.label || 'Painel Admin'}
              </h1>
              <p className="text-xs text-red-450 mt-1">Modo Super Administrador do Sistema</p>
            </div>
          </div>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
