import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { api } from '../../services/api';
import { useAuthStore } from '../../stores/auth';

interface MindMap {
  id: string;
  title: string;
  source_type: string;
  status: string;
  created_at: string;
}

interface UploadFile {
  id: string;
  filename: string;
  mime_type: string;
  size: number;
  status: string;
  created_at: string;
}

interface GenJob {
  id: string;
  type: string;
  status: string;
  error?: string;
  credits_cost: number;
  created_at: string;
}

interface PlanLimits {
  max_maps: number;
  max_files: number;
  max_users: number;
  max_storage_bytes: number;
}

interface PlanInfo {
  name: string;
  id: string;
  status: string;
  features: Record<string, boolean>;
  limits: PlanLimits;
}

interface UsageInfo {
  maps_count: number;
  uploads_count: number;
  storage_bytes: number;
  users_count: number;
}

export default function Dashboard() {
  const navigate = useNavigate();
  const { activeOrgId } = useAuthStore();
  const [plan, setPlan] = useState<PlanInfo | null>(null);
  const [usage, setUsage] = useState<UsageInfo | null>(null);
  const [maps, setMaps] = useState<MindMap[]>([]);
  const [uploads, setUploads] = useState<UploadFile[]>([]);
  const [jobs, setJobs] = useState<GenJob[]>([]);
  const [credits, setCredits] = useState(0);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (activeOrgId) {
      fetchDashboardData();
    }
  }, [activeOrgId]);

  const fetchDashboardData = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await api.get('/dashboard');
      const data = response.data.data;
      setPlan(data.plan);
      setUsage(data.usage);
      setCredits(data.credits?.balance ?? 0);
      setMaps(data.recent_maps ?? []);
      setUploads(data.recent_uploads ?? []);
      setJobs(data.recent_jobs ?? []);
    } catch (err) {
      console.error('Erro ao carregar dados do painel:', err);
      setError('Erro ao carregar dados consolidados do painel.');
    } finally {
      setLoading(false);
    }
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    const file = files[0];
    setUploading(true);
    setError('');
    setSuccess('');

    const formData = new FormData();
    formData.append('file', file);

    try {
      await api.post('/uploads', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      setSuccess('Arquivo enviado com sucesso para o armazenamento seguro!');
      fetchDashboardData();
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Falha ao enviar arquivo. Verifique os limites do seu plano.');
      }
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const renderLimitProgress = (current: number, max: number, label: string, formatter?: (v: number) => string) => {
    const isUnlimited = max >= 1000 || max <= 0;
    const percent = isUnlimited ? 0 : Math.min(100, (current / max) * 100);
    const displayCurrent = formatter ? formatter(current) : current.toString();
    const displayMax = formatter ? formatter(max) : max.toString();

    return (
      <div className="space-y-1 text-xs">
        <div className="flex justify-between text-slate-400 font-semibold">
          <span>{label}</span>
          <span>{isUnlimited ? `${displayCurrent} / Ilimitado` : `${displayCurrent} / ${displayMax}`}</span>
        </div>
        {!isUnlimited && (
          <div className="w-full bg-slate-950 h-1.5 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full ${percent >= 90 ? 'bg-red-500' : percent >= 75 ? 'bg-yellow-500' : 'bg-purple-600'}`}
              style={{ width: `${percent}%` }}
            />
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="space-y-8">
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

      {/* Main Grid: Plan, Credits, Limits */}
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* Plan card */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col justify-between hover:border-slate-700 transition-all">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">Plano Atual</p>
            <h3 className="text-2xl font-bold text-slate-100 mb-1">{loading ? 'Carregando...' : plan?.name}</h3>
            <p className="text-xs text-slate-400">Assinatura status: <span className="text-purple-400 font-semibold uppercase">{plan?.status}</span></p>
          </div>
          <button
            onClick={() => navigate('/app/billing')}
            className="mt-6 w-full text-center py-2 bg-slate-800 hover:bg-slate-750 text-slate-200 text-xs font-bold rounded-lg transition-colors border border-slate-750"
          >
            Gerenciar Assinatura
          </button>
        </div>

        {/* Credits card */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col justify-between hover:border-slate-700 transition-all">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">Créditos de IA</p>
            <h3 className="text-3xl font-extrabold text-purple-400 mb-1">{loading ? '...' : `${credits} CRD`}</h3>
            <p className="text-xs text-slate-400">Utilizados para criar mapas por tema, texto ou PDF</p>
          </div>
          <button
            onClick={() => navigate('/app/credits')}
            className="mt-6 w-full text-center py-2 bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold rounded-lg transition-colors shadow-lg shadow-purple-500/10"
          >
            Ver Extrato de Créditos
          </button>
        </div>

        {/* Limits card */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-slate-700 transition-all space-y-4">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">Limites do Plano</p>
          {loading ? (
            <p className="text-xs text-slate-500">Calculando limites...</p>
          ) : (
            <div className="space-y-3">
              {renderLimitProgress(usage?.maps_count ?? 0, plan?.limits?.max_maps ?? 0, 'Mapas Mentais')}
              {renderLimitProgress(usage?.uploads_count ?? 0, plan?.limits?.max_files ?? 0, 'Documentos')}
              {renderLimitProgress(usage?.storage_bytes ?? 0, plan?.limits?.max_storage_bytes ?? 0, 'Armazenamento', formatBytes)}
            </div>
          )}
        </div>
      </div>

      {/* Quick AI Mindmap Generators */}
      <div>
        <h2 className="text-lg font-bold mb-4">Geração Automatizada com IA</h2>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
          <div
            onClick={() => navigate('/app/maps/new')}
            className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-600/40 transition-all cursor-pointer group"
          >
            <span className="text-2xl mb-4 block">💡</span>
            <h3 className="font-bold text-slate-100 group-hover:text-purple-400 transition-colors mb-2">
              Gerar por Tema/Tópico
            </h3>
            <p className="text-xs text-slate-400 leading-relaxed">
              Insira um tema central (ex: "Estruturas de Dados") para criar um mapa conceitual.
            </p>
          </div>

          <div
            onClick={() => navigate('/app/maps/new')}
            className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-600/40 transition-all cursor-pointer group"
          >
            <span className="text-2xl mb-4 block">📝</span>
            <h3 className="font-bold text-slate-100 group-hover:text-purple-400 transition-colors mb-2">
              Gerar por Texto Colado
            </h3>
            <p className="text-xs text-slate-400 leading-relaxed">
              Cole anotações de aulas ou artigos e deixe a IA estruturar as conexões de ideias.
            </p>
          </div>

          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-600/40 transition-all cursor-pointer group relative">
            <input
              type="file"
              ref={fileInputRef}
              onChange={handleFileUpload}
              className="hidden"
              accept=".pdf"
            />
            <div
              onClick={() => !uploading && fileInputRef.current?.click()}
              className="h-full flex flex-col justify-between"
            >
              <div>
                <span className="text-2xl mb-4 block">📂</span>
                <h3 className="font-bold text-slate-100 group-hover:text-purple-400 transition-colors mb-2">
                  {uploading ? 'Enviando arquivo...' : 'Upload de PDF/Documento'}
                </h3>
                <p className="text-xs text-slate-400 leading-relaxed">
                  {uploading ? 'Aguarde o processamento...' : 'Faça upload de apostilas ou resumos para extrair mapas mentais.'}
                </p>
              </div>
              {uploading && (
                <div className="mt-4 w-full bg-slate-950 h-1 rounded-full overflow-hidden">
                  <div className="bg-purple-600 h-full animate-pulse w-2/3" />
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Onboarding Steps */}
      <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
        <h3 className="font-bold text-slate-100 mb-4">🚀 Primeiros Passos / Guia Rápido</h3>
        <div className="grid md:grid-cols-3 gap-6 text-xs leading-relaxed text-slate-400">
          <div className="space-y-2">
            <h4 className="font-bold text-slate-200 flex items-center gap-2">
              <span className="h-5 w-5 rounded bg-purple-950 text-purple-400 flex items-center justify-center font-bold">1</span>
              Assine um Plano
            </h4>
            <p>Se for Super Admin, gerencie integrações no painel. Usuários comuns devem assinar um plano pago para desbloquear limites maiores de armazenamento e geração.</p>
          </div>
          <div className="space-y-2">
            <h4 className="font-bold text-slate-200 flex items-center gap-2">
              <span className="h-5 w-5 rounded bg-purple-950 text-purple-400 flex items-center justify-center font-bold">2</span>
              Crie Mapas conceituais
            </h4>
            <p>Use geradores inteligentes por Tema ou Texto na tela de criação para estruturar instantaneamente grandes volumes de estudo.</p>
          </div>
          <div className="space-y-2">
            <h4 className="font-bold text-slate-200 flex items-center gap-2">
              <span className="h-5 w-5 rounded bg-purple-950 text-purple-400 flex items-center justify-center font-bold">3</span>
              Semântica de PDFs (RAG)
            </h4>
            <p>Envie apostilas no painel de uploads. O worker processará e salvará no banco PGVector, permitindo fazer perguntas e criar mapas refinados de capítulos inteiros.</p>
          </div>
        </div>
      </div>

      {/* Activity Grid */}
      <div className="grid lg:grid-cols-3 gap-8">
        {/* Mindmaps list */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col lg:col-span-1">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between">
            <span>Mapas Recentes</span>
            <button onClick={() => navigate('/app/maps')} className="text-xs text-purple-400 hover:underline cursor-pointer">Ver todos</button>
          </h3>

          {loading ? (
            <p className="text-xs text-slate-500 py-4">Carregando mapas...</p>
          ) : maps.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center py-10 text-center text-slate-500">
              <span className="text-3xl mb-2">🧠</span>
              <p className="text-xs leading-relaxed">Nenhum mapa gerado ainda.</p>
            </div>
          ) : (
            <div className="divide-y divide-slate-850">
              {maps.map((m) => (
                <div key={m.id} className="py-3 flex justify-between items-center text-xs">
                  <div>
                    <Link to={`/app/maps/${m.id}/editor`} className="font-bold text-slate-200 hover:text-purple-400 hover:underline">
                      🧠 {m.title}
                    </Link>
                    <p className="text-[10px] text-slate-500">
                      {m.source_type} &bull; {new Date(m.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <span className="px-2 py-0.5 rounded bg-purple-950 text-purple-400 font-semibold">{m.status}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Documents List */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col lg:col-span-1">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between">
            <span>Documentos</span>
            <button onClick={() => navigate('/app/maps/new')} className="text-xs text-purple-400 hover:underline cursor-pointer">Novo upload</button>
          </h3>

          {loading ? (
            <p className="text-xs text-slate-500 py-4">Carregando arquivos...</p>
          ) : uploads.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center py-10 text-center text-slate-500">
              <span className="text-3xl mb-2">📁</span>
              <p className="text-xs leading-relaxed">Nenhum documento anexado.</p>
            </div>
          ) : (
            <div className="divide-y divide-slate-850">
              {uploads.map((file) => (
                <div key={file.id} className="py-3 flex justify-between items-center text-xs">
                  <div>
                    <p className="font-bold text-slate-200 truncate max-w-[150px]">{file.filename}</p>
                    <p className="text-[10px] text-slate-500">
                      {formatBytes(file.size)} &bull; {new Date(file.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-bold uppercase text-[9px]">
                    {file.status}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* AI Jobs History List */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col lg:col-span-1">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between">
            <span>Histórico de Jobs</span>
            <button onClick={() => navigate('/app/generation-jobs')} className="text-xs text-purple-400 hover:underline cursor-pointer">Ver todos</button>
          </h3>

          {loading ? (
            <p className="text-xs text-slate-500 py-4">Carregando jobs...</p>
          ) : jobs.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center py-10 text-center text-slate-500">
              <span className="text-3xl mb-2">⚙️</span>
              <p className="text-xs leading-relaxed">Nenhum job processado ainda.</p>
            </div>
          ) : (
            <div className="divide-y divide-slate-850">
              {jobs.map((job) => (
                <div key={job.id} className="py-3 flex justify-between items-center text-xs">
                  <div>
                    <p className="font-bold text-slate-200 truncate max-w-[160px]">{job.type}</p>
                    <p className="text-[10px] text-slate-500">
                      {job.credits_cost} CRD &bull; {new Date(job.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <span
                    className={`px-2 py-0.5 rounded font-bold text-[9px] ${
                      job.status === 'COMPLETED'
                        ? 'bg-green-950 text-green-400'
                        : job.status === 'FAILED'
                        ? 'bg-red-950 text-red-400'
                        : 'bg-yellow-950 text-yellow-400'
                    }`}
                  >
                    {job.status}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
