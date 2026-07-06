import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../../services/api';

export default function CreateMindMap() {
  const navigate = useNavigate();
  const [formData, setFormData] = useState({
    type: 'TOPIC', // TOPIC or TEXT
    title: '',
    content: '',
    depth: 3,
    language: 'pt-BR',
    style: 'study',
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  
  // Polling State
  const [pollingJobId, setPollingJobId] = useState<string | null>(null);
  const [jobStatus, setJobStatus] = useState<string>('');
  const [jobErrorMsg, setJobErrorMsg] = useState<string>('');

  useEffect(() => {
    let interval: any;
    if (pollingJobId) {
      interval = setInterval(async () => {
        try {
          const res = await api.get(`/generation-jobs/${pollingJobId}`);
          const job = res.data.data;
          setJobStatus(job.status);
          
          if (job.status === 'COMPLETED') {
            clearInterval(interval);
            setPollingJobId(null);
            // Redirect to view the mind map
            if (job.mind_map_id) {
              navigate(`/app/maps/${job.mind_map_id}`);
            } else {
              setError('Geração concluída, mas o identificador do mapa não foi retornado.');
            }
          } else if (job.status === 'FAILED') {
            clearInterval(interval);
            setPollingJobId(null);
            setJobErrorMsg(job.error || 'A geração com Inteligência Artificial falhou.');
            setLoading(false);
          }
        } catch (err) {
          console.error('Erro de polling:', err);
        }
      }, 2500);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [pollingJobId, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setJobErrorMsg('');
    setLoading(true);

    try {
      const res = await api.post('/mindmaps/generate', {
        type: formData.type,
        title: formData.title,
        content: formData.content,
        options: {
          depth: formData.depth,
          language: formData.language,
          style: formData.style,
        },
      });

      const { jobId, status } = res.data.data;
      setPollingJobId(jobId);
      setJobStatus(status || 'PENDING');
    } catch (err: any) {
      setLoading(false);
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Ocorreu um erro ao enviar a requisição de geração.');
      }
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-100">Criar Novo Mapa Mental</h1>
        <p className="text-slate-400 text-xs mt-1">Gere mapas conceituais inteligentes por tema ou a partir de um texto colado.</p>
      </div>

      {error && (
        <div className="p-4 bg-red-500/10 border border-red-500/35 rounded-xl text-red-400 text-xs">
          ⚠️ {error}
        </div>
      )}

      {jobErrorMsg && (
        <div className="p-4 bg-red-500/10 border border-red-500/35 rounded-xl text-red-400 text-xs">
          ❌ <strong>Falha na IA:</strong> {jobErrorMsg}
        </div>
      )}

      {pollingJobId ? (
        <div className="p-8 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col items-center justify-center text-center space-y-4">
          <div className="h-12 w-12 rounded-full border-4 border-purple-500/20 border-t-purple-600 animate-spin" />
          <div>
            <h3 className="font-bold text-slate-200">Processando Mapa Mental</h3>
            <p className="text-xs text-slate-400 mt-1">A Inteligência Artificial está sintetizando os dados. Por favor, aguarde.</p>
          </div>
          <div className="px-3 py-1 bg-purple-950 text-purple-400 text-[10px] font-mono rounded-lg uppercase font-bold animate-pulse">
            Status: {jobStatus}
          </div>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="p-6 bg-slate-900 border border-slate-800 rounded-2xl space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Método de Entrada</label>
              <select
                value={formData.type}
                onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              >
                <option value="TOPIC">Geração por Tema / Tópico</option>
                <option value="TEXT">Geração a partir de Texto Colado</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Título do Mapa</label>
              <input
                type="text"
                required
                value={formData.title}
                onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                placeholder="Ex: Revolução Industrial ou Resumo de Física"
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 mb-1">
              {formData.type === 'TOPIC' ? 'Tema / Palavras-chave' : 'Conteúdo do Texto'}
            </label>
            {formData.type === 'TOPIC' ? (
              <input
                type="text"
                required
                maxLength={300}
                value={formData.content}
                onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                placeholder="Descreva o tema com detalhes (máx 300 caracteres)"
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              />
            ) : (
              <textarea
                required
                maxLength={20000}
                rows={8}
                value={formData.content}
                onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                placeholder="Cole o artigo, resumo ou anotações (máx 20.000 caracteres)"
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600 font-sans"
              />
            )}
          </div>

          <div className="grid sm:grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Nível de Profundidade</label>
              <select
                value={formData.depth}
                onChange={(e) => setFormData({ ...formData, depth: parseInt(e.target.value) || 3 })}
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              >
                <option value="2">2 níveis (Curto)</option>
                <option value="3">3 níveis (Médio)</option>
                <option value="4">4 níveis (Detalhado)</option>
                <option value="5">5 níveis (Exaustivo)</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Idioma</label>
              <input
                type="text"
                required
                value={formData.language}
                onChange={(e) => setFormData({ ...formData, language: e.target.value })}
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Estilo de Aprendizado</label>
              <select
                value={formData.style}
                onChange={(e) => setFormData({ ...formData, style: e.target.value })}
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              >
                <option value="study">Estudo Acadêmico</option>
                <option value="technical">Documentação Técnica</option>
                <option value="executive">Sumário Executivo</option>
              </select>
            </div>
          </div>

          <div className="pt-4 border-t border-slate-800/80 flex justify-end">
            <button
              type="submit"
              disabled={loading}
              className="px-6 py-2.5 bg-purple-600 hover:bg-purple-700 text-slate-100 font-bold text-xs rounded-xl transition-all cursor-pointer disabled:opacity-50"
            >
              {loading ? 'Inicializando...' : 'Gerar Mapa com IA'}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
