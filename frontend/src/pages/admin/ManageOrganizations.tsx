import { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface Org {
  id: string;
  name: string;
  slug: string;
  status: string;
  created_at: string;
}

export default function ManageOrganizations() {
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [loading, setLoading] = useState(true);

  // Form states
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const fetchOrgs = async () => {
    setLoading(true);
    try {
      const res = await api.get('/admin/organizations');
      setOrgs(res.data.data.organizations || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOrgs();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    try {
      await api.post('/admin/organizations', { name, slug });
      setSuccess('Organização criada com sucesso!');
      setName('');
      setSlug('');
      setShowForm(false);
      fetchOrgs();
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Erro ao criar organização.');
      }
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-lg font-bold text-slate-100 mb-1">Gerenciamento de Empresas</h3>
          <p className="text-xs text-slate-400">Visualize e gerencie todos os inquilinos (tenants) do seu SaaS.</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-purple-600 hover:bg-purple-500 text-white font-semibold text-xs px-4 py-2.5 rounded-lg transition-colors cursor-pointer"
        >
          {showForm ? 'Fechar Formulário' : 'Nova Organização'}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="p-6 bg-slate-900 border border-slate-800 rounded-xl space-y-4 max-w-md">
          <h4 className="font-bold text-sm text-slate-200">Criar Organização Manualmente</h4>
          
          {error && <p className="text-xs text-red-400">{error}</p>}
          {success && <p className="text-xs text-green-400">{success}</p>}

          <div>
            <label className="block text-xs text-slate-400 mb-2 font-medium">Nome da Organização</label>
            <input
              type="text"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                setSlug(e.target.value.toLowerCase().replace(/ /g, '-').replace(/[^\w-]+/g, ''));
              }}
              className="w-full bg-slate-950 border border-slate-800 focus:border-purple-600 rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none"
              placeholder="Ex: ACME Corp"
              required
            />
          </div>

          <div>
            <label className="block text-xs text-slate-400 mb-2 font-medium">Slug de Roteamento</label>
            <input
              type="text"
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 focus:border-purple-600 rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none"
              placeholder="ex-acme-corp"
              required
            />
          </div>

          <button
            type="submit"
            className="bg-purple-600 hover:bg-purple-500 text-white font-semibold text-xs px-4 py-2.5 rounded-lg transition-all"
          >
            Salvar Organização
          </button>
        </form>
      )}

      {loading ? (
        <p className="text-xs text-slate-500 py-4">Carregando organizações...</p>
      ) : orgs.length === 0 ? (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-8 text-center text-slate-500 text-xs">
          Nenhuma organização cadastrada.
        </div>
      ) : (
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <table className="w-full border-collapse text-left text-xs text-slate-300">
            <thead className="bg-slate-950 text-slate-400 uppercase font-semibold text-[10px] border-b border-slate-800">
              <tr>
                <th className="p-4">Nome</th>
                <th className="p-4">Slug</th>
                <th className="p-4">Status</th>
                <th className="p-4">Data de Criação</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {orgs.map((o) => (
                <tr key={o.id} className="hover:bg-slate-850/50 transition-colors">
                  <td className="p-4 font-bold text-slate-200">{o.name}</td>
                  <td className="p-4 text-slate-400">{o.slug}</td>
                  <td className="p-4">
                    <span className="bg-green-950 text-green-400 border border-green-500/20 text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded">
                      {o.status}
                    </span>
                  </td>
                  <td className="p-4 text-slate-500">{new Date(o.created_at).toLocaleDateString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
