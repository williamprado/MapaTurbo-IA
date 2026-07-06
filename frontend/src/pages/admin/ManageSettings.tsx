import { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface SystemSetting {
  key: string;
  value: any;
  description: string;
  is_public: boolean;
}

interface AiActionPrice {
  id: string;
  action_key: string;
  name: string;
  description: string;
  credits_cost: number;
  is_active: boolean;
}

export default function ManageSettings() {
  const [activeTab, setActiveTab] = useState<'system' | 'ai'>('system');
  const [settings, setSettings] = useState<SystemSetting[]>([]);
  const [aiPrices, setAiPrices] = useState<AiActionPrice[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    setError('');
    try {
      const [settingsRes, aiPricesRes] = await Promise.all([
        api.get('/admin/settings'),
        api.get('/admin/ai-action-prices'),
      ]);
      setSettings(settingsRes.data.data || []);
      setAiPrices(aiPricesRes.data.data || []);
    } catch (err) {
      setError('Erro ao carregar configurações. Verifique os logs.');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateSystemSetting = async (key: string, value: any) => {
    setSaving(key);
    setError('');
    setSuccess('');
    try {
      await api.patch(`/admin/settings/${key}`, { value });
      setSuccess(`Configuração "${key}" salva com sucesso!`);
      // Update local state
      setSettings((prev) =>
        prev.map((s) => (s.key === key ? { ...s, value } : s))
      );
    } catch (err) {
      setError(`Erro ao salvar configuração "${key}".`);
    } finally {
      setSaving(null);
    }
  };

  const handleUpdateAiPrice = async (price: AiActionPrice, newCost: number, isActive: boolean) => {
    setSaving(price.id);
    setError('');
    setSuccess('');
    try {
      await api.patch(`/admin/ai-action-prices/${price.id}`, {
        credits_cost: Number(newCost),
        is_active: isActive,
      });
      setSuccess(`Preço de "${price.name}" atualizado!`);
      setAiPrices((prev) =>
        prev.map((p) =>
          p.id === price.id ? { ...p, credits_cost: newCost, is_active: isActive } : p
        )
      );
    } catch (err) {
      setError(`Erro ao atualizar preço de "${price.name}".`);
    } finally {
      setSaving(null);
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

      {/* Tabs Selector */}
      <div className="flex border-b border-slate-800">
        <button
          onClick={() => setActiveTab('system')}
          className={`px-6 py-3 font-semibold text-sm transition-all border-b-2 cursor-pointer ${
            activeTab === 'system'
              ? 'border-red-500 text-white'
              : 'border-transparent text-slate-400 hover:text-white'
          }`}
        >
          ⚙️ Parâmetros do Sistema
        </button>
        <button
          onClick={() => setActiveTab('ai')}
          className={`px-6 py-3 font-semibold text-sm transition-all border-b-2 cursor-pointer ${
            activeTab === 'ai'
              ? 'border-red-500 text-white'
              : 'border-transparent text-slate-400 hover:text-white'
          }`}
        >
          🧠 Preços de Ações de IA
        </button>
      </div>

      {/* System Settings Tab */}
      {activeTab === 'system' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {settings.map((setting) => {
            const isSaving = saving === setting.key;
            return (
              <div
                key={setting.key}
                className="bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4 hover:border-slate-750 transition-all"
              >
                <div>
                  <h3 className="text-sm font-bold text-white uppercase tracking-wider">
                    {setting.key.replace(/_/g, ' ')}
                  </h3>
                  <p className="text-xs text-slate-400 mt-1">{setting.description || 'Nenhuma descrição fornecida.'}</p>
                </div>

                <div className="flex items-center gap-4">
                  {typeof setting.value === 'boolean' ? (
                    <div className="flex items-center gap-3">
                      <button
                        onClick={() => handleUpdateSystemSetting(setting.key, !setting.value)}
                        disabled={isSaving}
                        className={`px-4 py-2 rounded-xl text-xs font-bold transition-all cursor-pointer ${
                          setting.value
                            ? 'bg-green-600 hover:bg-green-500 text-white'
                            : 'bg-slate-800 hover:bg-slate-700 text-slate-300'
                        }`}
                      >
                        {setting.value ? 'ATIVADO' : 'DESATIVADO'}
                      </button>
                    </div>
                  ) : typeof setting.value === 'number' ? (
                    <input
                      type="number"
                      defaultValue={setting.value}
                      onBlur={(e) => {
                        const val = Number(e.target.value);
                        if (val !== setting.value) {
                          handleUpdateSystemSetting(setting.key, val);
                        }
                      }}
                      disabled={isSaving}
                      className="bg-slate-950 border border-slate-800 focus:border-red-600 rounded-xl px-4 py-2 text-sm focus:outline-none transition-all w-32"
                    />
                  ) : (
                    <input
                      type="text"
                      defaultValue={String(setting.value)}
                      onBlur={(e) => {
                        const val = e.target.value;
                        if (val !== String(setting.value)) {
                          handleUpdateSystemSetting(setting.key, val);
                        }
                      }}
                      disabled={isSaving}
                      className="bg-slate-950 border border-slate-800 focus:border-red-600 rounded-xl px-4 py-2 text-sm focus:outline-none transition-all w-full"
                    />
                  )}

                  {isSaving && (
                    <span className="text-[10px] text-red-400 animate-pulse font-semibold">
                      Salvando...
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* AI Action Prices Tab */}
      {activeTab === 'ai' && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-950 text-slate-400 text-xs font-semibold uppercase border-b border-slate-800">
                  <th className="py-4 px-6">Ação de IA</th>
                  <th className="py-4 px-6">Identificador (Chave)</th>
                  <th className="py-4 px-6 text-center">Custo (Créditos)</th>
                  <th className="py-4 px-6 text-center">Status</th>
                  <th className="py-4 px-6 text-right">Ação</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50 text-sm">
                {aiPrices.map((price) => {
                  const isSaving = saving === price.id;
                  return (
                    <tr key={price.id} className="hover:bg-slate-850/40 transition-colors">
                      <td className="py-4 px-6">
                        <p className="font-bold text-white">{price.name}</p>
                        <p className="text-xs text-slate-400 mt-0.5">{price.description}</p>
                      </td>
                      <td className="py-4 px-6 font-mono text-xs text-slate-350">{price.action_key}</td>
                      <td className="py-4 px-6 text-center">
                        <input
                          id={`cost-${price.id}`}
                          type="number"
                          defaultValue={price.credits_cost}
                          disabled={isSaving}
                          className="bg-slate-950 border border-slate-800 focus:border-red-600 rounded-lg px-2 py-1 text-xs text-center focus:outline-none transition-all w-20"
                        />
                      </td>
                      <td className="py-4 px-6 text-center">
                        <select
                          id={`status-${price.id}`}
                          defaultValue={price.is_active ? 'true' : 'false'}
                          disabled={isSaving}
                          className="bg-slate-950 border border-slate-800 rounded-lg px-2 py-1 text-xs focus:outline-none text-slate-300"
                        >
                          <option value="true">Ativo</option>
                          <option value="false">Inativo</option>
                        </select>
                      </td>
                      <td className="py-4 px-6 text-right">
                        <button
                          onClick={() => {
                            const costInput = document.getElementById(`cost-${price.id}`) as HTMLInputElement;
                            const statusSelect = document.getElementById(`status-${price.id}`) as HTMLSelectElement;
                            handleUpdateAiPrice(
                              price,
                              Number(costInput.value),
                              statusSelect.value === 'true'
                            );
                          }}
                          disabled={isSaving}
                          className="bg-red-700 hover:bg-red-600 disabled:bg-red-900 text-white font-bold py-1 px-3 rounded-lg text-xs transition-colors cursor-pointer"
                        >
                          {isSaving ? 'Salvando...' : 'Salvar'}
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
