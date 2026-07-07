# Manual de Deploy: Docker Swarm & Portainer — MapaTurbo IA

Este documento orienta sobre a implantação, orquestração e gerenciamento da infraestrutura do **MapaTurbo IA** em ambiente de produção utilizando um cluster **Docker Swarm** com **Portainer** e roteamento reverso via **Traefik**.

---

## 📋 1. Pré-requisitos

* Cluster Docker Swarm inicializado (`docker swarm init`).
* **Traefik** implantado globalmente no cluster escutando a rede overlay de borda (por padrão chamada `traefik-net` ou integrada).
* Chave de API OpenAI e token/chave Asaas Sandbox ou Produção configurados.
* Servidor com portas `80` e `443` livres no Host Manager.

---

## 🌐 2. Redes Docker Overlay Necessárias

Para isolar o banco de dados e Redis do tráfego web público, os serviços compartilham uma rede de overlay isolada. Caso queira expor o front/API via Traefik, certifique-se de que há uma rede externa para o proxy.

Criação da rede interna do projeto:
```bash
docker network create --driver=overlay mapaturbo_net
```

---

## 💾 3. Volumes Necessários

Os serviços estáticos persistirão dados no Host local utilizando volumes locais do cluster Swarm:
* `postgres_prod_data`: Armazenamento físico de tabelas e vetores.
* `redis_prod_data`: Estado das filas e jobs em andamento.
* `minio_prod_data`: Armazenamento de arquivos PDF enviados.

---

## 🔒 4. Variáveis de Ambiente & Secrets

Os segredos não devem ser fixados no arquivo `stack.yml`. Para implantações seguras no Portainer ou CLI Swarm, defina as seguintes variáveis de ambiente ou utilize Docker Secrets:

```txt
# Banco e Cache
DATABASE_URL=postgres://mapaturbo:<SENHA>@postgres:5432/mapaturbo?sslmode=disable
POSTGRES_USER=mapaturbo
POSTGRES_PASSWORD=<SENHA_BANCO_COMPLEXA>
POSTGRES_DB=mapaturbo

# Segurança e Criptografia
JWT_SECRET=<JWT_CHAVE_32_BYTES>
ENCRYPTION_KEY=<CHAVE_AES_EXATA_32_BYTES>

# Gateways e IA
ASAAS_API_KEY=<API_KEY_ASAAS>
ASAAS_WEBHOOK_TOKEN=<WEBHOOK_TOKEN_ASAAS>
OPENAI_API_KEY=<OPENAI_API_KEY>

# MinIO
MINIO_ROOT_USER=mapaturbo
MINIO_ROOT_PASSWORD=<SENHA_MINIO>
MINIO_BUCKET=mapaturbo-files
MINIO_USE_SSL=false
```

---

## 🐳 5. Build e Publicação das Imagens

Antes de rodar a stack, as imagens contendo a versão final do código devem ser geradas e enviadas ao registro de contêineres (Docker Hub ou Registry Privado):

```bash
# Build e Push API
docker build -t william/mapaturbo-api:latest --target api ./backend
docker push william/mapaturbo-api:latest

# Build e Push Worker
docker build -t william/mapaturbo-worker:latest --target worker ./backend
docker push william/mapaturbo-worker:latest

# Build e Push Web (React + Nginx)
docker build \
  --build-arg VITE_API_URL=https://api.seudominio.com \
  --build-arg VITE_APP_URL=https://app.seudominio.com \
  -t william/mapaturbo-web:latest ./frontend
docker push william/mapaturbo-web:latest
```

---

## 🚀 6. Processo de Implantação (Deploy)

### Opção A: Deploy via CLI (Stack Deploy)
Copie o arquivo `docker/stack.yml` para o servidor gerente e execute:
```bash
docker stack deploy -c docker/stack.yml mapaturbo_stack
```

### Opção B: Deploy via Portainer
1. Acesse o painel do Portainer.
2. Navegue até **Stacks** -> **Add stack**.
3. Escolha um nome (ex: `mapaturbo-stack`).
4. Cole o conteúdo de `docker/stack.yml` no Web Editor.
5. Adicione as variáveis de ambiente necessárias em **Environment variables** na base da tela.
6. Clique em **Deploy the stack**.

---

## 🏷️ 7. Configurações de Roteamento (Traefik)

O `stack.yml` declara labels integradas do Traefik. Os domínios sugeridos para subida são:
* **Frontend Web (React/Nginx)**: `mapaturbo.local` (ou seu domínio de produção) exposto na porta `80`.
* **Backend API (Go)**: `api.mapaturbo.local` (ou seu subdomínio API) na porta `8080`.
* **MinIO Storage S3**: `s3.mapaturbo.local` (porta `9000`) e console administrativo em `console-s3.mapaturbo.local` (porta `9001`).

---

## 🗃️ 8. Rodando as Migrations no Swarm

Para atualizar as tabelas do banco no contêiner Postgres ativo do Swarm:
1. Localize o ID do container ativo:
   ```bash
   docker ps | grep postgres
   ```
2. Acesse o container e aplique as migrações (as migrações Go estão auto-embutidas e rodam no startup da API ou podem ser disparadas manualmente):
   * Nota: A API Go do MapaTurbo possui auto-migration no startup, ou seja, ao subir o container `api`, a migration é executada de forma idempotente e segura contra concorrência.

---

## 🔍 9. Homologação e Validação dos Serviços

Após a subida, valide cada serviço individualmente:

### A. Validar API (Health Check)
```bash
curl -f http://api.mapaturbo.local/health
# Esperado: {"status":"OK"}
```

### B. Validar Worker (Fila Redis)
Acesse os logs do Worker para certificar conexão correta com o Redis:
```bash
docker service logs mapaturbo_stack_worker
# Esperado: "Starting Asynq worker server..." e ausência de erros de conexão Redis.
```

### C. Validar Web (Nginx e Fallback SPA)
Abra o navegador no endereço configurado. Navegue diretamente para `/precos` ou `/app` e recarregue a página (F5).
* Se retornar a página correta sem erro 404, o Nginx está servindo as rotas SPA perfeitamente.

### D. Validar PGVector (Banco de Dados)
Acesse a CLI do banco:
```bash
docker exec -it <POSTGRES_CONTAINER_ID> psql -U mapaturbo -d mapaturbo
```
Rode:
```sql
SELECT * FROM pg_extension WHERE extname = 'vector';
-- Esperado: ver o registro contendo a versão do pgvector.
```

---

## 🔄 10. Atualizações (Rolling Updates) e Rollback

Para atualizar imagens em produção sem indisponibilidade de serviço:
```bash
docker service update --image william/mapaturbo-api:latest mapaturbo_stack_api
```

Se houver problemas durante a inicialização da nova versão, execute o rollback imediato para restaurar o estado anterior:
```bash
docker service update --rollback mapaturbo_stack_api
```
