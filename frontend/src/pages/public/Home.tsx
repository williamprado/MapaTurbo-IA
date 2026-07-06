import { Link } from 'react-router-dom';

export default function Home() {
  return (
    <div className="min-h-screen bg-slate-950 text-white font-sans selection:bg-purple-600 selection:text-white">
      {/* Header */}
      <header className="border-b border-slate-800/80 backdrop-blur bg-slate-950/80 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-purple-600 to-indigo-600 flex items-center justify-center font-bold text-white shadow-lg shadow-purple-500/20">
              M
            </div>
            <span className="font-extrabold text-xl tracking-tight bg-gradient-to-r from-white via-slate-100 to-purple-400 bg-clip-text text-transparent">
              MapaTurbo <span className="text-purple-500">IA</span>
            </span>
          </div>

          <nav className="hidden md:flex items-center gap-8 text-sm font-medium text-slate-300">
            <a href="#features" className="hover:text-purple-400 transition-colors">Recursos</a>
            <a href="#how-it-works" className="hover:text-purple-400 transition-colors">Como Funciona</a>
            <Link to="/precos" className="hover:text-purple-400 transition-colors">Preços</Link>
          </nav>

          <div className="flex items-center gap-4">
            <Link to="/login" className="text-sm font-medium text-slate-300 hover:text-white transition-colors">
              Entrar
            </Link>
            <Link
              to="/cadastro"
              className="bg-purple-600 hover:bg-purple-500 text-white px-4 py-2 rounded-lg text-sm font-medium transition-all shadow-lg shadow-purple-500/20 hover:scale-[1.02]"
            >
              Criar Conta Grátis
            </Link>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative py-24 md:py-32 overflow-hidden">
        {/* Glow Effects */}
        <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[500px] h-[500px] bg-purple-600/10 rounded-full blur-[120px] pointer-events-none" />
        <div className="absolute top-1/3 left-1/3 w-[300px] h-[300px] bg-indigo-600/10 rounded-full blur-[100px] pointer-events-none" />

        <div className="max-w-5xl mx-auto px-6 text-center relative z-10">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-slate-900 border border-slate-800 text-xs text-purple-400 font-medium mb-8">
            <span className="flex h-2 w-2 rounded-full bg-purple-500 animate-pulse" />
            Nova Geração de Estudos com IA
          </div>

          <h1 className="text-4xl md:text-6xl font-black tracking-tight leading-[1.15] mb-8 bg-gradient-to-b from-white via-slate-100 to-slate-400 bg-clip-text text-transparent">
            Transforme PDFs, textos, aulas e links em <span className="bg-gradient-to-r from-purple-400 to-indigo-400 bg-clip-text text-transparent">mapas mentais</span> com IA em segundos.
          </h1>

          <p className="text-lg text-slate-400 max-w-2xl mx-auto mb-10 leading-relaxed">
            Esqueça resumos cansativos. Nossa Inteligência Artificial lê seus materiais e gera automaticamente mapas estruturados, flashcards dinâmicos e cronogramas inteligentes.
          </p>

          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              to="/cadastro"
              className="w-full sm:w-auto bg-purple-600 hover:bg-purple-500 text-white px-8 py-4 rounded-xl font-semibold transition-all shadow-xl shadow-purple-500/25 hover:scale-[1.02]"
            >
              Começar Agora
            </Link>
            <Link
              to="/precos"
              className="w-full sm:w-auto bg-slate-900 hover:bg-slate-850 border border-slate-800 text-slate-200 px-8 py-4 rounded-xl font-semibold transition-all hover:border-slate-700"
            >
              Ver Planos
            </Link>
          </div>
        </div>
      </section>

      {/* Feature Section */}
      <section id="features" className="py-24 border-t border-slate-900 bg-slate-950/50">
        <div className="max-w-7xl mx-auto px-6">
          <div className="text-center max-w-xl mx-auto mb-16">
            <h2 className="text-3xl font-bold mb-4">Gerencie seu aprendizado em velocidade turbo</h2>
            <p className="text-slate-400">Tudo o que você precisa para dominar qualquer conteúdo e passar em provas, concursos ou acelerar no trabalho.</p>
          </div>

          <div className="grid md:grid-cols-3 gap-8">
            {/* Card 1 */}
            <div className="p-8 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-purple-500/35 transition-all group">
              <div className="h-12 w-12 rounded-xl bg-purple-600/10 text-purple-400 flex items-center justify-center font-bold text-lg mb-6 group-hover:bg-purple-600 group-hover:text-white transition-all">
                ✏️
              </div>
              <h3 className="text-xl font-bold mb-3 text-slate-100">Geração Inteligente</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                Digite um tema ou cole textos longos e veja a IA criar um mapa completo com tópicos perfeitamente estruturados e explicados.
              </p>
            </div>

            {/* Card 2 */}
            <div className="p-8 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-purple-500/35 transition-all group">
              <div className="h-12 w-12 rounded-xl bg-purple-600/10 text-purple-400 flex items-center justify-center font-bold text-lg mb-6 group-hover:bg-purple-600 group-hover:text-white transition-all">
                📂
              </div>
              <h3 className="text-xl font-bold mb-3 text-slate-100">Upload de Arquivos</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                Envie PDFs, livros, anotações ou slides. Nossa IA processa e extrai o núcleo de conhecimento do documento em segundos.
              </p>
            </div>

            {/* Card 3 */}
            <div className="p-8 rounded-2xl bg-slate-900/60 border border-slate-800 hover:border-purple-500/35 transition-all group">
              <div className="h-12 w-12 rounded-xl bg-purple-600/10 text-purple-400 flex items-center justify-center font-bold text-lg mb-6 group-hover:bg-purple-600 group-hover:text-white transition-all">
                ⚡
              </div>
              <h3 className="text-xl font-bold mb-3 text-slate-100">Flashcards e Revisão</h3>
              <p className="text-slate-400 text-sm leading-relaxed">
                Crie cartões de memorização baseados nos mapas gerados e estude com o sistema de repetição espaçada integrado (SRS).
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-slate-900 py-12 text-slate-500 text-sm">
        <div className="max-w-7xl mx-auto px-6 flex flex-col md:flex-row items-center justify-between gap-6">
          <div>
            &copy; {new Date().getFullYear()} MapaTurbo IA. Todos os direitos reservados.
          </div>
          <div className="flex gap-6">
            <a href="#" className="hover:text-slate-300">Termos de Uso</a>
            <a href="#" className="hover:text-slate-300">Política de Privacidade</a>
          </div>
        </div>
      </footer>
    </div>
  );
}
