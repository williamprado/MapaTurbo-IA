import { Handle, Position } from '@xyflow/react';

interface MindMapNodeProps {
  id: string;
  data: {
    title: string;
    content: string;
    level: number;
    isRoot: boolean;
    onAddChild: (id: string) => void;
    onDelete: (id: string) => void;
  };
  selected?: boolean;
}

export default function MindMapNode({ id, data, selected }: MindMapNodeProps) {
  const { title, content, level, isRoot, onAddChild, onDelete } = data;

  // Level specific border and background themes
  const levelStyles = () => {
    if (isRoot) {
      return 'border-purple-500 bg-purple-950/40 text-slate-100 shadow-purple-500/10 shadow-lg';
    }
    switch (level) {
      case 1:
        return 'border-indigo-500/80 bg-slate-900/90';
      case 2:
        return 'border-sky-500/60 bg-slate-900/90';
      default:
        return 'border-slate-800 bg-slate-900/90';
    }
  };

  return (
    <div
      className={`px-4 py-3 rounded-xl border-2 ${levelStyles()} ${
        selected ? 'ring-2 ring-purple-600 border-transparent scale-105' : ''
      } transition-all duration-200 w-[260px] text-left relative group select-none`}
    >
      {/* Handles */}
      {!isRoot && (
        <Handle
          type="target"
          position={Position.Left}
          className="w-2.5 h-2.5 !bg-purple-600 border-2 !border-slate-950"
        />
      )}
      <Handle
        type="source"
        position={Position.Right}
        className="w-2.5 h-2.5 !bg-purple-600 border-2 !border-slate-950"
      />

      {/* Level Tag */}
      <div className="flex justify-between items-center mb-1">
        <span className={`px-1.5 py-0.5 text-[8px] font-bold rounded uppercase ${
          isRoot ? 'bg-purple-900 text-purple-200' : 'bg-slate-850 text-slate-400'
        }`}>
          {isRoot ? 'Tópico Central' : `Nível ${level}`}
        </span>

        {/* Action Buttons visible on hover */}
        <div className="flex items-center gap-1.5 opacity-40 group-hover:opacity-100 transition-opacity">
          <button
            onClick={(e) => {
              e.stopPropagation();
              onAddChild(id);
            }}
            title="Adicionar nó filho"
            className="w-5 h-5 rounded bg-slate-950 hover:bg-purple-900 text-purple-400 hover:text-purple-200 flex items-center justify-center text-xs font-bold cursor-pointer transition-colors"
          >
            +
          </button>
          {!isRoot && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onDelete(id);
              }}
              title="Excluir nó e descendentes"
              className="w-5 h-5 rounded bg-slate-950 hover:bg-red-950 text-red-400 hover:text-red-200 flex items-center justify-center text-[10px] cursor-pointer transition-colors"
            >
              🗑️
            </button>
          )}
        </div>
      </div>

      {/* Node Content */}
      <h4 className="font-bold text-slate-100 text-xs truncate mb-1 pr-4">{title}</h4>
      <p className="text-slate-400 text-[10px] line-clamp-3 leading-relaxed font-sans">{content || '(Sem descrição)'}</p>
    </div>
  );
}
