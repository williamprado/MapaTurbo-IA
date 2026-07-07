import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../../services/api';

interface Member {
  id: string;
  email: string;
  name?: string;
  role: string;
  status: string;
}

interface MindMap {
  id: string;
  title: string;
  status: string;
  created_at: string;
}

interface UploadFile {
  id: string;
  filename: string;
  size: number;
  status: string;
  created_at: string;
}

interface Invoice {
  id: string;
  amount: number;
  due_date: string;
  status: string;
}

interface AuditLog {
  id: string;
  actor_email?: string;
  action: string;
  entity_type: string;
  ip?: string;
  created_at: string;
}

interface OrgDetails {
  organization: {
    id: string;
    name: string;
    slug: string;
    status: string;
    created_at: string;
  };
  plan: {
    name: string;
    id: string;
    status: string;
    current_period_end?: string;
  };
  users: Member[];
  maps: MindMap[];
  uploads: UploadFile[];
  invoices: Invoice[];
  audit_logs: AuditLog[];
}

export default function OrganizationSummary() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [details, setDetails] = useState<OrgDetails | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('overview');
  const [error, setError] = useState('');

  useEffect(() => {
    if (id) {
      fetchSummary();
    }
  }, [id]);

  const fetchSummary = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api.get(`/admin/organizations/${id}/summary`);
      setDetails(res.data.data);
    } catch (err) {
      console.error('Error fetching org summary:', err);
      setError('Falha ao carregar os dados detalhados do inquilino (tenant).');
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center py-24">
        <div className="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-purple-500"></div>
      </div>
    );
  }

  if (error || !details) {
    return (
      <div className="space-y-4">
        <button onClick={() => navigate('/admin/organizations')} className="text-slate-400 hover:text-white text-xs">&larr; Voltar para Empresas</button>
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
          ⚠️ {error || 'Organização não encontrada.'}
        </div>
      </div>
    );
  }

  const { organization, plan, users, maps, uploads, invoices, audit_logs } = details;

  return (
    <div className="space-y-8">
      {/* Header and Back Button */}
      <div className="space-y-2">
        <button
          onClick={() => navigate('/admin/organizations')}
          className="text-slate-400 hover:text-white text-xs flex items-center gap-1 cursor-pointer"
        >
          &larr; Voltar para Lista de Empresas
        </button>
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <div>
            <h2 className="text-2xl font-black text-slate-100">{organization.name}</h2>
            <p className="text-xs text-slate-500">Slug: <span className="font-mono text-slate-400">{organization.slug}</span> &bull; ID: <span className="font-mono text-slate-400">{organization.id}</span></p>
          </div>
          <span
            className={`px-3 py-1 rounded-full text-xs font-bold uppercase ${
              organization.status === 'ACTIVE'
                ? 'bg-green-950 text-green-400 border border-green-500/20'
                : 'bg-slate-800 text-slate-400'
            }`}
          >
            {organization.status}
          </span>
        </div>
      </div>

      {/* Tabs list */}
      <div className="flex border-b border-slate-850 gap-4">
        {['overview', 'members', 'maps', 'uploads', 'billing', 'audit_logs'].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`pb-3 text-xs font-semibold capitalize border-b-2 transition-all cursor-pointer ${
              activeTab === tab
                ? 'border-purple-500 text-purple-400'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            {tab === 'overview' ? 'Visão Geral' :
             tab === 'members' ? 'Membros' :
             tab === 'maps' ? 'Mapas Mentais' :
             tab === 'uploads' ? 'Documentos' :
             tab === 'billing' ? 'Faturamento' : 'Logs de Auditoria'}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className="space-y-6">
        {activeTab === 'overview' && (
          <div className="grid md:grid-cols-2 gap-6">
            {/* Plan and billing overview card */}
            <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
              <h3 className="font-bold text-slate-200 text-sm">Plano de Assinatura</h3>
              <div className="grid grid-cols-2 gap-4 text-xs">
                <div>
                  <p className="text-slate-500">Plano Ativo</p>
                  <p className="font-bold text-slate-200">{plan.name}</p>
                </div>
                <div>
                  <p className="text-slate-500">Status da Assinatura</p>
                  <p className="font-bold text-purple-400 uppercase">{plan.status}</p>
                </div>
                {plan.current_period_end && (
                  <div className="col-span-2">
                    <p className="text-slate-500">Renovação / Fim do Período</p>
                    <p className="font-bold text-slate-300">{new Date(plan.current_period_end).toLocaleString('pt-BR')}</p>
                  </div>
                )}
              </div>
            </div>

            {/* General metrics card */}
            <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
              <h3 className="font-bold text-slate-200 text-sm">Uso Geral</h3>
              <div className="grid grid-cols-3 gap-4 text-xs">
                <div className="p-3 bg-slate-950 rounded-xl text-center">
                  <p className="text-slate-500 text-[10px] uppercase font-semibold">Membros</p>
                  <p className="text-lg font-bold text-slate-200 mt-1">{users.length}</p>
                </div>
                <div className="p-3 bg-slate-950 rounded-xl text-center">
                  <p className="text-slate-500 text-[10px] uppercase font-semibold">Mapas</p>
                  <p className="text-lg font-bold text-slate-200 mt-1">{maps.length}</p>
                </div>
                <div className="p-3 bg-slate-950 rounded-xl text-center">
                  <p className="text-slate-500 text-[10px] uppercase font-semibold">Arquivos</p>
                  <p className="text-lg font-bold text-slate-200 mt-1">{uploads.length}</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'members' && (
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
            <h3 className="font-bold text-slate-100 text-sm mb-4">Lista de Membros</h3>
            {users.length === 0 ? (
              <p className="text-xs text-slate-500">Nenhum membro cadastrado.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-500 font-bold uppercase tracking-wider">
                      <th className="pb-3">E-mail</th>
                      <th className="pb-3">Permissão (Cargo)</th>
                      <th className="pb-3 text-right">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-850">
                    {users.map((m) => (
                      <tr key={m.id} className="text-slate-350">
                        <td className="py-3 font-bold text-slate-200">{m.email}</td>
                        <td className="py-3">{m.role}</td>
                        <td className="py-3 text-right">
                          <span className="bg-slate-800 text-slate-300 text-[9px] uppercase font-bold px-2 py-0.5 rounded">
                            {m.status ?? 'ACTIVE'}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {activeTab === 'maps' && (
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
            <h3 className="font-bold text-slate-100 text-sm mb-4">Mapas Mentais Criados</h3>
            {maps.length === 0 ? (
              <p className="text-xs text-slate-500">Nenhum mapa criado.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-500 font-bold uppercase tracking-wider">
                      <th className="pb-3">Título</th>
                      <th className="pb-3">Status</th>
                      <th className="pb-3 text-right">Criação</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-850">
                    {maps.map((map) => (
                      <tr key={map.id} className="text-slate-350">
                        <td className="py-3 font-bold text-slate-200">🧠 {map.title}</td>
                        <td className="py-3">
                          <span className="px-2 py-0.5 rounded bg-purple-950 text-purple-400 font-bold text-[9px]">
                            {map.status}
                          </span>
                        </td>
                        <td className="py-3 text-right text-slate-550">
                          {new Date(map.created_at).toLocaleDateString('pt-BR')}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {activeTab === 'uploads' && (
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
            <h3 className="font-bold text-slate-100 text-sm mb-4">Arquivos e Documentos</h3>
            {uploads.length === 0 ? (
              <p className="text-xs text-slate-500">Nenhum upload de arquivo realizado.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-500 font-bold uppercase tracking-wider">
                      <th className="pb-3">Nome do Arquivo</th>
                      <th className="pb-3">Tamanho</th>
                      <th className="pb-3">Status</th>
                      <th className="pb-3 text-right">Enviado em</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-850">
                    {uploads.map((f) => (
                      <tr key={f.id} className="text-slate-350">
                        <td className="py-3 font-bold text-slate-200">📄 {f.filename}</td>
                        <td className="py-3 font-mono text-slate-400">{formatBytes(f.size)}</td>
                        <td className="py-3">
                          <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-bold uppercase text-[9px]">
                            {f.status}
                          </span>
                        </td>
                        <td className="py-3 text-right text-slate-550">
                          {new Date(f.created_at).toLocaleString('pt-BR')}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {activeTab === 'billing' && (
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
            <h3 className="font-bold text-slate-100 text-sm mb-4">Invoices / Faturas</h3>
            {invoices.length === 0 ? (
              <p className="text-xs text-slate-500">Nenhuma fatura registrada.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-slate-800 text-slate-500 font-bold uppercase tracking-wider">
                      <th className="pb-3">Valor</th>
                      <th className="pb-3">Vencimento</th>
                      <th className="pb-3 text-right">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-850">
                    {invoices.map((inv) => (
                      <tr key={inv.id} className="text-slate-350">
                        <td className="py-3 font-bold text-slate-200">
                          R$ {inv.amount.toLocaleString('pt-BR', { minimumFractionDigits: 2 })}
                        </td>
                        <td className="py-3">{new Date(inv.due_date).toLocaleDateString('pt-BR')}</td>
                        <td className="py-3 text-right">
                          <span
                            className={`px-2 py-0.5 rounded font-bold text-[9px] uppercase ${
                              inv.status === 'PAID'
                                ? 'bg-green-950 text-green-400'
                                : 'bg-yellow-950 text-yellow-400'
                            }`}
                          >
                            {inv.status}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {activeTab === 'audit_logs' && (
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
            <h3 className="font-bold text-slate-100 text-sm mb-4">Histórico Recente de Auditoria</h3>
            {audit_logs.length === 0 ? (
              <p className="text-xs text-slate-500">Nenhum log gravado.</p>
            ) : (
              <div className="space-y-3">
                {audit_logs.map((log) => (
                  <div key={log.id} className="p-3 bg-slate-950 rounded-xl border border-slate-850 text-xs flex justify-between items-center">
                    <div>
                      <span className="bg-slate-800 text-slate-350 border border-slate-700/50 text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded mr-3">
                        {log.ip || '127.0.0.1'}
                      </span>
                      <span className="font-bold text-slate-200">{log.action}</span>
                      <span className="text-slate-550 ml-2">por {log.actor_email || 'Sistema'}</span>
                    </div>
                    <span className="text-slate-550 text-[10px]">
                      {new Date(log.created_at).toLocaleString('pt-BR')}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
