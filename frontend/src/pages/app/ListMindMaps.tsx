import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../services/api';

interface MindMap {
  id: string;
  title: string;
  source_type: string;
  status: string;
  created_at: string;
}

export default function ListMindMaps() {
  const [maps, setMaps] = useState<MindMap[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    fetchMaps();
  }, []);

  const fetchMaps = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api.get('/mindmaps');
      setMaps(res.data.data || []);
    } catch (err: any) {
      setError('Erro ao carregar seus mapas mentais.');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string, title: string) => {
    if (!window.confirm(`Tem certeza que deseja excluir o mapa "${title}"?`)) return;
    try {
      await api.delete(`/mindmaps/${id}`);
      setMaps(maps.filter((m) => m.id !== id));
    } catch {
      alert('Falha ao excluir o mapa mental.');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-slate-100">Meus Mapas Mentais</h1>
          <p className="text-slate-400 text-xs mt-1">Crie e visualize mapas conceituais gerados por Inteligência Artificial.</p>
        </div>
        <Link
          to="/app/maps/new"
          className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 font-bold text-xs rounded-xl transition-all cursor-pointer"
        >
          + Novo Mapa
        </Link>
      </div>

      {error && (
        <div className="p-4 bg-red-500/10 border border-red-500/35 rounded-xl text-red-400 text-xs">
          ⚠️ {error}
        </div>
      )}

      {loading ? (
        <p className="text-slate-400 text-xs animate-pulse">Carregando mapas...</p>
      ) : (
        <div className="overflow-x-auto rounded-2xl border border-slate-800 bg-slate-900">
          <table className="w-full border-collapse text-left text-xs">
            <thead>
              <tr className="border-b border-slate-800 bg-slate-950 font-bold text-slate-300">
                <th className="p-4">Título</th>
                <th className="p-4">Origem</th>
                <th className="p-4">Status</th>
                <th className="p-4">Criado em</th>
                <th className="p-4 text-right">Ações</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800 text-slate-400">
              {maps.length === 0 ? (
                <tr>
                  <td colSpan={5} className="p-8 text-center text-slate-500">
                    Você ainda não gerou nenhum mapa mental. Comece criando um novo!
                  </td>
                </tr>
              ) : (
                maps.map((m) => (
                  <tr key={m.id} className="hover:bg-slate-950/40 transition-colors">
                    <td className="p-4 font-bold text-slate-200">
                      <Link to={`/app/maps/${m.id}`} className="hover:text-purple-400 hover:underline">
                        🧠 {m.title}
                      </Link>
                    </td>
                    <td className="p-4">
                      <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-slate-800 text-slate-300 uppercase">
                        {m.source_type}
                      </span>
                    </td>
                    <td className="p-4">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                        m.status === 'READY' ? 'bg-green-950 text-green-400' : 'bg-yellow-950/50 text-yellow-400'
                      }`}>
                        {m.status}
                      </span>
                    </td>
                    <td className="p-4">{new Date(m.created_at).toLocaleString()}</td>
                    <td className="p-4 text-right space-x-2">
                      <Link
                        to={`/app/maps/${m.id}`}
                        className="px-2.5 py-1 bg-purple-950 hover:bg-purple-900 text-purple-450 font-bold rounded text-[10px] transition-all"
                      >
                        Visualizar
                      </Link>
                      <button
                        onClick={() => handleDelete(m.id, m.title)}
                        className="px-2.5 py-1 bg-red-950/30 hover:bg-red-950/70 text-red-400 font-bold rounded text-[10px] transition-all cursor-pointer"
                      >
                        Excluir
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
  );
}
