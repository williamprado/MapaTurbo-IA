# Guia de Validação Runtime: MapaTurbo IA

Este guia descreve os passos para rodar e validar operacionalmente o projeto **MapaTurbo IA** assim que um motor de contêineres (**Docker Desktop**, **Podman** ou **WSL**) estiver ativo no sistema host.

---

## 📋 Pré-requisitos
1. **Docker Desktop** (com suporte a Docker Compose v2) ou **Podman CLI**.
2. **WSL 2** instalado (se estiver rodando Docker no Windows sem Docker Desktop).
3. **Go** (v1.22+) instalado localmente para execução dos binários da API/Worker sem contêiner (opcional).
4. **Node.js** (v20+) instalado localmente para execução do frontend.

---

## 🛠️ Passo a Passo Operacional

### 1. Inicializando a Stack de Infraestrutura
Navegue até a raiz do projeto e crie o arquivo `.env` a partir do modelo:
```bash
cp .env.example .env
```
Inicie os serviços do banco (Postgre + pgvector), Redis e MinIO:
```bash
docker compose -f docker/docker-compose.yml up -d
```
Valide que todos os contêineres subiram com o status `running`:
```bash
docker compose -f docker/docker-compose.yml ps
```
Os seguintes contêineres devem estar ativos:
* `mapaturbo_postgres` (porta `5432`)
* `mapaturbo_redis` (porta `6379`)
* `mapaturbo_minio` (porta `9000` / `9001`)

---

## 🧪 Roteiro de Testes e Validações

### 1. Validação de Extensões do Banco (PostgreSQL)
Conecte-se ao Postgres (via psql, DBeaver ou similar):
```bash
docker exec -it mapaturbo_postgres psql -U mapaturbo -d mapaturbo
```
Rode as queries abaixo para confirmar o funcionamento do `pgvector` e `pgcrypto`:
```sql
SELECT * FROM pg_extension WHERE extname = 'vector';
SELECT * FROM pg_extension WHERE extname = 'pgcrypto';
\dt -- Lista todas as tabelas
```

### 2. Validação da Autenticação
* **Fazer Login (Admin)**:
  ```bash
  curl -X POST http://localhost:8080/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email": "admin@admin.com", "password": "@Admin2328"}'
  ```
  O retorno deve conter um `accessToken` de vida curta, um `refreshToken` longo e a estrutura `user` sem expor segredos.
* **Renovar Token (Refresh)**:
  ```bash
  curl -X POST http://localhost:8080/auth/refresh \
    -H "Content-Type: application/json" \
    -d '{"refresh_token": "<SEU_REFRESH_TOKEN>"}'
  ```
* **Consultar Perfil (Me)**:
  ```bash
  curl -X GET http://localhost:8080/auth/me \
    -H "Authorization: Bearer <SEU_ACCESS_TOKEN>"
  ```

### 3. Validação das Proteções de Admin
Tente acessar as rotas abaixo com e sem o token `SUPER_ADMIN`:
```bash
curl -X GET http://localhost:8080/admin/organizations
curl -X GET http://localhost:8080/admin/users
curl -X GET http://localhost:8080/admin/plans
```
* **Sem Token**: Deve retornar `401 Unauthorized`.
* **Token de Usuário Comum**: Deve retornar `403 Forbidden`.
* **Token Super Admin**: Deve retornar status `200 OK`.

### 4. Validação do CRUD de Empresas e Planos
* **Criar Empresa**:
  ```bash
  curl -X POST http://localhost:8080/admin/organizations \
    -H "Authorization: Bearer <TOKEN_ADMIN>" \
    -H "Content-Type: application/json" \
    -d '{"name": "Nova Empresa", "slug": "nova-empresa"}'
  ```
* **Criar Plano**:
  ```bash
  curl -X POST http://localhost:8080/admin/plans \
    -H "Authorization: Bearer <TOKEN_ADMIN>" \
    -H "Content-Type: application/json" \
    -d '{"name": "Plan Turbo", "price_monthly": 29.90, "currency": "BRL", "is_public": true}'
  ```

### 5. Validação de Armazenamento (MinIO)
Acesse o console em http://localhost:9001 (User: `mapaturbo`, Pass: `mapaturbo_password`) e confirme se o bucket `mapaturbo-files` foi auto-provisionado com sucesso.

### 6. Validação do Pipeline de Geração com IA (Fase 3A)
* **Geração por Tema/Texto**:
  Rode o seguinte cURL para iniciar a geração:
  ```bash
  curl -X POST http://localhost:8080/mindmaps/generate \
    -H "Authorization: Bearer <TOKEN_USUARIO>" \
    -H "X-Organization-ID: <UUID_ORGANIZACAO>" \
    -H "Content-Type: application/json" \
    -d '{
      "type": "TOPIC",
      "title": "Mitose",
      "content": "Divisão celular biológica",
      "options": {
        "depth": 3,
        "language": "pt-BR",
        "style": "study"
      }
    }'
  ```
* **Acompanhar status do Job (Polling)**:
  ```bash
  curl -H "Authorization: Bearer <TOKEN>" -H "X-Organization-ID: <ORG>" http://localhost:8080/generation-jobs/<jobId>
  ```

### 7. Validação de Faturamento & Webhooks Asaas
Consulte o guia detalhado em `docs/payments-validation.md` para testar os endpoints de faturamento localmente simulando as requisições de webhook do Asaas.

### 8. Validação do Editor de Mapas Mentais (Fase 3B)
* **Salvar alterações do canvas (PATCH)**:
  ```bash
  curl -X PATCH http://localhost:8080/mindmaps/<MINDMAP_ID> \
    -H "Authorization: Bearer <TOKEN>" \
    -H "X-Organization-ID: <ORG>" \
    -H "Content-Type: application/json" \
    -d '{
      "jsonData": {
        "title": "Mitose",
        "centralTopic": "Mitose",
        "summary": "Divisão celular",
        "nodes": [
          {"id": "root", "parentId": null, "title": "Mitose", "content": "Divisão celular", "level": 0, "order": 0, "position": {"x": 100, "y": 300}},
          {"id": "node_1", "parentId": "root", "title": "Prófase", "content": "Condensação", "level": 1, "order": 0, "position": {"x": 420, "y": 300}}
        ],
        "edges": [
          {"id": "edge-root-node_1", "source": "root", "target": "node_1"}
        ]
      }
    }'
  ```
  O backend valida a existência dos nós apontados por `edges`, a presença única do nó `root`, a ausência de ciclos e limites de quantidade.

