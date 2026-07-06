# MapaTurbo IA

Plataforma SaaS para geração e gerenciamento de mapas mentais com Inteligência Artificial.

## Requisitos
* Docker & Docker Compose
* Go 1.22+ (para desenvolvimento local sem Docker)
* Node.js 20+ & npm (para desenvolvimento local sem Docker)

## Estrutura do Projeto
* `frontend/`: React + Vite + TypeScript (Interface do usuário)
* `backend/`: Go (API Gin + Worker Asynq)
* `docker/`: Configurações de serviços locais (PostgreSQL, Redis, MinIO)

## Como Rodar Localmente (Docker)

1. Clone o repositório.
2. Copie o arquivo `.env.example` para `.env`:
   ```bash
   cp .env.example .env
   ```
3. Preencha as chaves necessárias no arquivo `.env` (ex: `JWT_SECRET`, `ENCRYPTION_KEY`, chaves de IA).
4. Suba a stack com o Docker Compose:
   ```bash
   docker compose -f docker/docker-compose.yml up --build -d
   ```
5. Acesse:
   * Frontend: http://localhost:5173
   * API Backend: http://localhost:8080/health
   * Painel MinIO: http://localhost:9001 (User: `mapaturbo`, Pass: `mapaturbo_password`)

## Migrations (Goose) & SQL (sqlc)
Dentro da pasta `backend`:
* Aplicar migrations manualmente: `go run cmd/migrate/main.go`
* Gerar código de banco tipado (sqlc): `sqlc generate` (requer CLI do sqlc instalada)
