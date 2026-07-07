# Homologação e Validação do MVP Comercial — MapaTurbo IA

Este documento compila a lista de recursos que integram o MVP Comercial do **MapaTurbo IA** e discrimina o status de homologação de cada um deles, divididos em **Validação Estática** (concluída no ambiente local de build) e **Validação Runtime** (que requer Docker/WSL ativo).

---

## 📋 Resumo do Status Global

* **Validação Estática (Backend & Frontend)**: **100% Concluída** (Builds de produção, tipagens TypeScript, estruturas SQLC e testes locais sem falhas).
* **Validação Runtime (Contêineres e Integrações Reais)**: **PENDENTE** (Aguardando ativação do motor de contêineres Docker/WSL no host local).

---

## 🎯 Checklist Funcional do MVP

### 1. Autenticação & Níveis de Acesso
* [x] **Validação Estática**:
  * Estrutura de tokens de autenticação (JWT) e refresh token implementados com assinatura criptográfica.
  * Middleware de autorização para isolamento por tenant (`organization_id`) e nível administrativo (`SUPER_ADMIN`).
  * Atualização e higienização do perfil na rota `/auth/me`.
* [ ] **Validação Runtime Real**:
  * Teste real de login, expiração de JWT, renovação com refresh token silencioso via interceptor Axios no frontend, e persistência do status `last_login_at` no banco Postgres.

### 2. Multiempresa & Gestão Comercial
* [x] **Validação Estática**:
  * Isolamento rígido de tenant em todas as queries SQL do backend.
  * Lógica para cadastrar empresas e gerenciar planos via painel administrativo.
  * Histórico de créditos com transações financeiras no backend e frontend.
* [ ] **Validação Runtime Real**:
  * Simulação de concorrência com múltiplos usuários pertencentes a diferentes empresas para atestar que os dados de mapas, uploads, créditos e faturamento nunca vazam de um tenant para outro.

### 3. Integração com IA (OpenAI)
* [x] **Validação Estática**:
  * Conector estruturado para chamadas de completação de chat (OpenAI JSON Mode) e geração de embeddings (`text-embedding-3-small` - 1536 dimensões).
  * Painel Super Admin para controle, ativação e encriptação AES-256 de credenciais de provedores de IA.
  * Chamada de IA encapsulada em Workers assíncronos gerenciados via fila do Redis/Asynq.
* [ ] **Validação Runtime Real**:
  * Realização de teste de conexão com OpenAI Sandbox/Real.
  * Execução assíncrona do job de geração de mapa por tema/texto a partir do dashboard do usuário.

### 4. Gestão de Faturamento (Asaas Gateway)
* [x] **Validação Estática**:
  * Client do Asaas com suporte a checkouts em Pix, Boleto e Cartão de Crédito.
  * Receiver de webhooks de pagamento com controle de idempotência via tabela `webhook_events`.
  * Criação automática de faturas (`invoices`), transações (`payment_transactions`) e ativação automática de assinaturas/créditos de IA pós-confirmação.
* [ ] **Validação Runtime Real**:
  * Simular eventos do Asaas no sandbox e receber requisições de webhook no endpoint `/webhooks/asaas` do backend para garantir que as assinaturas e créditos sejam ativados automaticamente sem duplicidade.

### 5. Upload PDF & RAG com PGVector
* [x] **Validação Estática**:
  * Lógica de fatiamento de texto (chunks) e armazenamento de arquivos no MinIO.
  * Queries vetoriais via operador de distância de cosseno `<=>` do PGVector.
  * Integração frontend para seleção de PDF e digitação de foco temático.
* [ ] **Validação Runtime Real**:
  * Enviar PDF com mais de 10MB no frontend, validar gravação física no bucket MinIO, acompanhar o processamento de OCR/texto pelo Worker, geração dos embeddings com OpenAI e inserção correta dos vetores na tabela Postgres.

### 6. Editor Visual (React Flow)
* [x] **Validação Estática**:
  * Componentes customizados, algoritmos de auto-layout em árvore e controle de viewport integrados.
  * Proteção no backend contra ciclicidade direcionada e exclusão acidental do nó root.
  * Integração de botões de exportação PNG/PDF protegidos.
* [ ] **Validação Runtime Real**:
  * Mover nós no canvas, atualizar viewport, salvar estado e testar se o backend valida e aceita as alterações do mapa de forma síncrona.
  * Baixar mapas exportados em PDF/PNG localmente.

### 7. Painéis e Dashboards
* [x] **Validação Estática**:
  * Telas administrativas com estatísticas, totalizadores e logs de falhas técnicas.
  * Visão agregada do tenant (/admin/organizations/:id/summary) dividida em abas (Overview, Members, Mapas, Uploads, Invoices, Audit Logs).
  * Onboarding com primeiros passos e progressão no dashboard do usuário comum.
* [ ] **Validação Runtime Real**:
  * Visualizar a carga de dados dinâmicos do banco nas telas administrativas e do usuário à medida que transações, faturas e mapas são criados.

---

## 🛠️ Procedimento Pendente para a Fase 4B-B
Assim que o Docker estiver operacional, a homologação completa deve ser iniciada com a execução do comando a seguir:
```bash
docker compose -f docker/docker-compose.yml up -d
```
E a execução passo a passo do roteiro contido em `docs/runtime-validation.md`.
