# Guia de Validação de Upload de PDF & RAG com PGVector (Fase 3C)

Este guia descreve como validar operacionalmente a ingestão de PDFs, indexação de embeddings via PGVector no PostgreSQL e o fluxo de geração RAG.

---

## 1. Configurações de Ambiente Requeridas
Certifique-se de ter as seguintes variáveis de ambiente configuradas no arquivo `.env`:
```env
OPENAI_API_KEY=sk-proj-sua-chave-openai-real
ENCRYPTION_KEY=sua_chave_secreta_de_32_bytes_ou_mais
```
A API Key do OpenAI é utilizada para a chamada de Embeddings (`text-embedding-3-small` com 1536 dimensões) e posterior síntese RAG com Chat Completion.

---

## 2. Roteiro de Teste 1: Upload e Ingestão de PDF

1. **Enviar um PDF textual de teste**:
   * **Endpoint**: `POST /uploads`
   * **Headers**:
     * `Authorization: Bearer <SEU_JWT_TOKEN>`
     * `X-Organization-ID: <UUID_DO_TENANT>`
   * **Payload**: Form-Data multipart contendo a chave `file` com o arquivo PDF.
   * **Validação**: O tamanho máximo permitido é de **20MB** e a extensão/tipo MIME deve ser estritamente PDF.
   * A resposta retorna o status inicial `UPLOADED` e a chave de persistência S3/MinIO (`storage_key`).

2. **Processamento em Background (Worker `process_pdf_upload`)**:
   * O worker Asynq inicia o processamento do PDF:
     1. Altera o status do upload para `PROCESSING`.
     2. Cria o registro `document_sources` com status `CHUNKING`.
     3. Faz o download do arquivo a partir do MinIO.
     4. Extrai o texto limpo do arquivo usando a biblioteca `github.com/dslipak/pdf`.
        * *Nota*: Caso seja enviado um PDF escaneado (imagem) ou protegido por senha sem texto selecionável, o status final é atualizado para `FAILED` com mensagem amigável no campo metadata: `"O PDF parece estar em branco ou foi escaneado (sem texto selecionável)."`.
     5. Fatia o texto em blocos (chunks) usando a estratégia de janela deslizante (tamanho: 750 caracteres, overlap: 150 caracteres).
     6. Gera os embeddings de 1536 dimensões chamando a API do OpenAI.
     7. Grava os vetores no banco Postgres com a extensão PGVector nativa.
     8. Atualiza `document_sources` para `READY` contendo a contagem final de blocos e palavras.
     9. Altera o upload para `PROCESSED`.

---

## 3. Roteiro de Teste 2: Geração RAG por PDF
Uma vez que o PDF está com o status `PROCESSED` no banco:

1. **Disparar Geração**:
   * **Endpoint**: `POST /mindmaps/generate-from-upload`
   * **Payload**:
     ```json
     {
       "uploadId": "uuid-do-pdf-carregado",
       "query": "tópico específico contido no documento (opcional)",
       "options": {
         "depth": 3,
         "language": "pt-BR",
         "style": "study"
       }
     }
     ```
   * **Validação de Créditos**: O sistema consulta a tabela `ai_action_prices` para buscar o custo da ação `GENERATE_MAP_PDF` (fallback: 15 créditos). O saldo é verificado antes de enfileirar o job.

2. **Execução no Worker (`generate_mindmap_from_pdf`)**:
   * Se a propriedade `query` for enviada:
     * O worker gera o embedding da consulta do usuário.
     * Executa a busca por cosseno no banco vetorial via PGVector:
       ```sql
       SELECT content FROM document_chunks
       WHERE organization_id = $1 AND upload_id = $2
       ORDER BY embedding <=> $3::vector LIMIT 10
       ```
   * Se a propriedade `query` estiver vazia:
     * O worker recupera os primeiros 12 chunks do documento em ordem sequencial para gerar um resumo geral.
   * Concatena os trechos em um prompt contextualizado que instrui a OpenAI a criar o mapa mental em JSON.
   * Em caso de sucesso, efetua o débito do saldo de créditos de forma atômica no banco usando lock de linha pessimista (`SELECT FOR UPDATE`), cria o rastro de auditoria `AI_CREDITS_DEBITED`, salva o mapa conceitual com `source_type = PDF` e atualiza o job para `COMPLETED`.

---

## 4. Testes de Segurança Multiempresa (Tenant Isolation)

1. Tente realizar o upload de um arquivo PDF simulando um cabeçalho `X-Organization-ID` de outra empresa. O sistema deve barrar o acesso.
2. Tente disparar o endpoint `POST /mindmaps/generate-from-upload` passando um `uploadId` pertencente à outra organização. O backend deve recusar com `403 Forbidden` e o job não deve ser criado.
