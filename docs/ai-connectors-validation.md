# Guia de Validação de Conectores de IA

Este documento descreve como configurar, criptografar chaves de API e testar os conectores de Inteligência Artificial do MapaTurbo IA.

---

## 1. Segurança e Criptografia Simétrica (AES-256-GCM)
Todas as chaves secretas e tokens de integração dos provedores de IA são armazenadas criptografadas no Postgres na tabela `ai_providers.api_key_secure`.
* O backend utiliza a variável `ENCRYPTION_KEY` definida no arquivo `.env`.
* A chave é derivada de forma segura utilizando SHA-256 para gerar uma chave simétrica de 32 bytes robusta para a cifra AES-GCM.
* Em nenhum endpoint público ou administrativo a chave de API descriptografada é retornada. O backend envia uma máscara como `****abcd` ou `********` para visualização segura no painel do administrador.

---

## 2. Cadastro de Provedores no Banco
O Super Admin pode cadastrar os provedores via painel `/admin/ai-providers` ou via API:
```bash
curl -X POST http://localhost:8080/admin/ai-providers \
  -H "Authorization: Bearer <SEU_JWT_TOKEN_SUPER_ADMIN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "OpenAI Production",
    "slug": "openai",
    "apiKey": "sk-proj-your_secret_openai_key",
    "baseUrl": "https://api.openai.com/v1",
    "defaultModel": "gpt-4o",
    "textModel": "gpt-4o",
    "visionModel": "gpt-4o-mini",
    "embeddingModel": "text-embedding-3-small",
    "embeddingDimensions": 1536,
    "isActive": true,
    "priority": 100,
    "isDefault": true,
    "limitPerMinute": 60,
    "limitPerDay": 10000,
    "costPerCredit": 0.05
  }'
```

---

## 3. Teste de Conexão em Tempo Real
Para verificar se o provedor está devidamente configurado e operacional com uma chave de API válida:
```bash
curl -X POST http://localhost:8080/admin/ai-providers/<UUID_DO_PROVEDOR>/test \
  -H "Authorization: Bearer <SEU_JWT_TOKEN_SUPER_ADMIN>"
```

### Comportamento Esperado:
1. O backend busca o provedor, decodifica a chave criptografada via AES-GCM com a `ENCRYPTION_KEY`.
2. Para o provedor `openai`, executa uma chamada real ao endpoint `https://api.openai.com/v1/models` para validar as credenciais.
3. Se a chamada retornar HTTP `200`, responde com `{"ok": true, "message": "Connection successful: models retrieved."}`.
4. Para provedores temporários (`gemini`, `grok`, `anthropic`), responde indicando que o teste real ainda é um placeholder.
5. Grava um rastro de auditoria com a ação `AI_PROVIDER_TESTED`.
