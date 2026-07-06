import { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface MindMap {
  id: string;
  title: string;
  source_type: string;
  status: string;
  created_at: string;
}

export default function Dashboard() {
  const [maps, setMaps] = useState<MindMap[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Fetch user maps
    api.get('/health') // Placeholder for maps fetch in Phase 1
      .then(() => {
        setMaps([]); // Mocking empty state for Phase 1
        setLoading(false);
      })
      .catch(() => {
        setLoading(false);
      });
  }, []);

  return (
    <div className="space-y-8">
      {/* Quick Stats Grid */}
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Plano Atual
          </p>
          <h3 className="text-xl font-bold text-slate-100 mb-2">Gratuito (Free Trial)</h3>
          <p className="text-xs text-slate-400">Expira em: Sem data de validade</p>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Mapas Mentais
          </p>
          <h3 className="text-xl font-bold text-slate-100 mb-2">{maps.length} / 3</h3>
          <div className="w-full bg-slate-950 h-1.5 rounded-full overflow-hidden mt-3 border border-slate-850">
            <div className="bg-purple-600 h-full rounded-full" style={{ width: '0%' }} />
          </div>
        </div>

        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-850">
          <p className="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-1">
            Arquivos PDF
          </p>
          <h3 className="text-xl font-bold text-slate-100 mb-2">0 / 0</h3>
          <p className="text-xs text-slate-400">Limite da sua assinatura ativa</p>
        </div>
      </div>

      {/* Quick AI Mindmap Generators */}
      <div>
        <h2 className="text-lg font-bold mb-4">Criar Novo Mapa Mental com IA</h2>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {/* Option 1: Topic */}
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-600/40 transition-all cursor-pointer group">
            <span className="text-2xl mb-4 block">💡</span>
            <h3 className="font-bold text-slate-100 group-hover:text-purple-400 transition-colors mb-2">
              Gerar por Tema/Tópico
            </h3>
            <p className="text-xs text-slate-400 leading-relaxed">
              Digite um conceito (ex: "Mitose", "Segunda Lei de Newton") e a IA criará o mapa.
            </p>
          </div>

          {/* Option 2: Text */}
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-600/40 transition-all cursor-pointer group">
            <span className="text-2xl mb-4 block">📝</span>
            <h3 className="font-bold text-slate-100 group-hover:text-purple-400 transition-colors mb-2">
              Gerar por Texto Colado
            </h3>
            <p className="text-xs text-slate-400 leading-relaxed">
              Cole artigos, anotações de aulas ou resumos diretamente no campo e a IA estruturará.
            </p>
          </div>

          {/* Option 3: PDF */}
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 hover:border-purple-600/40 transition-all cursor-pointer group">
            <span className="text-2xl mb-4 block">📂</span>
            <h3 className="font-bold text-slate-100 group-hover:text-purple-400 transition-colors mb-2">
              Gerar por PDF/Documento
            </h3>
            <p className="text-xs text-slate-400 leading-relaxed">
              Faça upload de livros inteiros ou resenhas em PDF para gerar o mapa do arquivo.
            </p>
          </div>
        </div>
      </div>

      {/* Lists of Recent Activity */}
      <div className="grid lg:grid-cols-2 gap-8">
        {/* Mindmaps list */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between">
            <span>Mapas Recentes</span>
            <button className="text-xs text-purple-400 hover:underline">Ver todos</button>
          </h3>

          {loading ? (
            <p className="text-xs text-slate-500 py-4">Carregando mapas...</p>
          ) : maps.length === 0 ? (
            <div className="text-center py-8 text-slate-500">
              <span className="text-3xl mb-2 block">🧠</span>
              <p className="text-xs">Nenhum mapa criado neste workspace ainda.</p>
            </div>
          ) : (
            <div className="divide-y divide-slate-800">
              {maps.map((m) => (
                <div key={m.id} className="py-3 flex justify-between items-center text-xs">
                  <div>
                    <p className="font-bold text-slate-200">{m.title}</p>
                    <p className="text-[10px] text-slate-500">{m.source_type} &bull; {new Date(m.created_at).toLocaleDateString()}</p>
                  </div>
                  <span className="px-2 py-0.5 rounded bg-purple-950 text-purple-400 font-semibold">{m.status}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Files list */}
        <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800">
          <h3 className="font-bold text-slate-100 mb-4 flex items-center justify-between">
            <span>Arquivos Processados</span>
            <button className="text-xs text-purple-400 hover:underline">Ver todos</button>
          </h3>

          <div className="text-center py-8 text-slate-500">
            <span className="text-3xl mb-2 block">📁</span>
            <p className="text-xs">Nenhum arquivo enviado para o MinIO ainda.</p>
          </div>
        </div>
      </div>
    </div>
  );
}
