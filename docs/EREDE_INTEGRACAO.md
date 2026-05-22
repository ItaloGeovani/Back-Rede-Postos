# Integração e.Rede (Itaú) — GasPass

Este documento descreve a operação da integração e.Rede no servidor Go e no painel web.

## Credenciais

| Portal sandbox | API v2 |
|----------------|--------|
| **PV** | `clientId` |
| **Token** | `clientSecret` |

Autenticação: OAuth 2.0 `client_credentials` (Bearer ~24 min), não o token fixo do painel.

**Nunca** commitar PV/secret no repositório. Use o painel (Gateways → e.Rede) ou variáveis de ambiente apenas em dev local.

## Endpoints (padrão)

| Ambiente | OAuth | Transações PIX |
|----------|-------|----------------|
| Sandbox | `https://rl7-sandbox-api.useredecloud.com.br/oauth2/token` | `https://sandbox-erede.useredecloud.com.br/v2/transactions` |
| Produção | `https://api.userede.com.br/redelabs/oauth2/token` | `https://api.userede.com.br/erede/v2/transactions` |

Override opcional: `EREDE_OAUTH_SANDBOX_URL`, `EREDE_TX_SANDBOX_URL`, `EREDE_AMBIENTE_DEFAULT`.

## Fluxo voucher PIX

1. App: `POST /v1/eu/vouchers/pagar` (contrato inalterado: `qr_code`, `qr_code_base64`, `compra_id`; opcional `gateway_payment_id`).
2. Servidor: OAuth → `POST` transação `kind: pix`, valor em **centavos**, `reference` (máx. 16 caracteres).
3. Sandbox: pagamento simulado em ~**2 minutos** após exibir o QR.
4. Webhook público: `POST /v1/public/erede/webhook/{rede_id}` ou `.../{rede_id}/{posto_id}`.
5. Evento: `PV.UPDATE_TRANSACTION_PIX`; `data.id` = **tid**.
6. Servidor ativa voucher (`ATIVO`) e aplica regra de baixa no posto (`posto_id_compra` no modo POSTO).

## Painel

- **Provedor ativo**: Mercado Pago **ou** e.Rede (exclusivo por rede).
- **Meios**: PIX na fase 1; cartão desabilitado para e.Rede (“em breve”).
- **Modo REDE/POSTO**: igual ao Mercado Pago.
- Rotas gestor: `GET/PUT /v1/gestor-rede/dev/erede-gateway`, `PUT .../posto`, `PUT .../redes/gateway-provedor`.
- Gerente (modo POSTO): `GET/PUT /v1/gerente-posto/dev/erede-gateway/posto`.

`GET /v1/public/rede-info` expõe `gateway_provedor_ativo` e `gateway_meios_habilitados` para os apps.

## Checklist operacional

### 1. IPs confiáveis (obrigatório até OAuth pleno)

No portal e.Rede: **Para Vender → E-commerce → Gestão de IPs**, cadastre o IP público de **saída** do servidor (ou do proxy reverso). Erro **3301** indica IP não autorizado.

### 2. Webhook — sandbox

- Configure `PUBLIC_BASE_URL` apontando para URL acessível pela e.Rede (túnel em dev).
- Ao salvar credenciais (rede ou posto) em sandbox, o servidor registra a URL via `POST .../v1/transactions/notification-URL` (campo `URL` no body).
- URL a copiar no painel: `{PUBLIC_BASE_URL}/v1/public/erede/webhook/{rede_id}`.

### 3. Webhook — produção

Abertura de chamado no Itaú (call center): CNPJ, PV, e-mail, URL pública. Prazo típico ~2 dias úteis. Não basta colar URL só no painel GasPass.

### 4. Migração de banco

Aplicar `banco/migracoes/049_gateway_provedor_erede.sql` (e `048` se ainda não aplicada).

### 5. Teste sandbox (TestePostos)

1. Painel: provedor **e.Rede**, PIX habilitado, credenciais sandbox (PV + Token).
2. App cliente: comprar voucher PIX.
3. Pagar QR (ou aguardar simulação ~2 min).
4. Confirmar webhook e status **ATIVO** no painel/lista do cliente.
5. Modo POSTO: baixa apenas no posto escolhido na compra.

### 6. OAuth 2.0 obrigatório

Prazo documentado Itaú: **01/08/2026**. Implementação GasPass já usa API v2 com `client_credentials`.

## Referências oficiais

| Recurso | Uso no GasPass |
|---------|----------------|
| [Documentação e.Rede](https://developer.userede.com.br/e-rede) | PIX (`kind: pix`), webhooks `PV.UPDATE_TRANSACTION_PIX`, OAuth 2.0 |
| Collection Postman `docs/Sandboxe.Rede.postman_collection.json` | Variáveis sandbox; testes manuais de cartão/v1 e tokenização |

### Variáveis da collection (sandbox)

A collection **Sandbox e.Rede** define (aba *Variables*):

| Variável Postman | Valor padrão | Equivalente no servidor |
|------------------|--------------|-------------------------|
| `base_url` | `https://sandbox-erede.useredecloud.com.br` | `POST {base}/v2/transactions` (PIX) |
| `urlToken` | `https://rl7-sandbox-api.useredecloud.com.br` | OAuth `POST {url}/oauth2/token` |
| `pv` | PV do projeto TestePostos | Painel → e.Rede → PV |
| `token` | Token do portal | Painel → client secret |

**Autenticação na collection**

- **Transações v1** (cartão): Basic Auth `pv:token` em `{{base_url}}/v1/transactions`.
- **Tokenização / OAuth**: `POST {{urlToken}}/oauth2/token` com `grant_type=client_credentials` (igual ao nosso `interno/servicos/erede/oauth.go`).
- **PIX (voucher GasPass)**: não há request pronto na collection; use a doc [e.Rede → Pix](https://developer.userede.com.br/e-rede) ou replique o body do servidor: `kind: "pix"`, `amount` em centavos, `reference` (≤16 chars), `qrCode.dateTimeExpiration`.

### Conferência rápida com o código

```text
OAuth sandbox  → https://rl7-sandbox-api.useredecloud.com.br/oauth2/token
PIX sandbox    → https://sandbox-erede.useredecloud.com.br/v2/transactions
Consulta tid   → GET .../v2/transactions/{tid}
Webhook GasPass → POST {PUBLIC_BASE_URL}/v1/public/erede/webhook/{rede_id}
```

Guia interno complementar: `servidor-go/docs/Guia Completo_ Integração e Segurança e.Rede`.
