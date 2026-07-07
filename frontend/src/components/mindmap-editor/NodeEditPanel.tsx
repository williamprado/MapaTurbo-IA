import { useEffect, useState } from 'react';
import type { AINode } from './tree-utils';

interface NodeEditPanelProps {
  selectedNode: AINode | null;
  onUpdateNode: (id: string, updates: Partial<AINode>) => void;
  onClose: () => void;
}

export default function NodeEditPanel({ selectedNode, onUpdateNode, onClose }: NodeEditPanelProps) {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');

  useEffect(() => {
    if (selectedNode) {
      setTitle(selectedNode.title);
      setContent(selectedNode.content || '');
    }
  }, [selectedNode]);

  if (!selectedNode) {
    return (
      <div className="w-80 border-l border-slate-800 bg-slate-900/60 p-6 flex flex-col justify-center items-center text-center">
        <span className="text-3xl mb-2 text-slate-700">✏️</span>
        <p className="text-xs text-slate-500">Selecione um tópico no mapa para editar seus detalhes.</p>
      </div>
    );
  }

  const handleTitleChange = (val: string) => {
    setTitle(val);
    onUpdateNode(selectedNode.id, { title: val });
  };

  const handleContentChange = (val: string) => {
    setContent(val);
    onUpdateNode(selectedNode.id, { content: val });
  };

  return (
    <div className="w-80 border-l border-slate-800 bg-slate-900 p-6 flex flex-col justify-between h-full space-y-6">
      <div className="space-y-4 flex-1 overflow-y-auto">
        <div className="flex justify-between items-center border-b border-slate-800 pb-3">
          <h3 className="font-bold text-slate-200 text-xs">Propriedades do Nó</h3>
          <button
            onClick={onClose}
            className="text-[10px] text-slate-500 hover:text-slate-350 cursor-pointer"
          >
            Fechar
          </button>
        </div>

        <div>
          <span className="px-1.5 py-0.5 text-[8px] font-bold rounded uppercase bg-slate-800 text-slate-400">
            ID: {selectedNode.id}
          </span>
        </div>

        <div className="space-y-1">
          <label className="block text-[10px] font-semibold text-slate-500 uppercase tracking-wide">Título</label>
          <input
            type="text"
            value={title}
            onChange={(e) => handleTitleChange(e.target.value)}
            className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-250 focus:outline-none focus:border-purple-600 font-bold"
            maxLength={150}
          />
        </div>

        <div className="space-y-1">
          <label className="block text-[10px] font-semibold text-slate-500 uppercase tracking-wide">Conteúdo Explicativo</label>
          <textarea
            value={content}
            onChange={(e) => handleContentChange(e.target.value)}
            rows={8}
            className="w-full px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-slate-300 focus:outline-none focus:border-purple-600 font-sans leading-relaxed"
            maxLength={2000}
          />
        </div>
      </div>

      <div className="pt-4 border-t border-slate-800/80 text-[10px] text-slate-500 space-y-1">
        <p>As alterações são aplicadas localmente.</p>
        <p>Clique em <strong>Salvar mapa</strong> para persistir no banco.</p>
      </div>
    </div>
  );
}
