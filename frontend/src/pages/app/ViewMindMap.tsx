import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api } from '../../services/api';

interface AINode {
  id: string;
  parentId: string | null;
  title: string;
  content: string;
  level: number;
  order: number;
}

interface AIEdge {
  source: string;
  target: string;
}

interface MindMapData {
  title: string;
  centralTopic: string;
  summary: string;
  nodes: AINode[];
  edges: AIEdge[];
}

interface MindMap {
  id: string;
  title: string;
  source_type: string;
  json_data: MindMapData;
  created_at: string;
}

export default function ViewMindMap() {
  const { id } = useParams();
  const [map, setMap] = useState<MindMap | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Node toggle open/close state
  const [collapsedNodes, setCollapsedNodes] = useState<Record<string, boolean>>({});

  useEffect(() => {
    fetchMap();
  }, [id]);

  const fetchMap = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api.get(`/mindmaps/${id}`);
      setMap(res.data.data);
    } catch (err: any) {
      setError('Erro ao carregar mapa mental.');
    } finally {
      setLoading(false);
    }
  };

  const toggleCollapse = (nodeId: string) => {
    setCollapsedNodes((prev) => ({
      ...prev,
      [nodeId]: !prev[nodeId],
    }));
  };

  if (loading) {
    return <p className="text-slate-400 text-xs">Carregando mapa mental...</p>;
  }

  if (error || !map) {
    return (
      <div className="p-4 bg-red-500/10 border border-red-500/35 rounded-xl text-red-400 text-xs">
        ⚠️ {error || 'Mapa mental não encontrado.'}
        <div className="mt-4">
          <Link to="/app/maps" className="text-purple-400 underline">Voltar para meus mapas</Link>
        </div>
      </div>
    );
  }

  // Parse map structure safely
  const mapData = map.json_data || { title: map.title, centralTopic: '', summary: '', nodes: [], edges: [] };
  const nodes = mapData.nodes || [];

  // Group nodes by parentId
  const nodesByParent: Record<string, AINode[]> = {};
  let rootNode: AINode | null = null;

  nodes.forEach((n) => {
    if (n.parentId === null || n.parentId === '' || n.id === 'root') {
      rootNode = n;
    } else {
      if (!nodesByParent[n.parentId]) {
        nodesByParent[n.parentId] = [];
      }
      nodesByParent[n.parentId].push(n);
    }
  });

  // Sort nodes by order
  Object.keys(nodesByParent).forEach((key) => {
    nodesByParent[key].sort((a, b) => a.order - b.order);
  });

  // Recursive Tree Node Renderer
  const renderNode = (node: AINode) => {
    const children = nodesByParent[node.id] || [];
    const isCollapsed = !!collapsedNodes[node.id];
    const hasChildren = children.length > 0;

    return (
      <div key={node.id} className="ml-6 border-l border-slate-800 pl-4 relative my-2">
        {/* Connection bullet */}
        <div className="absolute top-3 -left-[5px] w-2.5 h-2.5 rounded-full bg-purple-600 border border-slate-950" />

        <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 hover:border-slate-700 transition-all space-y-2 max-w-2xl">
          <div className="flex justify-between items-center gap-4">
            <h4 className="font-bold text-slate-100 text-xs flex items-center gap-2">
              <span className="px-1.5 py-0.5 rounded text-[8px] bg-purple-950 text-purple-400 font-mono font-bold">
                Nível {node.level}
              </span>
              {node.title}
            </h4>
            {hasChildren && (
              <button
                onClick={() => toggleCollapse(node.id)}
                className="text-[10px] text-purple-400 hover:text-purple-300 font-bold px-2 py-0.5 rounded bg-slate-950 cursor-pointer"
              >
                {isCollapsed ? 'Expandir' : 'Recolher'}
              </button>
            )}
          </div>
          <p className="text-slate-400 text-xs leading-relaxed font-sans">{node.content}</p>
        </div>

        {hasChildren && !isCollapsed && (
          <div className="mt-2 space-y-2">
            {children.map((child) => renderNode(child))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-start">
        <div>
          <Link to="/app/maps" className="text-[10px] text-purple-400 hover:underline">
            ← Voltar para Meus Mapas
          </Link>
          <h1 className="text-2xl font-bold text-slate-100 mt-1">🧠 {map.title}</h1>
          <p className="text-slate-400 text-xs mt-0.5">Origem: <span className="uppercase text-purple-400 font-bold">{map.source_type}</span></p>
        </div>
        <button
          onClick={() => alert('O Editor Visual com React Flow e interações completas está planejado para a Fase 4!')}
          className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-slate-100 font-bold text-xs rounded-xl transition-all cursor-pointer shadow-lg"
        >
          👁️ Abrir Editor Visual (Placeholder)
        </button>
      </div>

      {/* Summary Card */}
      {mapData.summary && (
        <div className="p-6 bg-slate-900 border border-slate-800 rounded-2xl space-y-2">
          <h3 className="font-bold text-slate-200 text-xs">Resumo do Mapa Mental</h3>
          <p className="text-slate-400 text-xs leading-relaxed font-sans">{mapData.summary}</p>
        </div>
      )}

      {/* Hierarchical Tree Render */}
      <div className="p-6 bg-slate-950 border border-slate-900 rounded-2xl space-y-4">
        <h3 className="font-bold text-slate-200 text-xs mb-6">Estrutura Hierárquica</h3>
        {rootNode ? (
          <div className="-ml-6">
            {renderNode(rootNode)}
          </div>
        ) : (
          <p className="text-slate-500 text-xs">Nenhum nó de hierarquia encontrado neste mapa mental.</p>
        )}
      </div>
    </div>
  );
}
