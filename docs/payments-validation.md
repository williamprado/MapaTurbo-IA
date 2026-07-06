# Guia de Validação de Cobranças (Billing Asaas)

Este documento descreve como configurar, testar e simular em ambiente local os fluxos de checkout e webhooks do gateway **Asaas** no MapaTurbo IA.

---

## 1. Variáveis de Ambiente Necessárias
Adicione ao arquivo `.env` as seguintes chaves do Asaas:
```env
ASAAS_ENV=sandbox
ASAAS_BASE_URL=https://sandbox.asaas.com/api/v3
ASAAS_API_KEY=your_sandbox_api_key
ASAAS_WEBHOOK_TOKEN=your_secure_webhook_token_signature
ASAAS_SUCCESS_URL=http://localhost:5173/app/billing/success
ASAAS_CANCEL_URL=http://localhost:5173/app/billing/cancel
```

---

## 2. Configurando o Sandbox
1. Crie uma conta de testes no painel [Asaas Sandbox](https://sandbox.asaas.com/).
2. Obtenha a API Key em **Configurações da Conta > Integrações > Gerar API Key**.
3. Acesse o painel **Super Admin > Faturamento > Gateways** em seu MapaTurbo local e salve a API Key (ela será criptografada com AES-256-GCM antes de ser salva na tabela `payment_providers`).

---

## 3. Testando Criação de Checkout (CURL)
Simule uma intenção de contratação de plano enviada pelo frontend:
```bash
curl -X POST http://localhost:8080/billing/checkout \
  -H "Authorization: Bearer <SEU_JWT_TOKEN>" \
  -H "X-Organization-ID: <UUID_DO_TENANT>" \
  -H "Content-Type: application/json" \
  -d '{
    "planId": "3b29c9cc-ffcc-47e1-88ff-45ea05151522",
    "cycle": "monthly",
    "billingType": "PIX",
    "document": "12345678909",
    "phone": "11988887777"
  }'
```
### Efeito Esperado:
1. O backend verifica se o `external_customer_id` já existe em `payment_customers` para aquele tenant. Caso contrário, cadastra o cliente no Asaas e armazena os metadados no Postgres.
2. Cria uma cobrança na API do Asaas e retorna a URL de checkout, QR Code e Pix Copia e Cola.
3. Cria localmente a `subscription` (status `PENDING`), a `invoice` (status `PENDING`) e a transação correspondente.

---

## 4. Simulação de Webhook (Confirmação de Pagamento)
Para testar a liberação automática sem precisar pagar de fato no Asaas, dispare o payload de webhook localmente:
```bash
curl -X POST http://localhost:8080/webhooks/asaas \
  -H "asaas-access-token: your_secure_webhook_token_signature" \
  -H "Content-Type: application/json" \
  -d '{
    "event": "PAYMENT_CONFIRMED",
    "payment": {
      "id": "pay_asaas_external_invoice_id",
      "customer": "cus_external_customer_id",
      "value": 49.90,
      "billingType": "PIX",
      "status": "CONFIRMED",
      "externalReference": "local_invoice_reference_uuid"
    }
  }'
```

### Regras de Idempotência e Efeitos:
1. O endpoint grava o evento bruto em `webhook_events` com ID único `pay_asaas_external_invoice_id:PAYMENT_CONFIRMED`. Se o webhook for duplicado, o banco impede a inserção duplicada e retorna HTTP `200` imediatamente.
2. O worker de segundo plano consome a fila e processa a transação local:
   - Marca a `invoice` como `PAID`.
   - Atualiza a `subscription` para `ACTIVE` e estende a validade.
   - Acrescenta os créditos do plano ao saldo da empresa em `ai_credit_balances`.
   - Registra o log financeiro em `ai_credit_transactions` (evitando duplicações).
   - Escreve os registros de auditoria correspondentes.

---

## 5. Validação de Isolamento Multiempresa
* O middleware `TenantMiddleware` garante que nenhuma requisição a `/billing/checkout` prossiga sem validar a associação do usuário logado ao cabeçalho `X-Organization-ID`.
* O backend impede que uma empresa inicie o checkout ou altere faturas de outro workspace.
