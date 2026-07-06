import { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface Plan {
  id: string;
  name: string;
  price_monthly: string;
  price_yearly: string;
  currency: string;
  is_public: boolean;
  is_active: boolean;
  created_at: string;
}

export default function ManagePlans() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);

  // Form states
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [priceMonthly, setPriceMonthly] = useState(0.00);
  const [priceYearly, setPriceYearly] = useState(0.00);
  const [creditsMonthly, setCreditsMonthly] = useState(100);
  const [maxMaps, setMaxMaps] = useState(3);
  const [isPublic, setIsPublic] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const fetchPlans = async () => {
    setLoading(true);
    try {
      const res = await api.get('/admin/plans');
      setPlans(res.data.data.plans || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPlans();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      await api.post('/admin/plans', {
        name,
        description,
        price_monthly: priceMonthly,
        price_yearly: priceYearly,
        currency: 'BRL',
        credits_monthly: creditsMonthly,
        max_maps: maxMaps,
        max_files: 5,
        max_users: 1,
        max_storage_bytes: 100 * 1024 * 1024,
        features: {},
        is_public: isPublic,
        is_active: true,
      });

      setSuccess('Plano criado com sucesso!');
      setName('');
      setDescription('');
      setPriceMonthly(0.00);
      setPriceYearly(0.00);
      setCreditsMonthly(100);
      setMaxMaps(3);
      setShowForm(false);
      fetchPlans();
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Erro ao criar plano.');
      }
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-bold text-slate-100 mb-1">Modelos de Planos</h3>
          <p className="text-xs text-slate-400">Configure os limites de uso e preços cobrados dos inquilinos.</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-purple-600 hover:bg-purple-500 text-white font-semibold text-xs px-4 py-2.5 rounded-lg transition-colors cursor-pointer"
        >
          {showForm ? 'Fechar Formulário' : 'Novo Plano'}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-4 max-w-lg">
          <h4 className="font-bold text-sm text-slate-200">Criar Plano Comercial</h4>

          {error && <p className="text-xs text-red-400">{error}</p>}
          {success && <p className="text-xs text-green-400">{success}</p>}

          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs text-slate-400 mb-2 font-medium">Nome do Plano</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 focus:border-purple-600 rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none"
                placeholder="Ex: Turbo Pro"
                required
              />
            </div>

            <div>
              <label className="block text-xs text-slate-400 mb-2 font-medium">Descrição Curta</label>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 focus:border-purple-600 rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none"
                placeholder="Ex: Melhor para estudantes"
              />
            </div>

            <div>
              <label className="block text-xs text-slate-400 mb-2 font-medium">Preço Mensal (R$)</label>
              <input
                type="number"
                step="0.01"
                value={priceMonthly}
                onChange={(e) => setPriceMonthly(parseFloat(e.target.value))}
                className="w-full bg-slate-950 border border-slate-800 focus:border-purple-600 rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none"
                required
              />
            </div>

            <div>
              <label className="block text-xs text-slate-400 mb-2 font-medium">Preço Anual (R$)</label>
              <input
                type="number"
                step="0.01"
                value={priceYearly}
                onChange={(e) => setPriceYearly(parseFloat(e.target.value))}
                className="w-full bg-slate-950 border border-slate-800 focus:border-purple-600 rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none"
                required
              />
            </div>

            <div>
              <label className="block text-xs text-slate-400 mb-2 font-medium">Créditos de IA Mensais</label>
              <input
                type="number"
                value={creditsMonthly}
                onChange={(e) => setCreditsMonthly(parseInt(e.target.value))}
                className="w-full bg-slate-950 border border-slate-800 focus:border-purple-600 rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none"
                required
              />
            </div>

            <div>
              <label className="block text-xs text-slate-400 mb-2 font-medium">Limite Máximo de Mapas</label>
              <input
                type="number"
                value={maxMaps}
                onChange={(e) => setMaxMaps(parseInt(e.target.value))}
                className="w-full bg-slate-950 border border-slate-800 focus:border-purple-600 rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none"
                required
              />
            </div>
          </div>

          <div className="flex items-center gap-2 py-2">
            <input
              type="checkbox"
              id="isPublic"
              checked={isPublic}
              onChange={(e) => setIsPublic(e.target.checked)}
              className="rounded border-slate-800 bg-slate-950 text-purple-600 focus:ring-0 focus:ring-offset-0"
            />
            <label htmlFor="isPublic" className="text-xs text-slate-350 cursor-pointer">
              Disponível Publicamente na Página de Vendas
            </label>
          </div>

          <button
            type="submit"
            className="bg-purple-600 hover:bg-purple-500 text-white font-semibold text-xs px-4 py-2.5 rounded-lg transition-all"
          >
            Salvar Plano
          </button>
        </form>
      )}

      {loading ? (
        <p className="text-xs text-slate-500 py-4">Carregando planos...</p>
      ) : plans.length === 0 ? (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-8 text-center text-slate-500 text-xs">
          Nenhum plano cadastrado.
        </div>
      ) : (
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <table className="w-full border-collapse text-left text-xs text-slate-300">
            <thead className="bg-slate-950 text-slate-400 uppercase font-semibold text-[10px] border-b border-slate-800">
              <tr>
                <th className="p-4">Nome</th>
                <th className="p-4">Preço Mensal</th>
                <th className="p-4">Preço Anual</th>
                <th className="p-4">Visibilidade</th>
                <th className="p-4">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {plans.map((p) => (
                <tr key={p.id} className="hover:bg-slate-850/50 transition-colors">
                  <td className="p-4 font-bold text-slate-200">{p.name}</td>
                  <td className="p-4 text-slate-400">{p.currency} {parseFloat(p.price_monthly).toFixed(2)}</td>
                  <td className="p-4 text-slate-400">{p.currency} {parseFloat(p.price_yearly).toFixed(2)}</td>
                  <td className="p-4">
                    {p.is_public ? (
                      <span className="bg-blue-950 text-blue-400 border border-blue-500/20 text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded">
                        Público
                      </span>
                    ) : (
                      <span className="bg-orange-950 text-orange-400 border border-orange-500/20 text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded">
                        Privado
                      </span>
                    )}
                  </td>
                  <td className="p-4">
                    {p.is_active ? (
                      <span className="bg-green-950 text-green-400 border border-green-500/20 text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded">
                        Ativo
                      </span>
                    ) : (
                      <span className="bg-slate-950 text-slate-400 border border-slate-800 text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded">
                        Inativo
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
