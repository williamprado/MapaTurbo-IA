# Configuração de CI/CD: GitHub Actions & Docker Hub

Este guia explica como configurar e utilizar a esteira de integração e entrega contínua (CI/CD) para gerar e publicar as imagens do **MapaTurbo IA** no Docker Hub.

---

## 🔑 1. Configurando Credenciais no Docker Hub

Para evitar expor a senha principal da sua conta no Docker Hub, é obrigatório criar um **Access Token (PAT)** dedicado para o GitHub Actions:

1. Acesse o [Docker Hub](https://hub.docker.com/) e faça login com a conta `williamwilmer10`.
2. Vá em **My Account** -> **Security** -> **Personal Access Tokens**.
3. Clique em **Generate new token**.
4. Defina uma descrição (ex: `GitHub Actions CI`) e permissões de **Read & Write**.
5. Copie o token gerado (começa com `dckr_pat_`). **Aviso**: ele só é exibido uma vez.

---

## ⚙️ 2. Configurando o Repositório do GitHub

No seu repositório no GitHub (`https://github.com/williamprado/MapaTurbo-IA`), acesse:
`Settings > Secrets and variables > Actions`

### A. Cadastrar as Variables (Públicas)
Clique em **New repository variable** e adicione:
* `DOCKER_USERNAME` = `williamwilmer10`
* `DOCKER_NAMESPACE` = `williamwilmer10`
* `VITE_API_URL` = `https://apimapaturbo.wapainel.com.br`
* `VITE_APP_URL` = `https://mapaturbo.wapainel.com.br`

### B. Cadastrar os Secrets (Confidenciais)
Clique em **New repository secret** e adicione:
* `DOCKER_TOKEN` = `<COLE_O_ACCESS_TOKEN_DO_DOCKER_HUB>`
* `DOCKER_PASSWORD` = `<COLE_O_ACCESS_TOKEN_DO_DOCKER_HUB>` *(cadastre em ambos os nomes para garantir compatibilidade com workflows legados)*

---

## 🐳 3. Imagens e Tags Publicadas

O workflow automatizado compila e publica 3 imagens separadas no namespace `williamwilmer10`:
1. `williamwilmer10/mapaturbo-api`
2. `williamwilmer10/mapaturbo-worker`
3. `williamwilmer10/mapaturbo-web`

Para cada push aceito na branch `main`, as imagens recebem as seguintes tags:
* `latest`: Aponta sempre para a versão mais recente da branch principal.
* `v1.0.<RUN_NUMBER>`: Tag incremental de versão comercial (ex: `v1.0.12`).
* `sha-<hash_curto>`: Identificador do commit correspondente no Git (ex: `sha-9dd768d`).

---

## 💻 4. Rodando o Script de Build Localmente (`build_and_push.sh`)

Você pode compilar as imagens na sua máquina local sem precisar disparar o GitHub Actions.

### Build Local (Apenas compilação, sem push)
Dê permissão de execução (se necessário) e execute:
```bash
# Definindo as variáveis necessárias localmente
export DOCKER_NAMESPACE=williamwilmer10
export VITE_API_URL=https://apimapaturbo.wapainel.com.br
export VITE_APP_URL=https://mapaturbo.wapainel.com.br

./build_and_push.sh
```

### Build e Push Local para o Docker Hub
Caso queira compilar localmente e já enviar as imagens para o repositório público `williamwilmer10`:
```bash
# Efetue login no Docker Hub antes de rodar
docker login -u williamwilmer10

export DOCKER_NAMESPACE=williamwilmer10
export PUSH_IMAGES=true
export VITE_API_URL=https://apimapaturbo.wapainel.com.br
export VITE_APP_URL=https://mapaturbo.wapainel.com.br

./build_and_push.sh
```

---

## ⚓ 5. Utilizando as Imagens no Portainer/Swarm

No painel do Portainer do WA Painel, ao subir a stack usando o [docker/stack.yml](file:///I:/MapaTurbo%20IA/docker/stack.yml), as imagens serão puxadas automaticamente do Docker Hub usando a tag `:latest` (ou uma tag de release específica):

```yaml
services:
  mapaturbo_api:
    image: williamwilmer10/mapaturbo-api:latest
    ...
  mapaturbo_worker:
    image: williamwilmer10/mapaturbo-worker:latest
    ...
  mapaturbo_web:
    image: williamwilmer10/mapaturbo-web:latest
    ...
```
