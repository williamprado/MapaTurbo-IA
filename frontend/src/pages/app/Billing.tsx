export default function Billing() {
  return (
    <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-6">
      <div>
        <h3 className="text-lg font-bold text-slate-100 mb-1">Planos & Assinatura</h3>
        <p className="text-xs text-slate-400">Gerencie sua forma de cobrança e altere seu plano ativo.</p>
      </div>

      <div className="p-4 bg-slate-950 rounded-xl border border-slate-850 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <span className="bg-purple-950 text-purple-400 border border-purple-500/20 text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded">
            Plano Ativo
          </span>
          <h4 className="text-base font-bold mt-2">Free Trial (Testes)</h4>
          <p className="text-xs text-slate-500">100 créditos IA mensais inclusos gratuitamente.</p>
        </div>

        <button className="bg-purple-600 hover:bg-purple-500 text-white font-semibold text-xs px-4 py-2.5 rounded-lg transition-colors cursor-pointer">
          Fazer Upgrade
        </button>
      </div>

      <div>
        <h4 className="text-sm font-bold text-slate-200 mb-3">Histórico de Faturas</h4>
        <div className="bg-slate-950 border border-slate-850 rounded-xl p-6 text-center text-slate-500 text-xs">
          Nenhuma fatura registrada neste workspace.
        </div>
      </div>
    </div>
  );
}
