# Manual de Deploy: Docker Swarm & Portainer — MapaTurbo IA

Este documento orienta sobre a implantação, orquestração e gerenciamento da infraestrutura do **MapaTurbo IA** em ambiente de produção utilizando um cluster **Docker Swarm** com **Portainer** no ecossistema do **WA Painel** com roteamento reverso via **Traefik**.

---

## 📋 1. Pré-requisitos

* Cluster Docker Swarm ativo e integrado com o Portainer.
* **Traefik** implantado globalmente no cluster escutando a rede overlay de borda do WA Painel.
* Chave de API OpenAI e token/chave Asaas Sandbox ou Produção configurados.
* Rede overlay externa `wapainelnet` criada.
* Volumes externos do cluster provisionados:
  * `mapaturbo_postgres`
  * `mapaturbo_redis`
  * `mapaturbo_minio`

---

## 🌐 2. Rede Docker Overlay Externa

Todos os serviços compartilham a rede externa padrão do painel para viabilizar a comunicação com o Traefik e isolar o tráfego interno:
```bash
# Caso a rede ainda não exista, crie-a com:
docker network create --driver=overlay --attachable wapainelnet
```

---

## 💾 3. Volumes Externos do Cluster

Os volumes persistirão os dados estáticos em storage do cluster Swarm. Eles devem ser criados antes do deploy da stack:
```bash
docker volume create --name=mapaturbo_postgres
docker volume create --name=mapaturbo_redis
docker volume create --name=mapaturbo_minio
```

---

## 🔒 4. Variáveis de Ambiente & Secrets (Portainer)

Os segredos não devem ser expostos no arquivo `stack.yml`. Ao criar a Stack no Portainer, declare os seguintes valores em **Environment variables**:

```txt
DATABASE_URL=postgres://mapaturbo:TROQUE_SENHA_FORTE@mapaturbo_postgres:5432/mapaturbo?sslmode=disable
JWT_SECRET=TROQUE_JWT_SECRET_FORTE
ENCRYPTION_KEY=TROQUE_AES_CHAVE_32_BYTES_FORTE
MINIO_ROOT_USER=mapaturbo
MINIO_ROOT_PASSWORD=TROQUE_MINIO_PASSWORD
MINIO_BUCKET=mapaturbo-files
MINIO_USE_SSL=false

ASAAS_API_KEY=TROQUE_ASAAS_API_KEY
ASAAS_WEBHOOK_TOKEN=TROQUE_ASAAS_WEBHOOK_TOKEN
OPENAI_API_KEY=TROQUE_OPENAI_API_KEY
```

---

## 🐳 5. Imagens Oficiais Geradas pelo CI/CD

As imagens públicas são publicadas automaticamente pelo GitHub Actions na conta `williamwilmer10` no Docker Hub:
* **API**: `williamwilmer10/mapaturbo-api:latest`
* **Worker**: `williamwilmer10/mapaturbo-worker:latest`
* **Web (SPA/Nginx)**: `williamwilmer10/mapaturbo-web:latest`

---

## 🚀 6. Processo de Implantação (Deploy via Portainer)

1. Acesse o painel administrativo do Portainer.
2. Navegue até **Stacks** -> **Add stack**.
3. Defina o nome como `mapaturbo-stack` (ou similar).
4. Copie e cole o conteúdo de `docker/stack.yml` no editor.
5. Adicione as variáveis de ambiente em **Environment variables**.
6. Clique em **Deploy the stack**.

---

## 🏷️ 7. Configurações de Roteamento (Traefik)

Os domínios de produção configurados nas labels do Traefik da stack são:
* **Frontend Web (React/Nginx)**: `mapaturbo.wapainel.com.br` (exposto na porta `80`).
* **Backend API (Go)**: `apimapaturbo.wapainel.com.br` (exposto na porta `8080`).
* **MinIO Storage S3**: `miniomapaturbo.wapainel.com.br` (porta `9000`).
* **MinIO Console**: `minioconsolemapaturbo.wapainel.com.br` (porta `9001`).

---

## 🗃️ 8. Rodando as Migrations no Swarm

A API Go do MapaTurbo possui execução automática de migrações (`Goose`) no startup. Assim que o container `mapaturbo_api` inicializa, ele sincroniza automaticamente o esquema do Postgres de forma idempotente, sem necessidade de ações manuais no banco de dados.

---

## 🔍 9. Validação e Teste do Ambiente

Após a subida do stack, valide cada componente executando os seguintes testes:

### A. Validar API (Health Check)
```bash
curl -f https://apimapaturbo.wapainel.com.br/health
# Resposta esperada: {"status":"OK"}
```

### B. Validar Frontend e SPA Fallback
Abra o navegador em `https://mapaturbo.wapainel.com.br`. Tente navegar diretamente para `/precos` ou `/app` e atualize a página pressionando F5. Se a rota carregar perfeitamente sob Nginx, o roteamento SPA está validado.

### C. Validar PGVector (Banco de Dados)
Acesse a CLI do Postgres:
```bash
docker exec -it <POSTGRES_CONTAINER_ID> psql -U mapaturbo -d mapaturbo
```
Rode:
```sql
SELECT * FROM pg_extension WHERE extname = 'vector';
-- Esperado: ver o registro contendo a versão do pgvector.
```
