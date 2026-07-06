import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface AIProvider {
  id: string;
  name: string;
  slug: string;
  apiKey: string;
  baseUrl: string;
  defaultModel: string;
  textModel: string;
  visionModel: string;
  audioModel: string;
  embeddingModel: string;
  embeddingDimensions: number;
  isActive: boolean;
  priority: number;
  isDefault: boolean;
  limitPerMinute: number;
  limitPerDay: number;
  costPerCredit: number;
}

export default function ManageAiProviders() {
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [testingId, setTestingId] = useState<string | null>(null);

  // Form State
  const [editingProvider, setEditingProvider] = useState<AIProvider | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    slug: 'openai',
    apiKey: '',
    baseUrl: '',
    defaultModel: 'gpt-4o',
    textModel: 'gpt-4o',
    visionModel: 'gpt-4o-mini',
    audioModel: '',
    embeddingModel: 'text-embedding-3-small',
    embeddingDimensions: 1536,
    isActive: true,
    priority: 10,
    isDefault: false,
    limitPerMinute: 60,
    limitPerDay: 5000,
    costPerCredit: 0.05,
  });

  useEffect(() => {
    fetchProviders();
  }, []);

  const fetchProviders = async () => {
    setLoading(true);
    try {
      const res = await api.get('/admin/ai-providers');
      setProviders(res.data.data || []);
    } catch (err: any) {
      setError('Erro ao listar provedores de IA.');
    } finally {
      setLoading(false);
    }
  };

  const handleOpenCreate = () => {
    setEditingProvider(null);
    setFormData({
      name: '',
      slug: 'openai',
      apiKey: '',
      baseUrl: 'https://api.openai.com/v1',
      defaultModel: 'gpt-4o',
      textModel: 'gpt-4o',
      visionModel: 'gpt-4o-mini',
      audioModel: '',
      embeddingModel: 'text-embedding-3-small',
      embeddingDimensions: 1536,
      isActive: true,
      priority: 10,
      isDefault: false,
      limitPerMinute: 60,
      limitPerDay: 5000,
      costPerCredit: 0.05,
    });
    setIsModalOpen(true);
  };

  const handleOpenEdit = (p: AIProvider) => {
    setEditingProvider(p);
    setFormData({
      name: p.name,
      slug: p.slug,
      apiKey: p.apiKey || '********',
      baseUrl: p.baseUrl || '',
      defaultModel: p.defaultModel,
      textModel: p.textModel || '',
      visionModel: p.visionModel || '',
      audioModel: p.audioModel || '',
      embeddingModel: p.embeddingModel || '',
      embeddingDimensions: p.embeddingDimensions || 0,
      isActive: p.isActive,
      priority: p.priority,
      isDefault: p.isDefault,
      limitPerMinute: p.limitPerMinute || 0,
      limitPerDay: p.limitPerDay || 0,
      costPerCredit: p.costPerCredit || 0,
    });
    setIsModalOpen(true);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      if (editingProvider) {
        await api.patch(`/admin/ai-providers/${editingProvider.id}`, formData);
        setSuccess('Provedor de IA atualizado com sucesso.');
      } else {
        await api.post('/admin/ai-providers', formData);
        setSuccess('Provedor de IA cadastrado com sucesso.');
      }
      setIsModalOpen(false);
      fetchProviders();
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Erro ao salvar provedor de IA.');
      }
    }
  };

  const handleTestConnection = async (id: string) => {
    setTestingId(id);
    setError('');
    setSuccess('');
    try {
      const res = await api.post(`/admin/ai-providers/${id}/test`);
      if (res.data.data.ok) {
        setSuccess(`Conexão bem sucedida: ${res.data.data.message}`);
      } else {
        setError(`Falha de conexão: ${res.data.data.message}`);
      }
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Falha de conexão técnica com o provedor.');
      }
    } finally {
      setTestingId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-slate-100">Provedores de Inteligência Artificial</h1>
          <p className="text-slate-400 text-xs mt-1">Configure chaves de API criptografadas e prioridades de LLM do MapaTurbo IA.</p>
        </div>
        <button
          onClick={handleOpenCreate}
          className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 font-bold text-xs rounded-xl transition-all cursor-pointer"
        >
          + Novo Provedor
        </button>
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

      {loading ? (
        <p className="text-slate-400 text-xs">Carregando provedores...</p>
      ) : (
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          {providers.map((p) => (
            <div
              key={p.id}
              className={`p-6 rounded-2xl bg-slate-900 border ${
                p.isDefault ? 'border-purple-600' : 'border-slate-800'
              } flex flex-col justify-between space-y-4`}
            >
              <div>
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="font-bold text-slate-100 flex items-center gap-2">
                      {p.name}
                      {p.isDefault && (
                        <span className="px-1.5 py-0.5 text-[9px] bg-purple-950 text-purple-400 rounded font-bold uppercase">
                          Padrão
                        </span>
                      )}
                    </h3>
                    <p className="text-[10px] text-slate-500 font-mono mt-0.5">{p.slug}</p>
                  </div>
                  <span
                    className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                      p.isActive ? 'bg-green-950 text-green-400' : 'bg-slate-800 text-slate-400'
                    }`}
                  >
                    {p.isActive ? 'Ativo' : 'Inativo'}
                  </span>
                </div>

                <div className="mt-4 space-y-2 text-xs text-slate-400">
                  <p>
                    <strong className="text-slate-300">Modelo Principal:</strong> {p.defaultModel}
                  </p>
                  <p>
                    <strong className="text-slate-300">Embedding:</strong> {p.embeddingModel || 'N/A'} ({p.embeddingDimensions}d)
                  </p>
                  <p>
                    <strong className="text-slate-300">Custo por Crédito:</strong> {p.costPerCredit} CRD
                  </p>
                  <p>
                    <strong className="text-slate-300">Prioridade:</strong> {p.priority}
                  </p>
                  <p>
                    <strong className="text-slate-300">API Key:</strong> <code className="text-purple-400">{p.apiKey}</code>
                  </p>
                </div>
              </div>

              <div className="flex gap-2 pt-4 border-t border-slate-800/60">
                <button
                  onClick={() => handleOpenEdit(p)}
                  className="flex-1 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-bold rounded-lg transition-all cursor-pointer text-center"
                >
                  Editar
                </button>
                <button
                  onClick={() => handleTestConnection(p.id)}
                  disabled={testingId !== null}
                  className="flex-1 py-1.5 bg-purple-900/40 hover:bg-purple-900/60 text-purple-300 text-xs font-bold rounded-lg transition-all cursor-pointer disabled:opacity-50 text-center"
                >
                  {testingId === p.id ? 'Testando...' : 'Testar Conexão'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modal Criar/Editar */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden flex flex-col max-h-[90vh]">
            <div className="p-6 border-b border-slate-800">
              <h2 className="text-lg font-bold text-slate-100">
                {editingProvider ? 'Editar Provedor de IA' : 'Novo Provedor de IA'}
              </h2>
            </div>

            <form onSubmit={handleSave} className="p-6 space-y-4 overflow-y-auto flex-1">
              <div className="grid sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Nome Comercial</label>
                  <input
                    type="text"
                    required
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                    placeholder="Ex: OpenAI Prod"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Slug identificador</label>
                  <select
                    disabled={!!editingProvider}
                    value={formData.slug}
                    onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600 disabled:opacity-50"
                  >
                    <option value="openai">openai</option>
                    <option value="gemini">gemini</option>
                    <option value="grok">grok</option>
                    <option value="anthropic">anthropic</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-400 mb-1">Chave de API Secreta</label>
                <input
                  type="password"
                  value={formData.apiKey}
                  onChange={(e) => setFormData({ ...formData, apiKey: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  placeholder="Insira a chave secreta"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-400 mb-1">URL Base API (Opcional)</label>
                <input
                  type="text"
                  value={formData.baseUrl}
                  onChange={(e) => setFormData({ ...formData, baseUrl: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  placeholder="https://api.openai.com/v1"
                />
              </div>

              <div className="grid sm:grid-cols-3 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Modelo Principal</label>
                  <input
                    type="text"
                    required
                    value={formData.defaultModel}
                    onChange={(e) => setFormData({ ...formData, defaultModel: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Modelo de Texto</label>
                  <input
                    type="text"
                    value={formData.textModel}
                    onChange={(e) => setFormData({ ...formData, textModel: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Modelo Visão</label>
                  <input
                    type="text"
                    value={formData.visionModel}
                    onChange={(e) => setFormData({ ...formData, visionModel: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
              </div>

              <div className="grid sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Modelo Embedding</label>
                  <input
                    type="text"
                    value={formData.embeddingModel}
                    onChange={(e) => setFormData({ ...formData, embeddingModel: e.target.value })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Dimensões Embedding</label>
                  <input
                    type="number"
                    value={formData.embeddingDimensions}
                    onChange={(e) => setFormData({ ...formData, embeddingDimensions: parseInt(e.target.value) || 0 })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
              </div>

              <div className="grid sm:grid-cols-3 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Prioridade</label>
                  <input
                    type="number"
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 0 })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Custo por Crédito (R$)</label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.costPerCredit}
                    onChange={(e) => setFormData({ ...formData, costPerCredit: parseFloat(e.target.value) || 0 })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">Limite Minuto (Requisições)</label>
                  <input
                    type="number"
                    value={formData.limitPerMinute}
                    onChange={(e) => setFormData({ ...formData, limitPerMinute: parseInt(e.target.value) || 0 })}
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
              </div>

              <div className="flex gap-6 items-center pt-2">
                <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.isActive}
                    onChange={(e) => setFormData({ ...formData, isActive: e.target.checked })}
                    className="rounded text-purple-600 focus:ring-0 focus:ring-offset-0 bg-slate-950 border-slate-800"
                  />
                  Ativo no Sistema
                </label>

                <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.isDefault}
                    onChange={(e) => setFormData({ ...formData, isDefault: e.target.checked })}
                    className="rounded text-purple-600 focus:ring-0 focus:ring-offset-0 bg-slate-950 border-slate-800"
                  />
                  Definir como Padrão Global
                </label>
              </div>

              <div className="flex gap-3 justify-end pt-4 border-t border-slate-800">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-bold rounded-xl transition-all cursor-pointer"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 text-xs font-bold rounded-xl transition-all cursor-pointer"
                >
                  Salvar Provedor
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
