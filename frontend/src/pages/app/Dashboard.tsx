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

export default function Dashboard() {
  const navigate = useNavigate();
  const { activeOrgId } = useAuthStore();
  const [maps, setMaps] = useState<MindMap[]>([]);
  const [uploads, setUploads] = useState<UploadFile[]>([]);
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
      const [balanceRes, uploadsRes, mapsRes] = await Promise.all([
        api.get('/credits/balance'),
        api.get('/uploads?limit=5'),
        api.get('/mindmaps'),
      ]);
      setCredits(balanceRes.data.data.balance || 0);
      setUploads(uploadsRes.data.data.uploads || []);
      setMaps((mapsRes.data.data || []).slice(0, 5));
    } catch (err) {
      console.error('Erro ao carregar dados do painel:', err);
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
        setError('Falha ao enviar arquivo. Verifique se o MinIO está ativo.');
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

      {/* Quick Stats Grid */}
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 relative overflow-hidden group hover:border-purple-900/35 transition-all">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Plano & Workspace
          </p>
          <h3 className="text-xl font-bold text-slate-100 mb-2">Workspace Ativo</h3>
          <p className="text-xs text-slate-400">Ambiente de estudos corporativo / multiempresa</p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-900/35 transition-all">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Créditos de IA Disponíveis
          </p>
          <h3 className="text-2xl font-bold text-purple-400 mb-1">
            {loading ? '...' : `${credits} CRD`}
          </h3>
          <p className="text-xs text-slate-400">Usados para geração automatizada de mapas</p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-900/35 transition-all">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Arquivos PDF Enviados
          </p>
          <h3 className="text-xl font-bold text-slate-100 mb-2">
            {loading ? '...' : `${uploads.length} documentos`}
          </h3>
          <p className="text-xs text-slate-400">Processados no banco PGVector/RAG</p>
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
              accept=".pdf,.txt,.doc,.docx"
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

      {/* Lists of Recent Activity */}
      <div className="grid lg:grid-cols-2 gap-8">
        {/* Mindmaps list */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between">
            <span>Mapas Mentais Recentes</span>
            <button onClick={() => navigate('/app/maps')} className="text-xs text-purple-400 hover:underline cursor-pointer">Ver todos</button>
          </h3>

          {loading ? (
            <p className="text-xs text-slate-500 py-4">Carregando mapas...</p>
          ) : maps.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center py-10 text-center text-slate-500">
              <span className="text-3xl mb-2">🧠</span>
              <p className="text-xs max-w-[250px] leading-relaxed">Nenhum mapa gerado. Use uma das opções acima para começar!</p>
            </div>
          ) : (
            <div className="divide-y divide-slate-800">
              {maps.map((m) => (
                <div key={m.id} className="py-3 flex justify-between items-center text-xs">
                  <div>
                    <Link to={`/app/maps/${m.id}`} className="font-bold text-slate-200 hover:text-purple-400 hover:underline">
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

        {/* Files list */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 flex flex-col">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between">
            <span>Documentos Enviados</span>
            <button className="text-xs text-purple-400 hover:underline cursor-pointer">Ver todos</button>
          </h3>

          {loading ? (
            <p className="text-xs text-slate-500 py-4">Carregando arquivos...</p>
          ) : uploads.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center py-10 text-center text-slate-500">
              <span className="text-3xl mb-2">📁</span>
              <p className="text-xs max-w-[250px] leading-relaxed">Nenhum documento anexado a este workspace ainda.</p>
            </div>
          ) : (
            <div className="divide-y divide-slate-800">
              {uploads.map((file) => (
                <div key={file.id} className="py-3 flex justify-between items-center text-xs">
                  <div>
                    <p className="font-bold text-slate-200 truncate max-w-[250px]">{file.filename}</p>
                    <p className="text-[10px] text-slate-550 mt-0.5 font-mono">
                      {formatBytes(file.size)} &bull; {new Date(file.created_at).toLocaleString('pt-BR')}
                    </p>
                  </div>
                  <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-bold uppercase text-[9px] tracking-wide">
                    {file.status}
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
