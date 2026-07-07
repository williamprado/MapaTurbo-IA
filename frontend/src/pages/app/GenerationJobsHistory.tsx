import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../../services/api';

interface GenJob {
  id: string;
  type: string;
  status: string; // PENDING, PROCESSING, COMPLETED, FAILED
  credits_cost: number;
  mind_map_id?: string;
  error?: string;
  created_at: string;
  finished_at?: string;
}

export default function GenerationJobsHistory() {
  const navigate = useNavigate();
  const [jobs, setJobs] = useState<GenJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [limit] = useState(15);
  const [filterStatus, setFilterStatus] = useState('ALL'); // ALL, PENDING, PROCESSING, COMPLETED, FAILED
  const [errorMsg, setErrorMsg] = useState('');

  useEffect(() => {
    fetchJobsHistory();
  }, [page, filterStatus]);

  const fetchJobsHistory = async () => {
    setLoading(true);
    setErrorMsg('');
    try {
      const statusQuery = filterStatus === 'ALL' ? '' : filterStatus;
      const res = await api.get(`/generation-jobs?page=${page}&limit=${limit}&status=${statusQuery}`);
      const data = res.data.data;
      setJobs(data.items || []);
      setTotal(data.pagination?.total || 0);
    } catch (err) {
      console.error('Erro ao carregar histórico de jobs:', err);
      setErrorMsg('Não foi possível carregar o histórico de processamento da IA.');
    } finally {
      setLoading(false);
    }
  };

  const handlePrevPage = () => {
    if (page > 1) setPage(page - 1);
  };

  const handleNextPage = () => {
    if (page * limit < total) setPage(page + 1);
  };

  const formatJobType = (type: string) => {
    if (type.includes('TOPIC')) return 'Geração por Tema';
    if (type.includes('TEXT')) return 'Geração por Texto';
    if (type.includes('PDF') || type.includes('UPLOAD')) return 'Geração por PDF';
    return type;
  };

  return (
    <div className="space-y-8">
      {errorMsg && (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
          ⚠️ {errorMsg}
        </div>
      )}

      {/* Header and Filter Controls */}
      <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6 border-b border-slate-850 pb-6">
          <div>
            <h3 className="font-bold text-slate-100 text-lg">Histórico de Geração IA</h3>
            <p className="text-xs text-slate-500">Acompanhe a fila de processamento assíncrono e jobs de IA executados no Worker.</p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {['ALL', 'COMPLETED', 'FAILED', 'PROCESSING', 'PENDING'].map((status) => (
              <button
                key={status}
                onClick={() => { setFilterStatus(status); setPage(1); }}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  filterStatus === status
                    ? 'bg-purple-600 text-white shadow-lg shadow-purple-500/15'
                    : 'bg-slate-800 text-slate-400 hover:text-slate-200'
                }`}
              >
                {status === 'ALL' ? 'Todos' : status}
              </button>
            ))}
          </div>
        </div>

        {/* Jobs List */}
        {loading ? (
          <div className="flex justify-center items-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-purple-500"></div>
          </div>
        ) : jobs.length === 0 ? (
          <div className="py-16 text-center text-slate-500 flex flex-col items-center justify-center">
            <span className="text-4xl mb-2">🤖</span>
            <p className="text-xs max-w-[280px]">Nenhum job de geração encontrado.</p>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="border-b border-slate-800 text-slate-500 font-bold uppercase tracking-wider">
                    <th className="pb-3">Tipo do Job</th>
                    <th className="pb-3">Status</th>
                    <th className="pb-3 text-right">Créditos</th>
                    <th className="pb-3 text-right">Data de Envio</th>
                    <th className="pb-3 text-right">Ação</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-850">
                  {jobs.map((job) => (
                    <tr key={job.id} className="text-slate-300">
                      <td className="py-3.5">
                        <div>
                          <p className="font-bold text-slate-200">{formatJobType(job.type)}</p>
                          <p className="text-[10px] text-slate-500 font-mono">ID: {job.id}</p>
                        </div>
                      </td>
                      <td className="py-3.5">
                        <div className="flex flex-col">
                          <span
                            className={`px-2 py-0.5 rounded font-bold text-[9px] uppercase w-fit ${
                              job.status === 'COMPLETED'
                                ? 'bg-green-950 text-green-400'
                                : job.status === 'FAILED'
                                ? 'bg-red-950 text-red-400'
                                : 'bg-yellow-950 text-yellow-400'
                            }`}
                          >
                            {job.status}
                          </span>
                          {job.error && (
                            <span className="text-[10px] text-red-400 mt-1 max-w-[250px] truncate" title={job.error}>
                              ⚠️ {job.error}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="py-3.5 text-right font-mono font-bold text-purple-400">
                        {job.credits_cost} CRD
                      </td>
                      <td className="py-3.5 text-right text-slate-500">
                        {new Date(job.created_at).toLocaleString('pt-BR')}
                      </td>
                      <td className="py-3.5 text-right">
                        {job.status === 'COMPLETED' && job.mind_map_id ? (
                          <button
                            onClick={() => navigate(`/app/maps/${job.mind_map_id}/editor`)}
                            className="px-2.5 py-1 bg-purple-600 hover:bg-purple-500 text-white text-[10px] font-bold rounded transition-colors"
                          >
                            Ver Mapa Concept
                          </button>
                        ) : (
                          <span className="text-slate-600">-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination controls */}
            {total > limit && (
              <div className="flex justify-between items-center pt-4 border-t border-slate-850 text-xs">
                <span className="text-slate-500">
                  Exibindo {jobs.length} de {total} jobs (Pág. {page})
                </span>
                <div className="flex items-center gap-2">
                  <button
                    disabled={page === 1}
                    onClick={handlePrevPage}
                    className="px-3 py-1.5 bg-slate-800 hover:bg-slate-750 text-slate-200 font-medium rounded-lg disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                  >
                    Anterior
                  </button>
                  <button
                    disabled={page * limit >= total}
                    onClick={handleNextPage}
                    className="px-3 py-1.5 bg-slate-800 hover:bg-slate-750 text-slate-200 font-medium rounded-lg disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                  >
                    Próxima
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
