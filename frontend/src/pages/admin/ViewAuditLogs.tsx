import { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface AuditLog {
  id: string;
  actor_user_id: string;
  actor_email?: string;
  organization_id?: string;
  organization_name?: string;
  action: string;
  entity_type: string;
  entity_id?: string;
  ip?: string;
  user_agent?: string;
  created_at: string;
}

export default function ViewAuditLogs() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [limit] = useState(25);
  const [offset, setOffset] = useState(0);

  useEffect(() => {
    fetchLogs();
  }, [offset]);

  const fetchLogs = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await api.get(`/admin/audit-logs?limit=${limit}&offset=${offset}`);
      setLogs(response.data.data.logs || []);
      setTotal(response.data.data.total || 0);
    } catch (err) {
      setError('Erro ao carregar logs de auditoria.');
    } finally {
      setLoading(false);
    }
  };

  const getActionBadgeClass = (action: string) => {
    if (action.includes('BLOCKED') || action.includes('FAILED')) {
      return 'bg-red-500/10 text-red-400 border border-red-500/20';
    }
    if (action.includes('CREATED') || action.includes('ADDED') || action.includes('SUCCESS')) {
      return 'bg-green-500/10 text-green-400 border border-green-500/20';
    }
    if (action.includes('UPDATED')) {
      return 'bg-yellow-500/10 text-yellow-400 border border-yellow-500/20';
    }
    return 'bg-slate-800 text-slate-350 border border-slate-700/50';
  };

  if (loading && logs.length === 0) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-red-500"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">Rastro de Auditoria (Security Audit Logs)</h2>
        <p className="text-xs text-slate-400 mt-1">Logs em tempo real de ações administrativas e alterações de segurança do MapaTurbo IA.</p>
      </div>

      {error && (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
          ⚠️ {error}
        </div>
      )}

      {/* Audit Logs Table */}
      <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-950 text-slate-400 text-xs font-semibold uppercase border-b border-slate-800">
                <th className="py-4 px-6">Usuário (Autor)</th>
                <th className="py-4 px-6">Empresa Scope</th>
                <th className="py-4 px-6">Ação Realizada</th>
                <th className="py-4 px-6">Entidade Alvo</th>
                <th className="py-4 px-6">Conexão (IP/Agent)</th>
                <th className="py-4 px-6 text-right">Data/Hora</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/50 text-xs">
              {logs.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-slate-500 text-sm">
                    Nenhum log de auditoria disponível no momento.
                  </td>
                </tr>
              ) : (
                logs.map((log) => (
                  <tr key={log.id} className="hover:bg-slate-850/40 transition-colors">
                    <td className="py-4 px-6">
                      <p className="font-semibold text-slate-200">{log.actor_email || 'Sistema'}</p>
                      <p className="text-[9px] text-slate-500 font-mono">{log.actor_user_id}</p>
                    </td>
                    <td className="py-4 px-6 text-slate-350">
                      {log.organization_name ? (
                        <div>
                          <p className="font-semibold text-slate-300">{log.organization_name}</p>
                          <p className="text-[9px] text-slate-500 font-mono">{log.organization_id}</p>
                        </div>
                      ) : (
                        <span className="text-slate-500 font-bold uppercase tracking-wider text-[9px]">Global</span>
                      )}
                    </td>
                    <td className="py-4 px-6">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-md font-mono font-bold text-[10px] ${getActionBadgeClass(log.action)}`}>
                        {log.action}
                      </span>
                    </td>
                    <td className="py-4 px-6">
                      <p className="text-slate-300 font-medium">{log.entity_type}</p>
                      {log.entity_id && (
                        <p className="text-[9px] text-slate-500 font-mono mt-0.5">{log.entity_id}</p>
                      )}
                    </td>
                    <td className="py-4 px-6 text-slate-400">
                      <p className="font-mono text-slate-300">{log.ip || '0.0.0.0'}</p>
                      {log.user_agent && (
                        <p className="text-[9px] text-slate-500 truncate max-w-[150px] mt-0.5" title={log.user_agent}>
                          {log.user_agent}
                        </p>
                      )}
                    </td>
                    <td className="py-4 px-6 text-right text-slate-400">
                      {new Date(log.created_at).toLocaleString('pt-BR')}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination controls */}
        {total > limit && (
          <div className="p-4 bg-slate-950/40 border-t border-slate-800 flex items-center justify-between">
            <span className="text-xs text-slate-400">
              Mostrando {offset + 1} - {Math.min(offset + limit, total)} de {total} logs
            </span>
            <div className="flex gap-2">
              <button
                disabled={offset === 0}
                onClick={() => setOffset((prev) => Math.max(0, prev - limit))}
                className="px-3 py-1 bg-slate-800 hover:bg-slate-700 disabled:opacity-50 text-xs font-bold rounded-lg cursor-pointer transition-all"
              >
                Anterior
              </button>
              <button
                disabled={offset + limit >= total}
                onClick={() => setOffset((prev) => prev + limit)}
                className="px-3 py-1 bg-slate-800 hover:bg-slate-700 disabled:opacity-50 text-xs font-bold rounded-lg cursor-pointer transition-all"
              >
                Próximo
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
