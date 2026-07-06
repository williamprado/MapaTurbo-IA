import { Link } from 'react-router-dom';

export default function Prices() {
  const plans = [
    {
      name: 'Grátis',
      price: 'R$ 0',
      period: 'para sempre',
      desc: 'Perfeito para experimentar as potencialidades da plataforma.',
      credits: '10 créditos/mês',
      features: [
        'Até 3 mapas mentais',
        'Geração por tema',
        'Geração por texto',
        'Sem upload de arquivos',
        'Sem flashcards avançados',
      ],
      cta: 'Começar Grátis',
      highlight: false,
    },
    {
      name: 'Estudante',
      price: 'R$ 19,90',
      period: 'por mês',
      desc: 'Ideal para alunos, concurseiros e acadêmicos acelerarem.',
      credits: '500 créditos/mês',
      features: [
        'Mapas mentais ilimitados',
        'Geração por tema e texto',
        'Upload de até 10 arquivos/mês',
        'Gerador de Flashcards',
        'Exportação básica para imagem',
      ],
      cta: 'Assinar Plano',
      highlight: true,
    },
    {
      name: 'Turbo PRO',
      price: 'R$ 49,90',
      period: 'por mês',
      desc: 'Para profissionais e empresas que buscam o máximo desempenho.',
      credits: '2000 créditos/mês',
      features: [
        'Tudo do plano Estudante',
        'Busca semântica avançada (RAG)',
        'Geração por URL e YouTube',
        'Exportação ilimitada (PDF/Imagem)',
        'Suporte prioritário e API',
      ],
      cta: 'Assinar Turbo',
      highlight: false,
    },
  ];

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

          <div className="grid md:grid-cols-3 gap-8 items-stretch">
            {plans.map((p, idx) => (
              <div
                key={idx}
                className={`p-8 rounded-2xl bg-slate-900/60 border flex flex-col justify-between transition-all ${
                  p.highlight
                    ? 'border-purple-500 ring-2 ring-purple-500/20 scale-[1.03] md:relative'
                    : 'border-slate-800 hover:border-slate-700'
                }`}
              >
                {p.highlight && (
                  <span className="absolute top-0 right-1/2 translate-x-1/2 -translate-y-1/2 bg-purple-600 text-white text-[10px] uppercase font-bold tracking-widest px-3 py-1 rounded-full">
                    Mais Popular
                  </span>
                )}

                <div>
                  <h3 className="text-xl font-bold text-slate-100 mb-2">{p.name}</h3>
                  <p className="text-slate-400 text-sm mb-6 leading-relaxed min-h-[40px]">{p.desc}</p>

                  <div className="flex items-baseline gap-2 mb-6 border-b border-slate-800 pb-6">
                    <span className="text-4xl font-extrabold">{p.price}</span>
                    <span className="text-slate-400 text-sm">{p.period}</span>
                  </div>

                  <div className="mb-6 font-semibold text-purple-400 text-sm">
                    {p.credits}
                  </div>

                  <ul className="space-y-4 mb-8 text-sm text-slate-300">
                    {p.features.map((f, fIdx) => (
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
                    p.highlight
                      ? 'bg-purple-600 hover:bg-purple-500 text-white shadow-xl shadow-purple-500/25'
                      : 'bg-slate-800 hover:bg-slate-750 border border-slate-750 text-slate-200'
                  }`}
                >
                  {p.cta}
                </Link>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}
