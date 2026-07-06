import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';
import { useAuthStore } from '../../stores/auth';

interface Plan {
  id: string;
  name: string;
  description: string;
  price_monthly: string;
  price_yearly: string;
  currency: string;
  credits_monthly: number;
  max_maps: number;
  max_files: number;
  features: Record<string, any>;
}

interface Invoice {
  id: string;
  amount: string;
  currency: string;
  status: string;
  due_date: string;
  billing_type: string;
  invoice_url: string;
  pdf_url: string;
  pix_qr_code: string;
  pix_copy_paste: string;
  paid_at: string;
  created_at: string;
}

export default function Billing() {
  const { activeOrgId } = useAuthStore();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // Checkout modal
  const [selectedPlan, setSelectedPlan] = useState<Plan | null>(null);
  const [checkoutForm, setCheckoutForm] = useState({
    cycle: 'monthly',
    billingType: 'PIX',
    document: '',
    phone: '',
  });
  const [checkingOut, setCheckingOut] = useState(false);

  // Active Checkout Result Modal
  const [checkoutResult, setCheckoutResult] = useState<any>(null);

  useEffect(() => {
    fetchPlansAndInvoices();
  }, [activeOrgId]);

  const fetchPlansAndInvoices = async () => {
    setLoading(true);
    setError('');
    try {
      const [plansRes, invoicesRes] = await Promise.all([
        api.get('/plans/public'),
        api.get('/billing/invoices'),
      ]);
      setPlans(plansRes.data.data || []);
      setInvoices(invoicesRes.data.data || []);
    } catch (err: any) {
      console.error('Erro ao carregar dados do financeiro:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenCheckout = (plan: Plan) => {
    setSelectedPlan(plan);
    setCheckoutForm({
      cycle: 'monthly',
      billingType: 'PIX',
      document: '',
      phone: '',
    });
    setCheckoutResult(null);
  };

  const handleConfirmCheckout = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedPlan) return;
    setCheckingOut(true);
    setError('');
    setSuccess('');

    try {
      const res = await api.post('/billing/checkout', {
        planId: selectedPlan.id,
        cycle: checkoutForm.cycle,
        billingType: checkoutForm.billingType,
        document: checkoutForm.document,
        phone: checkoutForm.phone,
      });

      setSuccess('Fatura gerada com sucesso! Conclua o pagamento abaixo.');
      setCheckoutResult(res.data.data);
      setSelectedPlan(null); // Fecha modal de formulário
      fetchPlansAndInvoices(); // Atualiza histórico
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Erro ao iniciar sessão de pagamento. Verifique os dados.');
      }
    } finally {
      setCheckingOut(false);
    }
  };

  const handleCopyPix = () => {
    if (checkoutResult?.invoice?.pix_copy_paste) {
      navigator.clipboard.writeText(checkoutResult.invoice.pix_copy_paste);
      alert('Código Copia e Cola copiado para a área de transferência!');
    }
  };

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-slate-100">Assinaturas & Planos</h1>
        <p className="text-slate-400 text-xs mt-1">Gerencie os limites de IA do seu workspace e consulte faturas emitidas.</p>
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

      {/* Render Active Checkout Pix/Boleto Details */}
      {checkoutResult && (
        <div className="p-6 rounded-2xl bg-purple-950/20 border border-purple-500/35 space-y-6">
          <div className="flex justify-between items-center border-b border-purple-500/20 pb-4">
            <div>
              <h3 className="text-base font-bold text-slate-100">Aguardando Pagamento</h3>
              <p className="text-xs text-slate-400">Pague no gateway Asaas para liberar seu plano instantaneamente.</p>
            </div>
            <button
              onClick={() => setCheckoutResult(null)}
              className="text-xs text-slate-400 hover:text-slate-200 font-semibold cursor-pointer"
            >
              Fechar Detalhes
            </button>
          </div>

          <div className="grid md:grid-cols-2 gap-6 items-center">
            {checkoutResult.invoice?.billing_type === 'PIX' && (
              <>
                <div className="flex flex-col items-center justify-center p-4 bg-slate-950 rounded-xl border border-slate-800">
                  {checkoutResult.invoice?.pix_qr_code ? (
                    <img
                      src={`data:image/png;base64,${checkoutResult.invoice.pix_qr_code}`}
                      alt="Pix QR Code"
                      className="w-48 h-48 rounded"
                    />
                  ) : (
                    <div className="w-48 h-48 bg-slate-900 animate-pulse rounded" />
                  )}
                  <p className="text-[10px] text-slate-500 mt-2">Escaneie o QR Code no app do seu banco</p>
                </div>
                <div className="space-y-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-400 mb-1">Pix Copia e Cola</label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        readOnly
                        value={checkoutResult.invoice?.pix_copy_paste || ''}
                        className="flex-1 px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-[10px] font-mono text-slate-300 focus:outline-none"
                      />
                      <button
                        onClick={handleCopyPix}
                        className="px-3 py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 font-bold text-xs rounded-lg cursor-pointer"
                      >
                        Copiar
                      </button>
                    </div>
                  </div>
                  <a
                    href={checkoutResult.url}
                    target="_blank"
                    rel="noreferrer"
                    className="block w-full py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 font-bold text-xs rounded-xl cursor-pointer text-center"
                  >
                    Pagar no Asaas
                  </a>
                </div>
              </>
            )}

            {checkoutResult.invoice?.billing_type === 'BOLETO' && (
              <div className="space-y-4 col-span-2">
                <p className="text-xs text-slate-355">Fatura Boleto gerada com sucesso. Clique abaixo para fazer o download.</p>
                <div className="flex gap-4">
                  {checkoutResult.invoice?.pdf_url && (
                    <a
                      href={checkoutResult.invoice.pdf_url}
                      target="_blank"
                      rel="noreferrer"
                      className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 font-bold text-xs rounded-lg cursor-pointer"
                    >
                      Download Boleto (PDF)
                    </a>
                  )}
                  <a
                    href={checkoutResult.url}
                    target="_blank"
                    rel="noreferrer"
                    className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 font-bold text-xs rounded-lg cursor-pointer"
                  >
                    Ir para o Checkout Asaas
                  </a>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Available Plans Grid */}
      <div>
        <h2 className="text-lg font-bold mb-4">Escolha seu Plano</h2>
        {loading ? (
          <p className="text-slate-400 text-xs">Carregando planos...</p>
        ) : (
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {plans.map((plan) => (
              <div key={plan.id} className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col justify-between space-y-6">
                <div>
                  <h3 className="text-base font-bold text-slate-100">{plan.name}</h3>
                  <p className="text-xs text-slate-400 mt-2 min-h-[32px]">{plan.description}</p>
                  
                  <div className="mt-4">
                    <span className="text-2xl font-black text-slate-100">R$ {plan.price_monthly}</span>
                    <span className="text-xs text-slate-500 font-medium">/mês</span>
                  </div>

                  <ul className="mt-6 space-y-2 text-xs text-slate-400">
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✔</span> {plan.credits_monthly} créditos de IA / mês
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✔</span> Limite de {plan.max_maps} mapas mentais
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✔</span> Armazenamento de {plan.max_files} PDFs
                    </li>
                  </ul>
                </div>

                <button
                  onClick={() => handleOpenCheckout(plan)}
                  className="w-full py-2.5 bg-purple-600 hover:bg-purple-700 text-slate-100 font-bold text-xs rounded-xl transition-all cursor-pointer text-center"
                >
                  Contratar Plano
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Invoice History */}
      <div>
        <h2 className="text-lg font-bold mb-4">Histórico de Cobranças</h2>
        <div className="overflow-x-auto rounded-2xl border border-slate-800 bg-slate-900">
          <table className="w-full border-collapse text-left text-xs">
            <thead>
              <tr className="border-b border-slate-800 bg-slate-950 font-bold text-slate-300">
                <th className="p-4">Identificador</th>
                <th className="p-4">Valor</th>
                <th className="p-4">Método</th>
                <th className="p-4">Data Emissão</th>
                <th className="p-4">Vencimento</th>
                <th className="p-4">Status</th>
                <th className="p-4 text-center">Fatura</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800 text-slate-400">
              {invoices.length === 0 ? (
                <tr>
                  <td colSpan={7} className="p-4 text-center text-slate-500">Nenhuma fatura emitida para este workspace.</td>
                </tr>
              ) : (
                invoices.map((inv) => (
                  <tr key={inv.id} className="hover:bg-slate-950/45">
                    <td className="p-4 font-mono text-[10px]">{inv.id}</td>
                    <td className="p-4 font-bold text-purple-400">{inv.amount} {inv.currency}</td>
                    <td className="p-4 font-bold uppercase">{inv.billing_type}</td>
                    <td className="p-4">{new Date(inv.created_at).toLocaleDateString()}</td>
                    <td className="p-4">{inv.due_date ? new Date(inv.due_date).toLocaleDateString() : '-'}</td>
                    <td className="p-4">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                        inv.status === 'PAID' ? 'bg-green-950 text-green-400' :
                        inv.status === 'PENDING' ? 'bg-yellow-950/50 text-yellow-400' : 'bg-red-950 text-red-400'
                      }`}>
                        {inv.status}
                      </span>
                    </td>
                    <td className="p-4 text-center">
                      {inv.invoice_url ? (
                        <a
                          href={inv.invoice_url}
                          target="_blank"
                          rel="noreferrer"
                          className="text-xs text-purple-400 hover:underline"
                        >
                          Visualizar
                        </a>
                      ) : (
                        '-'
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Checkout Modal */}
      {selectedPlan && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
            <div className="p-6 border-b border-slate-800">
              <h3 className="text-base font-bold text-slate-100">Checkout: {selectedPlan.name}</h3>
              <p className="text-xs text-slate-400 mt-1">Forneça as informações requeridas pelo gateway de faturamento Asaas.</p>
            </div>
            
            <form onSubmit={handleConfirmCheckout} className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-400 mb-1">CPF ou CNPJ</label>
                <input
                  type="text"
                  required
                  value={checkoutForm.document}
                  onChange={(e) => setCheckoutForm({ ...checkoutForm, document: e.target.value })}
                  placeholder="Apenas números (Ex: 12345678909)"
                  className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-400 mb-1">Telefone Celular</label>
                <input
                  type="text"
                  value={checkoutForm.phone}
                  onChange={(e) => setCheckoutForm({ ...checkoutForm, phone: e.target.value })}
                  placeholder="Ex: 11988888888"
                  className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Ciclo</label>
                  <select
                    value={checkoutForm.cycle}
                    onChange={(e) => setCheckoutForm({ ...checkoutForm, cycle: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  >
                    <option value="monthly">Mensal (R$ {selectedPlan.price_monthly})</option>
                    <option value="yearly">Anual (R$ {selectedPlan.price_yearly})</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Método de Pagamento</label>
                  <select
                    value={checkoutForm.billingType}
                    onChange={(e) => setCheckoutForm({ ...checkoutForm, billingType: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  >
                    <option value="PIX">PIX</option>
                    <option value="BOLETO">Boleto Bancário</option>
                  </select>
                </div>
              </div>

              <div className="flex gap-3 justify-end pt-4 border-t border-slate-800">
                <button
                  type="button"
                  onClick={() => setSelectedPlan(null)}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-bold rounded-xl transition-all cursor-pointer"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={checkingOut}
                  className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 text-xs font-bold rounded-xl transition-all cursor-pointer disabled:opacity-50"
                >
                  {checkingOut ? 'Processando...' : 'Confirmar e Pagar'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
