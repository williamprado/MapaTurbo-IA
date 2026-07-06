import { useState, useEffect } from 'react';
import { api } from '../../services/api';

export default function AdminDashboard() {
  const [stats, setStats] = useState({
    organizations: 0,
    users: 0,
    plans: 0,
    logs: 0,
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Fetch stats in parallel
    const fetchData = async () => {
      try {
        const [orgsRes, usersRes, plansRes] = await Promise.all([
          api.get('/admin/organizations?limit=1'),
          api.get('/admin/users?limit=1'),
          api.get('/admin/plans?limit=1'),
        ]);

        setStats({
          organizations: orgsRes.data.data.total || 0,
          users: usersRes.data.data.total || 0,
          plans: plansRes.data.data.total || 0,
          logs: 1, // Mock
        });
      } catch (err) {
        console.error('Failed to load stats', err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  return (
    <div className="space-y-8">
      {/* Stats Cards */}
      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Empresas (Tenants)
          </p>
          <h3 className="text-2xl font-bold text-slate-100">{loading ? '...' : stats.organizations}</h3>
          <p className="text-[10px] text-slate-400 mt-2">Inquilinos cadastrados no SaaS</p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Usuários Globais
          </p>
          <h3 className="text-2xl font-bold text-slate-100">{loading ? '...' : stats.users}</h3>
          <p className="text-[10px] text-slate-400 mt-2">Contas criadas na plataforma</p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Planos de Assinatura
          </p>
          <h3 className="text-2xl font-bold text-slate-100">{loading ? '...' : stats.plans}</h3>
          <p className="text-[10px] text-slate-400 mt-2">Modelos de preços configurados</p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-850">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Eventos Recentes (Audit)
          </p>
          <h3 className="text-2xl font-bold text-slate-100">Bootstrap</h3>
          <p className="text-[10px] text-slate-400 mt-2">Logs de auditoria de segurança</p>
        </div>
      </div>

      {/* Audit Log Box */}
      <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
        <h3 className="font-bold text-slate-100 mb-4">Eventos Críticos de Auditoria</h3>
        <div className="space-y-3">
          <div className="p-4 bg-slate-950 border border-slate-850 rounded-xl flex items-center justify-between text-xs">
            <div>
              <span className="bg-red-950 text-red-400 border border-red-500/20 text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded mr-3">
                SYSTEM
              </span>
              <span className="font-bold text-slate-200">SUPER_ADMIN_CREATED</span>
              <span className="text-slate-500 ml-2">Seed do Administrador inicial executado com sucesso</span>
            </div>
            <span className="text-slate-500 text-[10px]">{new Date().toLocaleString()}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
