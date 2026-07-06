import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../../services/api';
import { useAuthStore } from '../../stores/auth';

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const navigate = useNavigate();
  const setAuth = useAuthStore((state) => state.setAuth);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const response = await api.post('/auth/login', { email, password });
      const { access_token, refresh_token, user } = response.data.data;

      // Fetch user profile to get organizations details
      const profileResponse = await api.get('/auth/me', {
        headers: {
          Authorization: `Bearer ${access_token}`,
        },
      });

      const { organizations } = profileResponse.data.data;

      setAuth(access_token, refresh_token, user, organizations);

      // Redirect depending on global role
      if (user.global_role === 'SUPER_ADMIN') {
        navigate('/admin');
      } else {
        navigate('/app');
      }
    } catch (err: any) {
      if (err.response && err.response.data && err.response.data.message) {
        setError(err.response.data.message);
      } else {
        setError('Ocorreu um erro ao fazer login. Tente novamente.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-white font-sans flex items-center justify-center px-6 selection:bg-purple-600">
      <div className="max-w-md w-full p-8 rounded-2xl bg-slate-900 border border-slate-800 shadow-2xl relative">
        <div className="text-center mb-8">
          <Link to="/" className="inline-flex items-center gap-3 mb-6">
            <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-purple-600 to-indigo-600 flex items-center justify-center font-bold text-white shadow-lg">
              M
            </div>
            <span className="font-extrabold text-xl tracking-tight bg-gradient-to-r from-white to-purple-400 bg-clip-text text-transparent">
              MapaTurbo <span className="text-purple-500">IA</span>
            </span>
          </Link>
          <h2 className="text-2xl font-bold mb-2">Bem-vindo de volta</h2>
          <p className="text-slate-400 text-sm">Entre com suas credenciais para acessar o painel</p>
        </div>

        {error && (
          <div className="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2">
              E-mail
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full bg-slate-950 border border-slate-850 focus:border-purple-600 rounded-xl px-4 py-3 text-sm focus:outline-none transition-all"
              placeholder="seu@email.com"
              required
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-slate-400 mb-2">
              Senha
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full bg-slate-950 border border-slate-850 focus:border-purple-600 rounded-xl px-4 py-3 text-sm focus:outline-none transition-all"
              placeholder="••••••••"
              required
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-purple-600 hover:bg-purple-500 disabled:bg-purple-750 text-white font-semibold py-3 rounded-xl transition-all shadow-lg shadow-purple-500/20 hover:scale-[1.01] active:scale-[0.99] cursor-pointer"
          >
            {loading ? 'Acessando...' : 'Entrar'}
          </button>
        </form>

        <p className="mt-8 text-center text-sm text-slate-400">
          Não tem uma conta?{' '}
          <Link to="/cadastro" className="text-purple-400 hover:text-purple-300 font-semibold transition-colors">
            Cadastre-se grátis
          </Link>
        </p>
      </div>
    </div>
  );
}
