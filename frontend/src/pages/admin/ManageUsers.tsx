import { useState, useEffect } from 'react';
import { api } from '../../services/api';

interface User {
  id: string;
  email: string;
  name: string;
  global_role: string;
  status: string;
  created_at: string;
}

export default function ManageUsers() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const res = await api.get('/admin/users');
      setUsers(res.data.data.users || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-bold text-slate-100 mb-1">Gerenciamento de Usuários</h3>
        <p className="text-xs text-slate-400">Gerencie permissões globais e visualize todos os usuários cadastrados.</p>
      </div>

      {loading ? (
        <p className="text-xs text-slate-500 py-4">Carregando usuários...</p>
      ) : users.length === 0 ? (
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-8 text-center text-slate-500 text-xs">
          Nenhum usuário cadastrado.
        </div>
      ) : (
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <table className="w-full border-collapse text-left text-xs text-slate-300">
            <thead className="bg-slate-950 text-slate-400 uppercase font-semibold text-[10px] border-b border-slate-800">
              <tr>
                <th className="p-4">Nome</th>
                <th className="p-4">E-mail</th>
                <th className="p-4">Cargo Global</th>
                <th className="p-4">Status</th>
                <th className="p-4">Data de Cadastro</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {users.map((u) => (
                <tr key={u.id} className="hover:bg-slate-850/50 transition-colors">
                  <td className="p-4 font-bold text-slate-200">{u.name}</td>
                  <td className="p-4 text-slate-400">{u.email}</td>
                  <td className="p-4">
                    <span className={`text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded ${
                      u.global_role === 'SUPER_ADMIN'
                        ? 'bg-red-950 text-red-400 border border-red-500/20'
                        : 'bg-slate-950 text-slate-400 border border-slate-800'
                    }`}>
                      {u.global_role}
                    </span>
                  </td>
                  <td className="p-4">
                    <span className={`text-[9px] uppercase font-bold tracking-wider px-2 py-0.5 rounded ${
                      u.status === 'ACTIVE'
                        ? 'bg-green-950 text-green-400 border border-green-500/20'
                        : 'bg-red-950 text-red-400 border border-red-500/20'
                    }`}>
                      {u.status}
                    </span>
                  </td>
                  <td className="p-4 text-slate-500">{new Date(u.created_at).toLocaleDateString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
