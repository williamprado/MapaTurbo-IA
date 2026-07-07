# MapaTurbo IA

Plataforma SaaS de alta performance para geração e gerenciamento de mapas mentais com Inteligência Artificial a partir de temas, textos, PDFs, URLs, vídeos do YouTube, áudios e imagens.

---

## 🚀 Requisitos de Sistema
* **Go** (v1.22 ou superior)
* **Node.js** (v20 ou superior) & **npm**
* **Docker & Docker Compose** (para orquestração local)

---

## 📂 Estrutura do Repositório
* [`/backend`](file:///I:/MapaTurbo%20IA/backend): API HTTP desenvolvida em Go (Gin Framework) e Worker assíncrono (Asynq + Redis).
* [`/frontend`](file:///I:/MapaTurbo%20IA/frontend): Interface do usuário moderna em React, Vite, TypeScript e TailwindCSS v4.
* [`/docker`](file:///I:/MapaTurbo%20IA/docker): Arquivos de compose locais e stacks produtivos para Docker Swarm.

---

## 🛠️ Instalação e Configuração

### 1. Variáveis de Ambiente
Copie o arquivo blueprint de ambiente na raiz do projeto:
```bash
cp .env.example .env
```
Edite o arquivo `.env` para ajustar segredos de segurança (`JWT_SECRET` e `ENCRYPTION_KEY` de 32 bytes) e inserir as suas chaves de provedor de IA (`OPENAI_API_KEY`, `GEMINI_API_KEY`).

### 2. Inicializando os Serviços Compartilhados (Docker)
Suba os contêineres de banco de dados, fila e armazenamento localizados no diretório `/docker`:
```bash
docker compose -f docker/docker-compose.yml up -d
```
Este comando inicializa:
* **PostgreSQL (com PGVector)** exposto na porta `5432`
* **Redis** (para filas Asynq) exposto na porta `6379`
* **MinIO Console** (Armazenamento S3) exposto na porta `9001` (painel) e `9000` (API)

---

## 💻 Executando Localmente (Sem Docker)

### Backend (API HTTP)
1. Acesse a pasta do backend:
   ```bash
   cd backend
   ```
2. Instale as dependências do Go:
   ```bash
   go mod download
   ```
3. Execute o servidor de desenvolvimento:
   ```bash
   go run cmd/api/main.go
   ```
   * *Nota*: Na inicialização, a API executa automaticamente todas as migrations Goose e executa o seed do Super Admin no banco.

### Backend Worker (Processamento de Filas)
1. Abra um terminal separado na pasta do backend:
   ```bash
   cd backend
   ```
2. Execute o Worker:
   ```bash
   go run cmd/worker/main.go
   ```

### Frontend (React App)
1. Acesse a pasta do frontend:
   ```bash
   cd frontend
   ```
2. Instale as dependências de pacotes:
   ```bash
   npm install
   ```
3. Inicie o servidor de desenvolvimento:
   ```bash
   npm run dev
   ```
   * A aplicação estará disponível em http://localhost:5173.

---

## 🗄️ Manutenção de Banco e Consultas (Goose & sqlc)

* **Gerar Códigos de Acesso a Banco (sqlc)**:
  Tendo a ferramenta `sqlc` instalada, execute de dentro de `/backend/db/`:
  ```bash
  go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
  ```
* **Status de Migrations (Goose)**:
  O Goose aplica as migrations automaticamente ao bootar o app, mas você pode verificar o status a partir de `/backend/` com:
  ```bash
  go run github.com/pressly/goose/v3/cmd/goose@latest -dir db/migrations postgres "postgres://mapaturbo:mapaturbo_password@localhost:5432/mapaturbo?sslmode=disable" status
  ```

---

## 🧪 Validando a Operação (Testes Rápido)

### 1. Health Check
```bash
curl -X GET http://localhost:8080/health
```
Resposta esperada:
```json
{
  "status": "OK",
  "time": "2026-07-06T..."
}
```

### 2. Acesso Super Admin (Credenciais do Seed)
O seed automático cria o primeiro administrador com os seguintes dados:
* **E-mail**: `admin@admin.com`
* **Senha**: `@Admin2328`
* **Nível Global**: `SUPER_ADMIN`
* **Status**: `ACTIVE`

### 3. Autenticação e Token JWT (Login)
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@admin.com", "password": "@Admin2328"}'
```
Resposta contendo o token `Bearer` e os dados básicos do usuário (excluindo qualquer hash de senha).

### 4. Perfil Protegido (GET /auth/me)
Use o token retornado no login:
```bash
curl -X GET http://localhost:8080/auth/me \
  -H "Authorization: Bearer <SEU_TOKEN_JWT>"
```
Retorna o perfil autenticado com cargo global `SUPER_ADMIN` e as organizações pertencentes.

---

## 📖 Documentação Detalhada
Para testar, validar e implantar cada uma das fases do projeto:
* **Validação Runtime**: [docs/runtime-validation.md](file:///I:/MapaTurbo%20IA/docs/runtime-validation.md)
* **Geração por IA & Créditos**: [docs/ai-generation-validation.md](file:///I:/MapaTurbo%20IA/docs/ai-generation-validation.md)
* **Editor Visual React Flow**: [docs/mindmap-editor-validation.md](file:///I:/MapaTurbo%20IA/docs/mindmap-editor-validation.md)
* **CI/CD & Docker Hub**: [docs/github-actions-dockerhub.md](file:///I:/MapaTurbo%20IA/docs/github-actions-dockerhub.md)
* **Deploy Docker Swarm / Portainer**: [docs/deploy-swarm.md](file:///I:/MapaTurbo%20IA/docs/deploy-swarm.md)
* **Checklist MVP Comercial**: [docs/mvp-validation.md](file:///I:/MapaTurbo%20IA/docs/mvp-validation.md)

