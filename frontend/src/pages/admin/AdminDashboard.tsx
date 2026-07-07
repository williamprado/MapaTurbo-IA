import { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface GenJob {
  id: string;
  type: string;
  status: string;
  error?: string;
  credits_cost: number;
  created_at: string;
}

interface WebhookEvent {
  id: string;
  provider: string;
  event_type: string;
  status: string;
  error?: string;
  created_at: string;
}

interface AdminStats {
  total_organizations: number;
  active_organizations: number;
  total_users: number;
  active_users: number;
  total_mind_maps: number;
  total_uploads: number;
  credits_consumed: number;
  active_subscriptions: number;
  revenue_estimated: number;
  recent_ia_errors: GenJob[];
  recent_webhook_errors: WebhookEvent[];
}

export default function AdminDashboard() {
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    fetchAdminStats();
  }, []);

  const fetchAdminStats = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api.get('/admin/dashboard');
      setStats(res.data.data);
    } catch (err) {
      console.error('Error fetching admin dashboard stats:', err);
      setError('Falha ao carregar métricas administrativas globais.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-8">
      {error && (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
          ⚠️ {error}
        </div>
      )}

      {/* Grid of Global Stats */}
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">Empresas (Tenants)</p>
          <h3 className="text-3xl font-extrabold text-slate-100">{loading ? '...' : stats?.total_organizations}</h3>
          <p className="text-[10px] text-slate-400 mt-2">Ativas: <span className="text-green-400 font-semibold">{stats?.active_organizations}</span></p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">Usuários Globais</p>
          <h3 className="text-3xl font-extrabold text-slate-100">{loading ? '...' : stats?.total_users}</h3>
          <p className="text-[10px] text-slate-400 mt-2">Ativos (Sessão): <span className="text-purple-400 font-semibold">{stats?.active_users}</span></p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">Mapas & Uploads</p>
          <h3 className="text-3xl font-extrabold text-slate-100">
            {loading ? '...' : `${stats?.total_mind_maps} / ${stats?.total_uploads}`}
          </h3>
          <p className="text-[10px] text-slate-400 mt-2">Total de mapas e arquivos PDF salvos</p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">Assinaturas Ativas</p>
          <h3 className="text-3xl font-extrabold text-indigo-400">{loading ? '...' : stats?.active_subscriptions}</h3>
          <p className="text-[10px] text-slate-400 mt-2">Faturamento Est.: <span className="text-green-400 font-bold">R$ {stats?.revenue_estimated.toLocaleString('pt-BR', { minimumFractionDigits: 2 })}</span></p>
        </div>
      </div>

      {/* Second Line: Credits Consumed */}
      <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
        <h3 className="font-semibold text-slate-400 text-xs uppercase tracking-wider mb-2">Créditos de IA Consumidos Globalmente</h3>
        <h2 className="text-4xl font-black text-white">{loading ? '...' : `${stats?.credits_consumed} CRD`}</h2>
        <p className="text-xs text-slate-500 mt-1">Soma de todos os débitos das empresas após execuções bem-sucedidas de IA.</p>
      </div>

      {/* Error Boards Grid */}
      <div className="grid lg:grid-cols-2 gap-8">
        {/* AI Errors */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between text-sm uppercase tracking-wider">
            <span>Falhas Recentes de Geração IA (Worker)</span>
            <span className="h-2 w-2 rounded-full bg-red-500 animate-pulse" />
          </h3>

          {loading ? (
            <p className="text-xs text-slate-500 py-4">Carregando erros...</p>
          ) : !stats?.recent_ia_errors || stats.recent_ia_errors.length === 0 ? (
            <p className="text-xs text-slate-500 py-10 text-center">Nenhum erro de IA registrado recentemente.</p>
          ) : (
            <div className="space-y-3">
              {stats.recent_ia_errors.map((job) => (
                <div key={job.id} className="p-3 bg-slate-950 rounded-xl border border-slate-850 hover:border-slate-800 transition-all text-xs space-y-1">
                  <div className="flex justify-between font-bold text-slate-300">
                    <span>{job.type}</span>
                    <span className="text-purple-400">{job.credits_cost} CRD</span>
                  </div>
                  <p className="text-red-400 font-mono text-[10px]">{job.error}</p>
                  <p className="text-[9px] text-slate-500 text-right">{new Date(job.created_at).toLocaleString('pt-BR')}</p>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Webhook Errors */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between text-sm uppercase tracking-wider">
            <span>Erros de Webhooks Financeiros</span>
            <span className="h-2 w-2 rounded-full bg-red-500 animate-pulse" />
          </h3>

          {loading ? (
            <p className="text-xs text-slate-500 py-4">Carregando erros...</p>
          ) : !stats?.recent_webhook_errors || stats.recent_webhook_errors.length === 0 ? (
            <p className="text-xs text-slate-500 py-10 text-center">Nenhum erro de webhook registrado recentemente.</p>
          ) : (
            <div className="space-y-3">
              {stats.recent_webhook_errors.map((evt) => (
                <div key={evt.id} className="p-3 bg-slate-950 rounded-xl border border-slate-850 hover:border-slate-800 transition-all text-xs space-y-1">
                  <div className="flex justify-between font-bold text-slate-300">
                    <span>{evt.provider.toUpperCase()} &bull; {evt.event_type}</span>
                    <span className="text-red-400 text-[10px]">FAILED</span>
                  </div>
                  <p className="text-red-400 font-mono text-[10px]">{evt.error}</p>
                  <p className="text-[9px] text-slate-500 text-right">{new Date(evt.created_at).toLocaleString('pt-BR')}</p>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
