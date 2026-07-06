import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface PaymentProvider {
  id: string;
  name: string;
  slug: string;
  apiKey: string;
  webhookSecret: string;
  isActive: boolean;
  mode: string;
}

interface Invoice {
  id: string;
  organization_id: string;
  organization_name: string;
  amount: string;
  currency: string;
  status: string;
  external_invoice_id: string;
  due_date: string;
  billing_type: string;
  invoice_url: string;
  paid_at: string;
  created_at: string;
}

interface Transaction {
  id: string;
  organization_name: string;
  amount: string;
  provider: string;
  external_transaction_id: string;
  status: string;
  payment_method: string;
  created_at: string;
}

interface WebhookEvent {
  id: string;
  provider: string;
  event_type: string;
  external_id: string;
  status: string;
  error: string;
  processed_at: string;
  created_at: string;
}

export default function ManagePayments() {
  const [activeTab, setActiveTab] = useState<'providers' | 'invoices' | 'transactions' | 'webhooks'>('providers');
  const [providers, setProviders] = useState<PaymentProvider[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [webhooks, setWebhooks] = useState<WebhookEvent[]>([]);
  
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // Modals / Edits
  const [editingProvider, setEditingProvider] = useState<PaymentProvider | null>(null);
  const [providerForm, setProviderForm] = useState({
    apiKey: '',
    webhookSecret: '',
    isActive: true,
    mode: 'sandbox',
  });
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);

  // Webhook JSON Viewer Modal
  const [viewingPayload, setViewingPayload] = useState<string | null>(null);

  useEffect(() => {
    loadTabContent();
  }, [activeTab]);

  const loadTabContent = async () => {
    setLoading(true);
    setError('');
    try {
      if (activeTab === 'providers') {
        const res = await api.get('/admin/payments/providers');
        setProviders(res.data.data || []);
      } else if (activeTab === 'invoices') {
        const res = await api.get('/admin/payments/invoices');
        setInvoices(res.data.data || []);
      } else if (activeTab === 'transactions') {
        const res = await api.get('/admin/payments/transactions');
        setTransactions(res.data.data || []);
      } else if (activeTab === 'webhooks') {
        const res = await api.get('/admin/payments/webhook-events');
        setWebhooks(res.data.data || []);
      }
    } catch (err: any) {
      setError('Erro ao carregar dados do faturamento.');
    } finally {
      setLoading(false);
    }
  };

  const handleOpenEditProvider = (p: PaymentProvider) => {
    setEditingProvider(p);
    setProviderForm({
      apiKey: p.apiKey || '********',
      webhookSecret: p.webhookSecret || '********',
      isActive: p.isActive,
      mode: p.mode,
    });
    setIsEditModalOpen(true);
  };

  const handleSaveProvider = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingProvider) return;
    setError('');
    setSuccess('');

    try {
      await api.patch(`/admin/payments/providers/${editingProvider.id}`, providerForm);
      setSuccess('Credenciais do gateway atualizadas com sucesso.');
      setIsEditModalOpen(false);
      loadTabContent();
    } catch (err: any) {
      setError('Falha ao atualizar parâmetros do gateway.');
    }
  };

  const handleInspectPayload = async (id: string) => {
    try {
      const res = await api.get('/admin/payments/webhook-events');
      const item = (res.data.data || []).find((x: any) => x.id === id);
      if (item) {
        // Formata o JSON bruto se disponível
        setViewingPayload(JSON.stringify(item.payload || {}, null, 2));
      } else {
        setViewingPayload('Payload indisponível');
      }
    } catch {
      setViewingPayload('Erro ao obter payload do banco.');
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-100">Faturamento & Cobranças</h1>
        <p className="text-slate-400 text-xs mt-1">Gerencie gateways de pagamento, faturas de clientes, transações em tempo real e webhooks.</p>
      </div>

      {error && (
        <div className="p-4 bg-red-500/10 border border-red-500/35 rounded-xl text-red-400 text-xs">
          ⚠️ {error}
        </div>
      )}
      {success && (
        <div className="p-4 bg-green-500/10 border border-green-500/35 rounded-xl text-green-400 text-xs">
          ✅ {success}
        </div>
      )}

      {/* Tabs Menu */}
      <div className="flex border-b border-slate-800 gap-6 text-xs font-bold text-slate-400">
        <button
          onClick={() => setActiveTab('providers')}
          className={`pb-3 ${activeTab === 'providers' ? 'text-purple-400 border-b-2 border-purple-600' : 'hover:text-slate-200'} cursor-pointer`}
        >
          Gateways (Asaas/Stripe)
        </button>
        <button
          onClick={() => setActiveTab('invoices')}
          className={`pb-3 ${activeTab === 'invoices' ? 'text-purple-400 border-b-2 border-purple-600' : 'hover:text-slate-200'} cursor-pointer`}
        >
          Faturas (Invoices)
        </button>
        <button
          onClick={() => setActiveTab('transactions')}
          className={`pb-3 ${activeTab === 'transactions' ? 'text-purple-400 border-b-2 border-purple-600' : 'hover:text-slate-200'} cursor-pointer`}
        >
          Transações
        </button>
        <button
          onClick={() => setActiveTab('webhooks')}
          className={`pb-3 ${activeTab === 'webhooks' ? 'text-purple-400 border-b-2 border-purple-600' : 'hover:text-slate-200'} cursor-pointer`}
        >
          Webhooks Processados
        </button>
      </div>

      {loading ? (
        <p className="text-slate-400 text-xs">Carregando dados...</p>
      ) : (
        <div>
          {/* 1. Providers Tab */}
          {activeTab === 'providers' && (
            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {providers.map((p) => (
                <div key={p.id} className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col justify-between space-y-4">
                  <div>
                    <div className="flex justify-between items-start">
                      <h3 className="font-bold text-slate-100 uppercase">{p.name}</h3>
                      <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${p.isActive ? 'bg-green-950 text-green-400' : 'bg-slate-800 text-slate-400'}`}>
                        {p.isActive ? 'Ativo' : 'Inativo'}
                      </span>
                    </div>
                    <p className="text-[10px] text-slate-500 font-mono mt-0.5">{p.slug}</p>

                    <div className="mt-4 space-y-2 text-xs text-slate-400">
                      <p><strong className="text-slate-300">Ambiente:</strong> <span className="uppercase text-purple-400">{p.mode}</span></p>
                      <p><strong className="text-slate-300">API Key:</strong> <code className="text-slate-500">{p.apiKey}</code></p>
                      <p><strong className="text-slate-300">Webhook Secret:</strong> <code className="text-slate-500">{p.webhookSecret}</code></p>
                    </div>
                  </div>

                  <button
                    onClick={() => handleOpenEditProvider(p)}
                    className="w-full py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-bold rounded-lg transition-all cursor-pointer text-center"
                  >
                    Configurar Credenciais
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* 2. Invoices Tab */}
          {activeTab === 'invoices' && (
            <div className="overflow-x-auto rounded-2xl border border-slate-800 bg-slate-900">
              <table className="w-full border-collapse text-left text-xs">
                <thead>
                  <tr className="border-b border-slate-800 bg-slate-950 font-bold text-slate-300">
                    <th className="p-4">Tenant / Empresa</th>
                    <th className="p-4">ID Fatura</th>
                    <th className="p-4">Valor</th>
                    <th className="p-4">Método</th>
                    <th className="p-4">Vencimento</th>
                    <th className="p-4">Data Pagamento</th>
                    <th className="p-4 text-center">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800 text-slate-400">
                  {invoices.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="p-4 text-center text-slate-500">Nenhuma fatura encontrada.</td>
                    </tr>
                  ) : (
                    invoices.map((inv) => (
                      <tr key={inv.id} className="hover:bg-slate-950/40">
                        <td className="p-4 font-bold text-slate-200">{inv.organization_name}</td>
                        <td className="p-4 font-mono text-[10px]">{inv.external_invoice_id || inv.id}</td>
                        <td className="p-4 text-purple-400 font-bold">{inv.amount} {inv.currency}</td>
                        <td className="p-4 font-bold uppercase">{inv.billing_type || 'N/A'}</td>
                        <td className="p-4">{inv.due_date ? new Date(inv.due_date).toLocaleDateString() : 'N/A'}</td>
                        <td className="p-4">{inv.paid_at ? new Date(inv.paid_at).toLocaleDateString() : '-'}</td>
                        <td className="p-4 text-center">
                          <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                            inv.status === 'PAID' ? 'bg-green-950 text-green-400' :
                            inv.status === 'PENDING' ? 'bg-yellow-950/60 text-yellow-400' : 'bg-red-950 text-red-400'
                          }`}>
                            {inv.status}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}

          {/* 3. Transactions Tab */}
          {activeTab === 'transactions' && (
            <div className="overflow-x-auto rounded-2xl border border-slate-800 bg-slate-900">
              <table className="w-full border-collapse text-left text-xs">
                <thead>
                  <tr className="border-b border-slate-800 bg-slate-950 font-bold text-slate-300">
                    <th className="p-4">Empresa</th>
                    <th className="p-4">Gateway</th>
                    <th className="p-4">Transação Externa</th>
                    <th className="p-4">Valor</th>
                    <th className="p-4">Método</th>
                    <th className="p-4">Status</th>
                    <th className="p-4">Criado em</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800 text-slate-400">
                  {transactions.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="p-4 text-center text-slate-500">Nenhuma transação financeira registrada.</td>
                    </tr>
                  ) : (
                    transactions.map((t) => (
                      <tr key={t.id} className="hover:bg-slate-950/40">
                        <td className="p-4 font-bold text-slate-200">{t.organization_name}</td>
                        <td className="p-4 uppercase font-bold text-[10px]">{t.provider}</td>
                        <td className="p-4 font-mono text-[10px] text-slate-500">{t.external_transaction_id}</td>
                        <td className="p-4 font-bold text-slate-300">{t.amount}</td>
                        <td className="p-4 font-bold uppercase">{t.payment_method}</td>
                        <td className="p-4">
                          <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                            t.status === 'PAID' ? 'bg-green-950 text-green-400' : 'bg-yellow-950/60 text-yellow-400'
                          }`}>
                            {t.status}
                          </span>
                        </td>
                        <td className="p-4">{new Date(t.created_at).toLocaleString()}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}

          {/* 4. Webhooks Tab */}
          {activeTab === 'webhooks' && (
            <div className="overflow-x-auto rounded-2xl border border-slate-800 bg-slate-900">
              <table className="w-full border-collapse text-left text-xs">
                <thead>
                  <tr className="border-b border-slate-800 bg-slate-950 font-bold text-slate-300">
                    <th className="p-4">Gateway</th>
                    <th className="p-4">Tipo do Evento</th>
                    <th className="p-4">ID Evento Externo</th>
                    <th className="p-4">Status Processo</th>
                    <th className="p-4">Data Recebido</th>
                    <th className="p-4">Erro</th>
                    <th className="p-4 text-center">Payload</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800 text-slate-400">
                  {webhooks.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="p-4 text-center text-slate-500">Nenhum evento de webhook recebido ainda.</td>
                    </tr>
                  ) : (
                    webhooks.map((w) => (
                      <tr key={w.id} className="hover:bg-slate-950/40">
                        <td className="p-4 font-bold uppercase text-[10px]">{w.provider}</td>
                        <td className="p-4 font-bold text-slate-300">{w.event_type}</td>
                        <td className="p-4 font-mono text-[10px]">{w.external_id}</td>
                        <td className="p-4">
                          <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                            w.status === 'PROCESSED' ? 'bg-green-950 text-green-400' :
                            w.status === 'PENDING' ? 'bg-yellow-950/65 text-yellow-400' : 'bg-red-950 text-red-400'
                          }`}>
                            {w.status}
                          </span>
                        </td>
                        <td className="p-4">{new Date(w.created_at).toLocaleString()}</td>
                        <td className="p-4 text-red-400 max-w-[200px] truncate">{w.error || '-'}</td>
                        <td className="p-4 text-center">
                          <button
                            onClick={() => handleInspectPayload(w.id)}
                            className="px-2.5 py-1 bg-slate-800 hover:bg-slate-700 text-purple-400 font-bold rounded cursor-pointer text-[10px]"
                          >
                            Ver JSON
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* Edit Provider Modal */}
      {isEditModalOpen && editingProvider && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-lg bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
            <div className="p-6 border-b border-slate-800">
              <h2 className="text-base font-bold text-slate-100">Configurar Credenciais: {editingProvider.name}</h2>
            </div>
            <form onSubmit={handleSaveProvider} className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-400 mb-1">API Key / Token Secreto</label>
                <input
                  type="password"
                  value={providerForm.apiKey}
                  onChange={(e) => setProviderForm({ ...providerForm, apiKey: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-400 mb-1">Webhook Secret / Signature Token</label>
                <input
                  type="password"
                  value={providerForm.webhookSecret}
                  onChange={(e) => setProviderForm({ ...providerForm, webhookSecret: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-400 mb-1">Ambiente Operacional</label>
                <select
                  value={providerForm.mode}
                  onChange={(e) => setProviderForm({ ...providerForm, mode: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                >
                  <option value="sandbox">Sandbox (Teste)</option>
                  <option value="production">Production (Real)</option>
                </select>
              </div>

              <div className="flex items-center gap-2 pt-2">
                <input
                  type="checkbox"
                  id="isActiveCheck"
                  checked={providerForm.isActive}
                  onChange={(e) => setProviderForm({ ...providerForm, isActive: e.target.checked })}
                  className="rounded text-purple-600 focus:ring-0 focus:ring-offset-0 bg-slate-950 border-slate-800"
                />
                <label htmlFor="isActiveCheck" className="text-xs font-semibold text-slate-300 cursor-pointer">
                  Provedor ativo para cobranças automáticas
                </label>
              </div>

              <div className="flex gap-3 justify-end pt-4 border-t border-slate-800">
                <button
                  type="button"
                  onClick={() => setIsEditModalOpen(false)}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-bold rounded-xl transition-all cursor-pointer"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 text-xs font-bold rounded-xl transition-all cursor-pointer"
                >
                  Salvar Configuração
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* JSON Viewer Modal */}
      {viewingPayload !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden flex flex-col max-h-[80vh]">
            <div className="p-6 border-b border-slate-800 flex justify-between items-center">
              <h2 className="text-sm font-bold text-slate-100">Webhook Raw Metadata Payload</h2>
              <button
                onClick={() => setViewingPayload(null)}
                className="text-slate-400 hover:text-slate-200 text-xs cursor-pointer font-bold"
              >
                Fechar
              </button>
            </div>
            <pre className="p-6 overflow-auto text-[11px] font-mono text-purple-300 bg-slate-950 flex-1 leading-relaxed">
              {viewingPayload}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
