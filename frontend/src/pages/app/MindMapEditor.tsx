import { useState, useEffect, useCallback, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  useReactFlow,
} from '@xyflow/react';
import type { Node, Edge } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { toPng } from 'html-to-image';
import { jsPDF } from 'jspdf';

import { api } from '../../services/api';
import {
  normalizeMindMapData,
  removeSubtree,
  countDescendants,
} from '../../components/mindmap-editor/tree-utils';
import type {
  AINode,
  AIEdge,
} from '../../components/mindmap-editor/tree-utils';
import { calculateAutoLayout } from '../../components/mindmap-editor/layout';
import { validateMindMapData } from '../../components/mindmap-editor/validation';
import MindMapNode from '../../components/mindmap-editor/MindMapNode';
import NodeEditPanel from '../../components/mindmap-editor/NodeEditPanel';
import MindMapToolbar from '../../components/mindmap-editor/MindMapToolbar';

// Register custom node type in React Flow
const nodeTypes = {
  mindMapNode: MindMapNode,
};

function FlowEditorInner() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { fitView, getViewport, setViewport } = useReactFlow();

  const [mapTitle, setMapTitle] = useState('');
  const [centralTopic, setCentralTopic] = useState('');
  const [summary, setSummary] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [hasChanges, setHasChanges] = useState(false);
  const [exportingPng, setExportingPng] = useState(false);
  const [exportingPdf, setExportingPdf] = useState(false);

  // React Flow states
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // Selected node state
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  // 1. Fetch mind map from backend
  useEffect(() => {
    fetchMap();
  }, [id]);

  const fetchMap = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api.get(`/mindmaps/${id}`);
      const map = res.data.data;
      setMapTitle(map.title);

      const normalized = normalizeMindMapData(map.json_data);
      setCentralTopic(normalized.centralTopic);
      setSummary(normalized.summary);

      // Auto layout if nodes lack positions
      const hasPositions = normalized.nodes.some(
        (n) => n.position && (n.position.x !== 0 || n.position.y !== 0)
      );
      const laidOutNodes = hasPositions
        ? normalized.nodes
        : calculateAutoLayout(normalized.nodes);

      // Convert domain AINode -> React Flow Node
      const flowNodes: Node[] = laidOutNodes.map((n) => ({
        id: n.id,
        type: 'mindMapNode',
        position: n.position || { x: 0, y: 0 },
        data: {
          title: n.title,
          content: n.content,
          level: n.level,
          isRoot: n.id === 'root',
          onAddChild: handleAddChild,
          onDelete: handleDeleteNode,
        },
      }));

      // Convert domain AIEdge -> React Flow Edge
      const flowEdges: Edge[] = normalized.edges.map((e) => ({
        id: e.id || `edge-${e.source}-${e.target}`,
        source: e.source,
        target: e.target,
        type: 'default',
        style: { stroke: '#8b5cf6', strokeWidth: 2 },
      }));

      setNodes(flowNodes);
      setEdges(flowEdges);

      // Set viewport if saved
      if (normalized.viewport) {
        setTimeout(() => {
          setViewport(normalized.viewport!);
        }, 100);
      } else {
        setTimeout(() => fitView({ padding: 0.2 }), 200);
      }
    } catch (err: any) {
      setError('Erro ao carregar o mapa mental no editor.');
    } finally {
      setLoading(false);
    }
  };

  // Warning when leaving with unsaved changes
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (hasChanges) {
        e.preventDefault();
        return 'Você tem alterações não salvas.';
      }
    };
    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => window.removeEventListener('beforeunload', handleBeforeUnload);
  }, [hasChanges]);

  // Navigate back safely checking changes
  const handleBack = () => {
    if (hasChanges) {
      if (!window.confirm('Você possui alterações não salvas que serão perdidas. Deseja sair?')) {
        return;
      }
    }
    navigate(`/app/maps/${id}`);
  };

  // Convert current Flow nodes back to domain AINodes
  const getDomainNodes = useCallback((): AINode[] => {
    return nodes.map((n) => {
      const parentEdge = edges.find((e) => e.target === n.id);
      return {
        id: n.id,
        parentId: parentEdge ? parentEdge.source : null,
        title: (n.data.title as string) || '',
        content: (n.data.content as string) || '',
        level: (n.data.level as number) ?? 0,
        order: 0, // order resolved by hierarchy
        position: n.position,
        collapsed: false,
      };
    });
  }, [nodes, edges]);

  // Convert current Flow edges back to domain AIEdges
  const getDomainEdges = useCallback((): AIEdge[] => {
    return edges.map((e) => ({
      id: e.id,
      source: e.source,
      target: e.target,
    }));
  }, [edges]);

  // Selected node computed object
  const selectedNode = useMemo(() => {
    if (!selectedNodeId) return null;
    const n = nodes.find((x) => x.id === selectedNodeId);
    if (!n) return null;
    const domainNodes = getDomainNodes();
    return domainNodes.find((x) => x.id === selectedNodeId) || null;
  }, [selectedNodeId, nodes, getDomainNodes]);

  // Update node details (title/content) from NodeEditPanel
  const handleUpdateNode = useCallback((nodeId: string, updates: Partial<AINode>) => {
    setNodes((nds) =>
      nds.map((n) => {
        if (n.id === nodeId) {
          const nextData = { ...n.data };
          if (updates.title !== undefined) nextData.title = updates.title;
          if (updates.content !== undefined) nextData.content = updates.content;
          return {
            ...n,
            data: nextData,
          };
        }
        return n;
      })
    );
    setHasChanges(true);
  }, [setNodes]);

  // Add child node
  const handleAddChild = useCallback((parentId: string) => {
    const parentNode = nodes.find((n) => n.id === parentId);
    if (!parentNode) return;

    const newId = `node_${Date.now()}`;
    const parentLevel = (parentNode.data.level as number) || 0;
    const newLevel = parentLevel + 1;

    // Position new child node slightly offset to the right
    const childPosition = {
      x: parentNode.position.x + 300,
      y: parentNode.position.y + (Math.random() - 0.5) * 160,
    };

    const newFlowNode: Node = {
      id: newId,
      type: 'mindMapNode',
      position: childPosition,
      data: {
        title: 'Novo Tópico',
        content: '',
        level: newLevel,
        isRoot: false,
        onAddChild: handleAddChild,
        onDelete: handleDeleteNode,
      },
    };

    const newFlowEdge: Edge = {
      id: `edge-${parentId}-${newId}`,
      source: parentId,
      target: newId,
      type: 'default',
      style: { stroke: '#8b5cf6', strokeWidth: 2 },
    };

    setNodes((nds) => [...nds, newFlowNode]);
    setEdges((eds) => [...eds, newFlowEdge]);
    setHasChanges(true);
    setSelectedNodeId(newId); // auto select newly created node
  }, [nodes, setNodes, setEdges]);

  // Delete node and its sub-tree recursively
  const handleDeleteNode = useCallback((nodeId: string) => {
    if (nodeId === 'root') {
      alert('Não é possível remover o nó principal.');
      return;
    }

    const domainNodes = getDomainNodes();
    const descendantsCount = countDescendants(nodeId, domainNodes);

    const message = descendantsCount > 0
      ? `Este nó possui ${descendantsCount} subitens. Ao remover, todos eles também serão apagados. Deseja continuar?`
      : 'Deseja remover este item?';

    if (!window.confirm(message)) return;

    const domainEdges = getDomainEdges();
    const { nodes: nextDomainNodes, edges: nextDomainEdges } = removeSubtree(
      nodeId,
      domainNodes,
      domainEdges
    );

    // Sync back to React Flow states
    setNodes((nds) => nds.filter((n) => nextDomainNodes.some((dn) => dn.id === n.id)));
    setEdges((eds) => eds.filter((e) => nextDomainEdges.some((de) => de.id === e.id)));

    if (selectedNodeId === nodeId) {
      setSelectedNodeId(null);
    }
    setHasChanges(true);
  }, [getDomainNodes, getDomainEdges, selectedNodeId, setNodes, setEdges]);

  // Trigger auto layout
  const handleAutoLayout = () => {
    const domainNodes = getDomainNodes();
    const laidOutNodes = calculateAutoLayout(domainNodes);
    setNodes((nds) =>
      nds.map((n) => {
        const matching = laidOutNodes.find((dn) => dn.id === n.id);
        return {
          ...n,
          position: matching ? matching.position || n.position : n.position,
        };
      })
    );
    setHasChanges(true);
    setTimeout(() => fitView({ padding: 0.2 }), 200);
  };

  // Save mind map changes to database
  const handleSave = async () => {
    setError('');
    setSuccess('');

    const domainNodes = getDomainNodes();
    const domainEdges = getDomainEdges();
    const viewport = getViewport();

    const payloadData = {
      title: mapTitle,
      centralTopic,
      summary,
      nodes: domainNodes,
      edges: domainEdges,
      viewport,
    };

    // Validate map integrity
    const validationError = validateMindMapData(payloadData);
    if (validationError) {
      setError(`Validação de integridade falhou: ${validationError.message}`);
      return;
    }

    setSaving(true);
    try {
      await api.patch(`/mindmaps/${id}`, {
        title: mapTitle,
        jsonData: payloadData,
      });
      setSuccess('Mapa mental salvo com sucesso.');
      setHasChanges(false);
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Erro técnico ao salvar o mapa mental.');
      }
    } finally {
      setSaving(false);
    }
  };

  const handleExport = async (format: 'PNG' | 'PDF') => {
    if (format === 'PNG') setExportingPng(true);
    else setExportingPdf(true);

    setError('');
    setSuccess('');

    try {
      // 1. Call Backend check
      const authRes = await api.post(`/mindmaps/${id}/export/check`, { format });
      if (!authRes.data || !authRes.data.data || !authRes.data.data.authorized) {
        throw new Error("Não autorizado");
      }

      // 2. Perform HTML element capture
      const flowElement = document.querySelector('.react-flow') as HTMLElement;
      if (!flowElement) {
        throw new Error("Não foi possível encontrar a tela do mapa mental.");
      }

      // Hide controls and minimap temporarily for clean capture
      const controls = document.querySelector('.react-flow__controls') as HTMLElement;
      const minimap = document.querySelector('.react-flow__minimap') as HTMLElement;
      if (controls) controls.style.visibility = 'hidden';
      if (minimap) minimap.style.visibility = 'hidden';

      const dataUrl = await toPng(flowElement, {
        backgroundColor: '#020617', // Slate 950 matching background
        quality: 0.95,
        style: {
          transform: 'none',
        }
      });

      // Restore visibility
      if (controls) controls.style.visibility = 'visible';
      if (minimap) minimap.style.visibility = 'visible';

      if (format === 'PNG') {
        const link = document.createElement('a');
        link.download = `${mapTitle || 'mapa-mental'}.png`;
        link.href = dataUrl;
        link.click();
        setSuccess('Mapa exportado como PNG com sucesso.');
      } else {
        const pdf = new jsPDF({
          orientation: 'landscape',
          unit: 'px',
          format: [flowElement.offsetWidth, flowElement.offsetHeight]
        });
        pdf.addImage(dataUrl, 'PNG', 0, 0, flowElement.offsetWidth, flowElement.offsetHeight);
        pdf.save(`${mapTitle || 'mapa-mental'}.pdf`);
        setSuccess('Mapa exportado como PDF com sucesso.');
      }
    } catch (err: any) {
      console.error(err);
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else if (err.response && err.response.status === 403) {
        setError('Seu plano atual não permite exportações neste formato.');
      } else {
        setError(err.message || 'Erro técnico na geração da exportação do mapa.');
      }
    } finally {
      if (format === 'PNG') setExportingPng(false);
      else setExportingPdf(false);
    }
  };


  const handleSelectionChange = (params: { nodes: Node[] }) => {
    if (params.nodes.length > 0) {
      setSelectedNodeId(params.nodes[0].id);
    } else {
      setSelectedNodeId(null);
    }
  };

  if (loading) {
    return <div className="p-8 text-slate-400 text-xs">Carregando editor visual...</div>;
  }

  return (
    <div className="h-[calc(100vh-140px)] flex flex-col border border-slate-800 rounded-2xl overflow-hidden bg-slate-950 text-slate-100">
      {/* Editor Header */}
      <div className="flex items-center gap-4 px-6 py-4 bg-slate-900 border-b border-slate-850">
        <button
          onClick={handleBack}
          className="text-xs text-purple-400 hover:text-purple-300 font-bold transition-colors cursor-pointer flex items-center gap-1"
        >
          ← Voltar
        </button>
        <span className="text-slate-700">|</span>
        <h1 className="text-xs font-bold text-slate-350 truncate">
          Mapa: <span className="text-slate-100">{mapTitle || 'Carregando...'}</span>
        </h1>
      </div>

      {/* Editor top toolbar */}
      <MindMapToolbar
        hasChanges={hasChanges}
        onSave={handleSave}
        onAutoLayout={handleAutoLayout}
        onFitView={() => fitView({ padding: 0.2 })}
        saving={saving}
        onExportPng={() => handleExport('PNG')}
        onExportPdf={() => handleExport('PDF')}
        exportingPng={exportingPng}
        exportingPdf={exportingPdf}
      />

      {error && (
        <div className="p-3 bg-red-500/10 border-b border-red-500/30 text-red-400 text-[11px] font-semibold">
          ⚠️ {error}
        </div>
      )}
      {success && (
        <div className="p-3 bg-green-500/10 border-b border-green-500/30 text-green-400 text-[11px] font-semibold">
          ✓ {success}
        </div>
      )}

      {/* Editor Body */}
      <div className="flex-1 flex overflow-hidden">
        {/* React Flow Canvas */}
        <div className="flex-1 h-full relative">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onSelectionChange={handleSelectionChange}
            nodeTypes={nodeTypes}
            fitView
            minZoom={0.1}
            maxZoom={2}
          >
            <Background color="#334155" gap={20} size={1.2} />
            <Controls className="!bg-slate-900 !border-slate-800 !text-slate-300" />
            <MiniMap
              className="!bg-slate-900 !border-slate-800"
              nodeColor="#8b5cf6"
              maskColor="rgba(15, 23, 42, 0.6)"
            />
          </ReactFlow>
        </div>

        {/* Node Properties editing side-panel */}
        <NodeEditPanel
          selectedNode={selectedNode}
          onUpdateNode={handleUpdateNode}
          onClose={() => setSelectedNodeId(null)}
        />
      </div>
    </div>
  );
}

export default function MindMapEditor() {
  return (
    <ReactFlowProvider>
      <FlowEditorInner />
    </ReactFlowProvider>
  );
}
