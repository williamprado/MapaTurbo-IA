import { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface Transaction {
  id: string;
  type: string; // ADD / SUB
  amount: number;
  description: string;
  created_at: string;
}

export default function CreditsHistory() {
  const [balance, setBalance] = useState(0);
  const [txs, setTxs] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [limit] = useState(15);
  const [filterType, setFilterType] = useState('ALL'); // ALL, CREDIT, DEBIT
  const [error, setError] = useState('');

  useEffect(() => {
    fetchCreditsData();
  }, [page, filterType]);

  const fetchCreditsData = async () => {
    setLoading(true);
    setError('');
    try {
      const typeQuery = filterType === 'ALL' ? '' : filterType;
      const res = await api.get(`/credits?page=${page}&limit=${limit}&type=${typeQuery}`);
      const data = res.data.data;
      setBalance(data.balance || 0);
      setTxs(data.items || []);
      setTotal(data.pagination?.total || 0);
    } catch (err) {
      console.error('Erro ao carregar histórico de créditos:', err);
      setError('Não foi possível carregar seu extrato de créditos.');
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

  return (
    <div className="space-y-8">
      {error && (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
          ⚠️ {error}
        </div>
      )}

      {/* Credit Balance Card */}
      <div className="p-8 rounded-2xl bg-gradient-to-tr from-slate-900 to-slate-950 border border-slate-800 flex flex-col md:flex-row items-center justify-between gap-6 relative overflow-hidden">
        <div className="absolute top-1/2 left-0 -translate-y-1/2 w-48 h-48 bg-purple-600/10 rounded-full blur-[80px] pointer-events-none" />
        <div className="relative z-10 text-center md:text-left">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">Saldo Atual de Créditos</p>
          <h2 className="text-4xl md:text-5xl font-black text-white">{balance} <span className="text-purple-400 text-2xl font-bold">CRD</span></h2>
          <p className="text-xs text-slate-400 mt-2">Créditos são consumidos automaticamente na geração de mapas por inteligência artificial.</p>
        </div>
        <div className="flex gap-3 relative z-10">
          <button
            onClick={() => fetchCreditsData()}
            className="px-4 py-2 bg-slate-800 hover:bg-slate-750 text-slate-200 text-xs font-bold rounded-lg transition-colors border border-slate-750"
          >
            🔄 Atualizar Saldo
          </button>
        </div>
      </div>

      {/* Filters and List */}
      <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6 border-b border-slate-850 pb-6">
          <div>
            <h3 className="font-bold text-slate-100">Histórico de Transações</h3>
            <p className="text-xs text-slate-500">Acompanhe todas as entradas e saídas de créditos do workspace.</p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => { setFilterType('ALL'); setPage(1); }}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                filterType === 'ALL' ? 'bg-purple-600 text-white shadow-lg shadow-purple-500/15' : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              }`}
            >
              Todos
            </button>
            <button
              onClick={() => { setFilterType('CREDIT'); setPage(1); }}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                filterType === 'CREDIT' ? 'bg-green-600 text-white shadow-lg shadow-green-500/15' : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              }`}
            >
              Entradas
            </button>
            <button
              onClick={() => { setFilterType('DEBIT'); setPage(1); }}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                filterType === 'DEBIT' ? 'bg-red-650 text-white shadow-lg shadow-red-500/15' : 'bg-slate-800 text-slate-400 hover:text-slate-200'
              }`}
            >
              Saídas
            </button>
          </div>
        </div>

        {/* Transactions Table/List */}
        {loading ? (
          <div className="flex justify-center items-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-purple-500"></div>
          </div>
        ) : txs.length === 0 ? (
          <div className="py-16 text-center text-slate-500 flex flex-col items-center justify-center">
            <span className="text-4xl mb-2">📜</span>
            <p className="text-xs max-w-[280px]">Nenhuma transação registrada neste workspace ainda.</p>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="border-b border-slate-800 text-slate-500 font-bold uppercase tracking-wider">
                    <th className="pb-3">Descrição</th>
                    <th className="pb-3 text-center">Tipo</th>
                    <th className="pb-3 text-right">Quantidade</th>
                    <th className="pb-3 text-right">Data</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-850">
                  {txs.map((tx) => (
                    <tr key={tx.id} className="text-slate-300">
                      <td className="py-3.5 font-medium text-slate-200">{tx.description}</td>
                      <td className="py-3.5 text-center">
                        <span
                          className={`px-2 py-0.5 rounded font-bold text-[9px] uppercase ${
                            tx.type === 'ADD'
                              ? 'bg-green-950 text-green-400'
                              : 'bg-red-950 text-red-400'
                          }`}
                        >
                          {tx.type === 'ADD' ? 'Entrada' : 'Saída'}
                        </span>
                      </td>
                      <td className={`py-3.5 text-right font-mono font-bold ${tx.type === 'ADD' ? 'text-green-400' : 'text-red-400'}`}>
                        {tx.type === 'ADD' ? `+${tx.amount}` : `-${tx.amount}`}
                      </td>
                      <td className="py-3.5 text-right text-slate-500">
                        {new Date(tx.created_at).toLocaleString('pt-BR')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination Controls */}
            {total > limit && (
              <div className="flex justify-between items-center pt-4 border-t border-slate-850 text-xs">
                <span className="text-slate-500">
                  Exibindo {txs.length} de {total} transações (Pág. {page})
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
