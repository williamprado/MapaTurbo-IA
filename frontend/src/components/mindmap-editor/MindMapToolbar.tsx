
interface MindMapToolbarProps {
  hasChanges: boolean;
  onSave: () => void;
  onAutoLayout: () => void;
  onFitView: () => void;
  saving: boolean;
  onExportPng: () => void;
  onExportPdf: () => void;
  exportingPng: boolean;
  exportingPdf: boolean;
}

export default function MindMapToolbar({
  hasChanges,
  onSave,
  onAutoLayout,
  onFitView,
  saving,
  onExportPng,
  onExportPdf,
  exportingPng,
  exportingPdf,
}: MindMapToolbarProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-4 p-4 bg-slate-900 border-b border-slate-800">
      <div className="flex items-center gap-3">
        {/* Unsaved Changes Indicator */}
        {hasChanges ? (
          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-950/60 text-amber-400 border border-amber-500/20 animate-pulse">
            ⚠️ Alterações não salvas
          </span>
        ) : (
          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-green-950/40 text-green-400 border border-green-500/20">
            ✓ Mapa salvo
          </span>
        )}
      </div>

      <div className="flex items-center gap-2.5">
        {/* Editor controls */}
        <button
          onClick={onFitView}
          className="px-3 py-1.5 bg-slate-950 hover:bg-slate-800 border border-slate-800 text-[10px] font-bold rounded-lg transition-colors cursor-pointer"
        >
          👁️ Centralizar
        </button>
        <button
          onClick={onAutoLayout}
          className="px-3 py-1.5 bg-slate-950 hover:bg-slate-800 border border-slate-800 text-[10px] font-bold rounded-lg transition-colors cursor-pointer"
          title="Organiza todos os nós automaticamente em formato de árvore"
        >
          🌲 Auto Layout
        </button>

        {/* Export Actions */}
        <div className="flex items-center gap-1.5 border-l border-slate-800 pl-3">
          <button
            onClick={onExportPng}
            disabled={exportingPng}
            className="px-3 py-1.5 bg-slate-950 hover:bg-slate-800 border border-slate-800 text-[10px] font-bold rounded-lg transition-colors cursor-pointer text-slate-300 disabled:opacity-40"
            title="Exportar mapa mental completo como imagem PNG"
          >
            {exportingPng ? 'Exportando PNG...' : '🖼️ Exportar PNG'}
          </button>
          <button
            onClick={onExportPdf}
            disabled={exportingPdf}
            className="px-3 py-1.5 bg-slate-950 hover:bg-slate-800 border border-slate-800 text-[10px] font-bold rounded-lg transition-colors cursor-pointer text-slate-300 disabled:opacity-40"
            title="Exportar mapa mental completo como documento PDF"
          >
            {exportingPdf ? 'Exportando PDF...' : '📄 Exportar PDF'}
          </button>
        </div>

        {/* Main Save Action */}
        <button
          onClick={onSave}
          disabled={saving || !hasChanges}
          className="px-4 py-1.5 bg-purple-600 hover:bg-purple-700 text-slate-100 disabled:opacity-40 font-bold text-[10px] rounded-lg transition-colors cursor-pointer shadow-lg shadow-purple-500/10"
        >
          {saving ? 'Salvando...' : 'Salvar Mapa'}
        </button>
      </div>
    </div>
  );
}
