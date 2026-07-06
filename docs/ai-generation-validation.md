# Guia de Validação da Geração de Mapas com IA (Fase 3A)

Este guia descreve como validar e testar localmente o pipeline de geração de mapas mentais com IA, gerenciamento de jobs e consumo seguro de créditos.

---

## 1. Configurando Chaves e Provedores IA

1. Insira a chave criptográfica simétrica no arquivo `.env`:
   ```env
   ENCRYPTION_KEY=sua_chave_secreta_de_32_bytes_ou_mais
   OPENAI_API_KEY=sk-proj-sua-chave-openai-real
   ```
2. Inicialize o banco de dados e as migrações:
   ```bash
   # Executado automaticamente no startup da API
   ```
3. Cadastre o provedor padrão no Super Admin:
   * Acesse `/admin/ai-providers`.
   * Clique em **+ Novo Provedor**.
   * Escolha o slug `openai`, insira a sua API Key real (ela será criptografada com AES-256-GCM em banco) e marque a opção **Definir como Padrão Global**.

---

## 2. Fluxo 1: Geração por Tema / Tópico
Este fluxo gera um mapa estruturado a partir de um título e palavras-chave.
* **Endpoint**: `POST /mindmaps/generate`
* **Headers**:
  * `Authorization: Bearer <SEU_JWT_TOKEN>`
  * `X-Organization-ID: <UUID_DO_TENANT>`
* **Payload**:
  ```json
  {
    "type": "TOPIC",
    "title": "Revolução Industrial",
    "content": "Impacto socioeconômico, surgimento das fábricas na Inglaterra e motores a vapor.",
    "options": {
      "depth": 3,
      "language": "pt-BR",
      "style": "study"
    }
  }
  ```

---

## 3. Fluxo 2: Geração por Texto Colado
Ideal para sumarizar artigos longos ou notas de aula.
* **Endpoint**: `POST /mindmaps/generate`
* **Payload**:
  ```json
  {
    "type": "TEXT",
    "title": "Resumo de Termodinâmica",
    "content": "A Primeira Lei da Termodinâmica é uma versão da lei de conservação da energia adaptada para sistemas termodinâmicos. A energia total de um sistema isolado é constante...",
    "options": {
      "depth": 4,
      "language": "pt-BR",
      "style": "study"
    }
  }
  ```

---

## 4. Acompanhamento por Polling (Fila de Trabalho)
Ao submeter a geração, a API retorna um `jobId` e o status inicial `PENDING`.
O cliente deve fazer requisições a cada 2 ou 3 segundos para:
* **Endpoint**: `GET /generation-jobs/<jobId>`
* **Resposta Esperada**:
  ```json
  {
    "status": "success",
    "data": {
      "id": "uuid-do-job",
      "status": "PROCESSING", // ou COMPLETED / FAILED
      "error": null,
      "mind_map_id": "uuid-do-mapa-se-completed"
    }
  }
  ```

---

## 5. Regras de Débito de Crédito e Prevenção de Concorrência
Para evitar vazamentos de créditos em requisições paralelas:
1. O backend valida previamente se a organização possui saldo >= custo da ação (`GENERATE_MAP_TOPIC` ou `GENERATE_MAP_TEXT`).
2. No worker, o débito ocorre **somente após a conclusão com sucesso** da chamada de IA.
3. O worker executa um bloqueio de linha no banco usando transação SQL:
   ```sql
   SELECT balance FROM ai_credit_balances WHERE organization_id = $1 FOR UPDATE;
   ```
4. Se o saldo for menor no encerramento (devido a débitos em outros jobs paralelos), o job falha com erro de saldo e nenhum crédito é debitado.
5. Se a IA falhar no processamento, o job é atualizado para `FAILED` e nenhum crédito é debitado.
6. A tabela `ai_credit_transactions` registra o hash do `generationJobId` impedindo transações duplicadas (idempotência).

---

## 6. Validação do JSON Estruturado e Autocorreção
A OpenAI é chamada com a propriedade `"response_format": {"type": "json_object"}`.
O worker realiza as seguintes validações sobre o JSON:
* Deve conter chaves `title`, `centralTopic`, `summary`, `nodes` e `edges`.
* Deve conter exatamente um nó `"root"` sem parentId (`null`) e nível `0`.
* Todos os outros nós precisam ter parentId apontando para nós válidos da lista.
* Os níveis dos nós devem respeitar o limite de profundidade (`depth`) solicitado.
* Limite máximo de 80 nós e limites de caracteres para títulos (120) e conteúdos (1000).

### Comportamento em Falha:
Se o JSON inicial violar qualquer regra, o worker executa **uma única tentativa** de correção enviando as mensagens de erro de validação de volta à IA. Se a correção falhar novamente, o status do job é marcado como `FAILED`.
