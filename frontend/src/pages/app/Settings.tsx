import { useAuthStore } from '../../stores/auth';

export default function Settings() {
  const { user } = useAuthStore();

  return (
    <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-6">
      <div>
        <h3 className="text-lg font-bold text-slate-100 mb-1">Configurações da Conta</h3>
        <p className="text-xs text-slate-400">Atualize seus dados pessoais de perfil e segurança.</p>
      </div>

      <div className="space-y-4 max-w-md">
        <div>
          <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 mb-2">
            Nome Completo
          </label>
          <input
            type="text"
            value={user?.name || ''}
            readOnly
            className="w-full bg-slate-950 border border-slate-850 rounded-xl px-4 py-2.5 text-xs text-slate-400 focus:outline-none"
          />
        </div>

        <div>
          <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 mb-2">
            Endereço de E-mail
          </label>
          <input
            type="email"
            value={user?.email || ''}
            readOnly
            className="w-full bg-slate-950 border border-slate-850 rounded-xl px-4 py-2.5 text-xs text-slate-400 focus:outline-none"
          />
        </div>

        <div>
          <label className="block text-xs font-semibold uppercase tracking-wider text-slate-500 mb-2">
            Nível de Permissão Global
          </label>
          <span className="inline-block bg-slate-950 border border-slate-850 rounded-lg px-3 py-1.5 text-xs text-slate-300 font-bold">
            {user?.global_role}
          </span>
        </div>
      </div>
    </div>
  );
}
