import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../../services/api';

export default function CreateMindMap() {
  const navigate = useNavigate();
  const [formData, setFormData] = useState({
    type: 'TOPIC', // TOPIC, TEXT, or PDF
    title: '',
    content: '',
    depth: 3,
    language: 'pt-BR',
    style: 'study',
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // PDF specific states
  const [pdfFile, setPdfFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState('');
  const [uploadedId, setUploadedId] = useState<string | null>(null);
  const [uploadStatus, setUploadStatus] = useState<string>('');
  const [uploadError, setUploadError] = useState<string>('');
  const [ragQuery, setRagQuery] = useState('');
  
  // Polling State
  const [pollingJobId, setPollingJobId] = useState<string | null>(null);
  const [jobStatus, setJobStatus] = useState<string>('');
  const [jobErrorMsg, setJobErrorMsg] = useState<string>('');

  // 1. Polling for AI Mindmap generation job
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
          console.error('Erro de polling do mapa:', err);
        }
      }, 2500);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [pollingJobId, navigate]);

  // 2. Polling for PDF upload processor RAG status
  useEffect(() => {
    let interval: any;
    if (uploadedId && (uploadStatus === 'UPLOADED' || uploadStatus === 'PROCESSING')) {
      interval = setInterval(async () => {
        try {
          const res = await api.get(`/uploads/${uploadedId}`);
          const upload = res.data.data.upload;
          setUploadStatus(upload.status);

          if (upload.status === 'PROCESSED') {
            clearInterval(interval);
          } else if (upload.status === 'FAILED') {
            clearInterval(interval);
            // Check metadata error details if present
            const meta = upload.metadata || {};
            setUploadError(meta.error || 'Falha ao processar e indexar o PDF.');
          }
        } catch (err) {
          console.error('Erro ao consultar status do upload:', err);
        }
      }, 2000);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [uploadedId, uploadStatus]);

  // PDF File Upload Handler
  const handlePdfUpload = async () => {
    if (!pdfFile) return;
    setError('');
    setUploadError('');
    setUploading(true);
    setUploadProgress('Enviando arquivo PDF para o servidor...');

    const uploadData = new FormData();
    uploadData.append('file', pdfFile);

    try {
      const res = await api.post('/uploads', uploadData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });

      const { upload } = res.data.data;
      setUploadedId(upload.id);
      setUploadStatus(upload.status || 'UPLOADED');
      setUploadProgress('Arquivo enviado! Processando textos e gerando embeddings...');
      
      // Auto-fill title with original PDF name (sans extension)
      const nameWithoutExt = upload.original_name.replace(/\.pdf$/i, '');
      setFormData((prev) => ({ ...prev, title: prev.title || nameWithoutExt }));
    } catch (err: any) {
      setPdfFile(null);
      if (err.response && err.response.data && err.response.data.message) {
        setUploadError(err.response.data.message);
      } else {
        setUploadError('Erro ao realizar o upload do PDF.');
      }
    } finally {
      setUploading(false);
    }
  };

  const handleClearPdf = () => {
    setPdfFile(null);
    setUploadedId(null);
    setUploadStatus('');
    setUploadError('');
    setUploadProgress('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setJobErrorMsg('');

    if (formData.type === 'PDF' && !uploadedId) {
      setError('Por favor, envie o PDF antes de iniciar a geração.');
      return;
    }

    setLoading(true);

    try {
      let res;
      if (formData.type === 'PDF') {
        res = await api.post('/mindmaps/generate-from-upload', {
          uploadId: uploadedId,
          query: ragQuery,
          options: {
            depth: formData.depth,
            language: formData.language,
            style: formData.style,
          },
        });
      } else {
        res = await api.post('/mindmaps/generate', {
          type: formData.type,
          title: formData.title,
          content: formData.content,
          options: {
            depth: formData.depth,
            language: formData.language,
            style: formData.style,
          },
        });
      }

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
        <p className="text-slate-400 text-xs mt-1">Gere mapas conceituais inteligentes por tema, texto colado ou a partir de um documento PDF.</p>
      </div>

      {error && (
        <div className="p-4 bg-red-500/10 border border-red-500/35 rounded-xl text-red-400 text-xs font-semibold">
          ⚠️ {error}
        </div>
      )}

      {jobErrorMsg && (
        <div className="p-4 bg-red-500/10 border border-red-500/35 rounded-xl text-red-400 text-xs font-semibold">
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
          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Método de Entrada</label>
              <select
                value={formData.type}
                onChange={(e) => {
                  setFormData({ ...formData, type: e.target.value });
                  setError('');
                }}
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              >
                <option value="TOPIC">Geração por Tema / Tópico</option>
                <option value="TEXT">Geração a partir de Texto Colado</option>
                <option value="PDF">Geração a partir de Documento PDF (RAG)</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Título do Mapa</label>
              <input
                type="text"
                required
                value={formData.title}
                onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                placeholder={formData.type === 'PDF' ? 'Nome do Mapa (auto-preenche com nome do PDF)' : 'Ex: Revolução Industrial ou Física'}
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              />
            </div>
          </div>

          {/* INPUT FORM CONTENT ACCORDING TO TYPE */}
          {formData.type === 'TOPIC' && (
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Tema / Palavras-chave</label>
              <input
                type="text"
                required
                maxLength={300}
                value={formData.content}
                onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                placeholder="Descreva o tema com detalhes (ex: Mitose e divisão celular biológica, máx 300 caracteres)"
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
              />
            </div>
          )}

          {formData.type === 'TEXT' && (
            <div>
              <label className="block text-xs font-semibold text-slate-400 mb-1">Conteúdo do Texto</label>
              <textarea
                required
                maxLength={20000}
                rows={8}
                value={formData.content}
                onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                placeholder="Cole o artigo, resumo ou anotações (máx 20.000 caracteres)"
                className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600 font-sans"
              />
            </div>
          )}

          {formData.type === 'PDF' && (
            <div className="space-y-4">
              <label className="block text-xs font-semibold text-slate-400">Arquivo Documento PDF (Limite: 20MB)</label>
              
              {uploadError && (
                <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-400 text-xs rounded-xl font-semibold">
                  ⚠️ {uploadError}
                </div>
              )}

              {/* UPLOADER OR PREVIEW CONTAINER */}
              {!uploadedId ? (
                <div className="border-2 border-dashed border-slate-800 rounded-2xl p-8 flex flex-col items-center justify-center bg-slate-950/40 text-center space-y-3">
                  <span className="text-3xl">📄</span>
                  <div>
                    <p className="text-xs text-slate-350 font-bold">Selecione o arquivo PDF para upload</p>
                    <p className="text-[10px] text-slate-500 mt-0.5">Apenas arquivos textuais no formato PDF são aceitos.</p>
                  </div>
                  {uploading && (
                    <p className="text-[10px] text-purple-400 font-semibold animate-pulse">{uploadProgress}</p>
                  )}
                  <input
                    type="file"
                    accept=".pdf,application/pdf"
                    onChange={(e) => {
                      if (e.target.files && e.target.files.length > 0) {
                        const file = e.target.files[0];
                        if (file.size > 20 * 1024 * 1024) {
                          setUploadError('O arquivo excede o limite máximo permitido de 20MB.');
                          return;
                        }
                        setPdfFile(file);
                      }
                    }}
                    className="hidden"
                    id="pdf-upload-input"
                  />
                  <div className="flex items-center gap-2">
                    <label
                      htmlFor="pdf-upload-input"
                      className="px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg text-[10px] font-bold hover:bg-slate-800 cursor-pointer text-slate-300 transition-colors"
                    >
                      Procurar arquivo...
                    </label>
                    {pdfFile && (
                      <button
                        type="button"
                        onClick={handlePdfUpload}
                        disabled={uploading}
                        className="px-3 py-1.5 bg-purple-600 hover:bg-purple-750 text-slate-100 rounded-lg text-[10px] font-bold cursor-pointer transition-colors"
                      >
                        {uploading ? 'Enviando...' : `Enviar "${pdfFile.name}"`}
                      </button>
                    )}
                  </div>
                </div>
              ) : (
                <div className="p-5 rounded-2xl bg-slate-950 border border-slate-850 flex items-center justify-between gap-4">
                  <div className="flex items-center gap-3 min-w-0">
                    <span className="text-2xl">pdf</span>
                    <div className="min-w-0">
                      <p className="text-xs font-bold text-slate-200 truncate">{pdfFile?.name || 'Documento PDF'}</p>
                      <p className="text-[10px] text-slate-400 mt-0.5 flex items-center gap-2">
                        <span>Status:</span>
                        <span className={`px-1.5 py-0.5 rounded text-[8px] uppercase font-bold ${
                          uploadStatus === 'PROCESSED' 
                            ? 'bg-green-950/80 text-green-400 border border-green-500/20'
                            : uploadStatus === 'FAILED'
                            ? 'bg-red-950/80 text-red-400 border border-red-500/20'
                            : 'bg-amber-950/80 text-amber-400 border border-amber-500/20 animate-pulse'
                        }`}>
                          {uploadStatus}
                        </span>
                      </p>
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={handleClearPdf}
                    className="text-[10px] text-slate-500 hover:text-red-400 cursor-pointer"
                  >
                    Excluir e Re-enviar
                  </button>
                </div>
              )}

              {/* RAG Query filter only shown when PDF is ready */}
              {uploadStatus === 'PROCESSED' && (
                <div className="pt-2 animate-fadeIn">
                  <label className="block text-xs font-semibold text-slate-400 mb-1">
                    Filtro de busca RAG (Opcional)
                  </label>
                  <input
                    type="text"
                    value={ragQuery}
                    onChange={(e) => setRagQuery(e.target.value)}
                    placeholder="Ex: Focar no capítulo de mitocôndrias (deixe vazio para mapear o PDF geral)"
                    className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-200 focus:outline-none focus:border-purple-600"
                  />
                </div>
              )}
            </div>
          )}

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
              disabled={loading || (formData.type === 'PDF' && uploadStatus !== 'PROCESSED')}
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
