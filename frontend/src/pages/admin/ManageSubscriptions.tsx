import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface Subscription {
  id: string;
  organization_id: string;
  organization_name: string;
  plan_id: string;
  plan_name: string;
  status: string;
  payment_provider: string;
  external_subscription_id: string;
  current_period_start: string;
  current_period_end: string;
  created_at: string;
}

interface Organization {
  id: string;
  name: string;
  slug: string;
}

interface Plan {
  id: string;
  name: string;
  price_monthly: string;
  currency: string;
}

export default function ManageSubscriptions() {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // Form State
  const [selectedOrg, setSelectedOrg] = useState('');
  const [selectedPlan, setSelectedPlan] = useState('');
  const [durationDays, setDurationDays] = useState(30);
  const [submitting, setSubmitting] = useState(false);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    setError('');
    try {
      const [subsRes, orgsRes, plansRes] = await Promise.all([
        api.get('/admin/subscriptions'),
        api.get('/admin/organizations?limit=100'),
        api.get('/admin/plans?limit=100'),
      ]);
      setSubscriptions(subsRes.data.data.subscriptions || []);
      setOrganizations(orgsRes.data.data.organizations || []);
      setPlans(plansRes.data.data.plans || []);
    } catch (err) {
      setError('Erro ao carregar dados de assinaturas.');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateManualSub = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError('');
    setSuccess('');

    try {
      await api.post('/admin/subscriptions/manual', {
        organization_id: selectedOrg,
        plan_id: selectedPlan,
        duration_days: Number(durationDays),
      });

      setSuccess('Assinatura manual criada com sucesso e créditos liberados!');
      setShowModal(false);
      // Reset form
      setSelectedOrg('');
      setSelectedPlan('');
      setDurationDays(30);
      
      // Reload subscriptions list
      const subsRes = await api.get('/admin/subscriptions');
      setSubscriptions(subsRes.data.data.subscriptions || []);
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Erro ao criar assinatura manual.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-red-500"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-xl font-bold">Assinaturas Ativas</h2>
          <p className="text-xs text-slate-400 mt-1">Gerenciamento de planos vinculados e ativação manual de licenças.</p>
        </div>
        <button
          onClick={() => setShowModal(true)}
          className="bg-red-700 hover:bg-red-650 text-white font-bold py-2 px-4 rounded-xl text-xs flex items-center gap-2 transition-all cursor-pointer shadow-lg shadow-red-950/20"
        >
          ➕ Nova Assinatura Manual
        </button>
      </div>

      {error && (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
          ⚠️ {error}
        </div>
      )}
      {success && (
        <div className="p-4 rounded-xl bg-green-500/10 border border-green-500/30 text-green-400 text-sm">
          ✅ {success}
        </div>
      )}

      {/* Subscriptions List */}
      <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-950 text-slate-400 text-xs font-semibold uppercase border-b border-slate-800">
                <th className="py-4 px-6">Empresa / Tenant</th>
                <th className="py-4 px-6">Plano Assinado</th>
                <th className="py-4 px-6 text-center">Status</th>
                <th className="py-4 px-6 text-center">Gateway / Provider</th>
                <th className="py-4 px-6">Período de Validade</th>
                <th className="py-4 px-6 text-right">Data de Ativação</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/50 text-sm">
              {subscriptions.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-slate-500 text-sm">
                    Nenhuma assinatura ativa encontrada no sistema.
                  </td>
                </tr>
              ) : (
                subscriptions.map((sub) => (
                  <tr key={sub.id} className="hover:bg-slate-850/40 transition-colors">
                    <td className="py-4 px-6">
                      <p className="font-bold text-white">{sub.organization_name || 'N/A'}</p>
                      <p className="text-[10px] text-slate-450 font-mono">{sub.organization_id}</p>
                    </td>
                    <td className="py-4 px-6 font-semibold text-slate-200">{sub.plan_name}</td>
                    <td className="py-4 px-6 text-center">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        sub.status === 'ACTIVE'
                          ? 'bg-green-500/10 text-green-400'
                          : 'bg-yellow-500/10 text-yellow-400'
                      }`}>
                        {sub.status}
                      </span>
                    </td>
                    <td className="py-4 px-6 text-center font-bold text-xs text-slate-400">{sub.payment_provider}</td>
                    <td className="py-4 px-6 text-xs text-slate-350">
                      <p>De: {new Date(sub.current_period_start).toLocaleDateString('pt-BR')}</p>
                      <p className="font-semibold text-red-400">Até: {new Date(sub.current_period_end).toLocaleDateString('pt-BR')}</p>
                    </td>
                    <td className="py-4 px-6 text-right text-xs text-slate-400">
                      {new Date(sub.created_at).toLocaleString('pt-BR')}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Manual Subscription Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-6 animate-fade-in">
          <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 max-w-md w-full space-y-6 relative shadow-2xl">
            <div>
              <h3 className="text-lg font-bold text-white">Criar Assinatura Manual</h3>
              <p className="text-xs text-slate-400 mt-1">Selecione o cliente, o plano e defina a vigência. Isso adicionará e registrará os créditos de IA automaticamente.</p>
            </div>

            <form onSubmit={handleCreateManualSub} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2">
                  Empresa / Workspace
                </label>
                <select
                  value={selectedOrg}
                  onChange={(e) => setSelectedOrg(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-sm focus:border-red-500 focus:outline-none text-slate-200"
                  required
                >
                  <option value="">Selecione uma empresa...</option>
                  {organizations.map((org) => (
                    <option key={org.id} value={org.id}>
                      {org.name} ({org.slug})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2">
                  Plano
                </label>
                <select
                  value={selectedPlan}
                  onChange={(e) => setSelectedPlan(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 text-sm focus:border-red-500 focus:outline-none text-slate-200"
                  required
                >
                  <option value="">Selecione um plano...</option>
                  {plans.map((plan) => (
                    <option key={plan.id} value={plan.id}>
                      {plan.name} - {plan.currency} {Number(plan.price_monthly).toFixed(2)}/mês
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2">
                  Duração da Licença (Dias)
                </label>
                <input
                  type="number"
                  value={durationDays}
                  onChange={(e) => setDurationDays(Number(e.target.value))}
                  min={1}
                  className="w-full bg-slate-950 border border-slate-800 focus:border-red-500 rounded-xl px-4 py-3 text-sm focus:outline-none text-slate-200"
                  required
                />
              </div>

              <div className="flex gap-3 justify-end pt-4">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-350 text-xs font-bold rounded-xl transition-all cursor-pointer"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="px-4 py-2 bg-red-700 hover:bg-red-600 disabled:bg-red-900 text-white text-xs font-bold rounded-xl transition-all cursor-pointer"
                >
                  {submitting ? 'Processando...' : 'Ativar Licença'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
