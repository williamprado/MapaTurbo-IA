import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../services/api';

interface Plan {
  id: string;
  name: string;
  description: string;
  price_monthly: string;
  price_yearly: string;
  currency: string;
  credits_monthly: number;
  max_maps: number;
  max_files: number;
  max_users: number;
  max_storage_bytes: number;
  features: Record<string, any>;
}

export default function Prices() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);

  const fallbackPlans: Plan[] = [
    {
      id: 'free',
      name: 'Grátis',
      description: 'Perfeito para experimentar as potencialidades da plataforma.',
      price_monthly: '0.00',
      price_yearly: '0.00',
      currency: 'BRL',
      credits_monthly: 10,
      max_maps: 3,
      max_files: 3,
      max_users: 1,
      max_storage_bytes: 0,
      features: {
        generateTopic: true,
        generateText: true,
      }
    },
    {
      id: 'student',
      name: 'Estudante',
      description: 'Ideal para alunos, concurseiros e acadêmicos acelerarem.',
      price_monthly: '19.90',
      price_yearly: '199.00',
      currency: 'BRL',
      credits_monthly: 500,
      max_maps: 1000,
      max_files: 10,
      max_users: 1,
      max_storage_bytes: 50 * 1024 * 1024,
      features: {
        generateTopic: true,
        generateText: true,
        uploadPdf: true,
        generatePdf: true,
        exportPng: true,
      }
    },
    {
      id: 'pro',
      name: 'Turbo PRO',
      description: 'Para profissionais e empresas que buscam o máximo desempenho.',
      price_monthly: '49.90',
      price_yearly: '499.00',
      currency: 'BRL',
      credits_monthly: 2000,
      max_maps: 1000,
      max_files: 100,
      max_users: 5,
      max_storage_bytes: 500 * 1024 * 1024,
      features: {
        generateTopic: true,
        generateText: true,
        uploadPdf: true,
        generatePdf: true,
        exportPng: true,
        exportPdf: true,
      }
    }
  ];

  useEffect(() => {
    async function loadPlans() {
      try {
        const response = await api.get('/plans/public');
        if (response.data && response.data.data && response.data.data.length > 0) {
          setPlans(response.data.data);
        } else {
          setPlans(fallbackPlans);
        }
      } catch (err) {
        console.error('Failed to load plans from backend, using fallback:', err);
        setPlans(fallbackPlans);
      } finally {
        setLoading(false);
      }
    }
    loadPlans();
  }, []);

  const getPlanFeatures = (p: Plan) => {
    const list = [];
    list.push(p.max_maps >= 1000 || p.max_maps <= 0 ? 'Mapas mentais ilimitados' : `Até ${p.max_maps} mapas mentais`);
    list.push(`${p.credits_monthly} créditos / mês`);
    list.push(p.max_users <= 1 ? '1 membro individual' : `Até ${p.max_users} membros na equipe`);
    
    if (p.max_files <= 0) {
      list.push('Sem upload de arquivos');
    } else {
      list.push(`Até ${p.max_files} arquivos de upload`);
    }

    if (p.features?.generatePdf) {
      list.push('Geração de mapas por PDF (RAG)');
    }

    if (p.features?.exportPng && p.features?.exportPdf) {
      list.push('Exportação PNG e PDF inclusa');
    } else if (p.features?.exportPng) {
      list.push('Exportação PNG inclusa');
    } else {
      list.push('Sem exportações externas');
    }

    return list;
  };

  return (
    <div className="min-h-screen bg-slate-950 text-white font-sans selection:bg-purple-600 selection:text-white">
      {/* Header */}
      <header className="border-b border-slate-800/80 backdrop-blur bg-slate-950/80 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-purple-600 to-indigo-600 flex items-center justify-center font-bold text-white shadow-lg shadow-purple-500/20">
              M
            </div>
            <span className="font-extrabold text-xl tracking-tight bg-gradient-to-r from-white via-slate-100 to-purple-400 bg-clip-text text-transparent">
              MapaTurbo <span className="text-purple-500">IA</span>
            </span>
          </Link>

          <div className="flex items-center gap-4">
            <Link to="/login" className="text-sm font-medium text-slate-300 hover:text-white transition-colors">
              Entrar
            </Link>
            <Link
              to="/cadastro"
              className="bg-purple-600 hover:bg-purple-500 text-white px-4 py-2 rounded-lg text-sm font-medium transition-all shadow-lg shadow-purple-500/20 hover:scale-[1.02]"
            >
              Criar Conta
            </Link>
          </div>
        </div>
      </header>

      {/* Pricing Section */}
      <section className="py-24 relative overflow-hidden">
        {/* Glow */}
        <div className="absolute top-1/3 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-purple-600/5 rounded-full blur-[120px] pointer-events-none" />

        <div className="max-w-6xl mx-auto px-6 relative z-10">
          <div className="text-center max-w-2xl mx-auto mb-20">
            <h1 className="text-4xl md:text-5xl font-black mb-6">Planos simples e transparentes</h1>
            <p className="text-slate-400 text-lg">Escolha o plano ideal para você e turbine sua capacidade de absorver e organizar informações.</p>
          </div>

          {loading ? (
            <div className="flex justify-center items-center py-12">
              <div className="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-purple-500"></div>
            </div>
          ) : (
            <div className="grid md:grid-cols-3 gap-8 items-stretch">
              {plans.map((p, idx) => {
                const priceNum = parseFloat(p.price_monthly) || 0;
                const highlight = p.name.toLowerCase().includes('estudante') || p.name.toLowerCase().includes('pro') && idx === 1;

                return (
                  <div
                    key={p.id || idx}
                    className={`p-8 rounded-2xl bg-slate-900/60 border flex flex-col justify-between transition-all ${
                      highlight
                        ? 'border-purple-500 ring-2 ring-purple-500/20 scale-[1.03] md:relative'
                        : 'border-slate-800 hover:border-slate-700'
                    }`}
                  >
                    {highlight && (
                      <span className="absolute top-0 right-1/2 translate-x-1/2 -translate-y-1/2 bg-purple-600 text-white text-[10px] uppercase font-bold tracking-widest px-3 py-1 rounded-full">
                        Recomendado
                      </span>
                    )}

                    <div>
                      <h3 className="text-xl font-bold text-slate-100 mb-2">{p.name}</h3>
                      <p className="text-slate-400 text-sm mb-6 leading-relaxed min-h-[40px]">{p.description}</p>

                      <div className="flex items-baseline gap-2 mb-6 border-b border-slate-800 pb-6">
                        <span className="text-4xl font-extrabold">
                          {priceNum === 0 ? 'R$ 0' : `R$ ${priceNum.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`}
                        </span>
                        <span className="text-slate-400 text-sm">/ mês</span>
                      </div>

                      <ul className="space-y-4 mb-8 text-sm text-slate-300">
                        {getPlanFeatures(p).map((f, fIdx) => (
                          <li key={fIdx} className="flex items-center gap-3">
                            <span className="text-purple-500">✓</span>
                            {f}
                          </li>
                        ))}
                      </ul>
                    </div>

                    <Link
                      to="/cadastro"
                      className={`w-full text-center py-3 rounded-xl font-semibold transition-all ${
                        highlight
                          ? 'bg-purple-600 hover:bg-purple-500 text-white shadow-xl shadow-purple-500/25'
                          : 'bg-slate-800 hover:bg-slate-750 border border-slate-750 text-slate-200'
                      }`}
                    >
                      Começar Agora
                    </Link>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
