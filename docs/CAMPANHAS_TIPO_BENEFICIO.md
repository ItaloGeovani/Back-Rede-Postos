# Campanhas: DESCONTO vs CASHBACK

## Coluna no banco (`campanhas.tipo_beneficio`)

| Valor | Significado |
|-------|-------------|
| `DESCONTO` | O benefício **reduz o valor do PIX** na hora da compra. |
| `CASHBACK` | O cliente **paga o valor integral**; após pagamento confirmado, o sistema **credita moeda virtual** na carteira. |

Constraint: `CHECK (tipo_beneficio IN ('DESCONTO', 'CASHBACK'))` (migração `043`).

Cashback exige `modalidade_desconto = 'PERCENTUAL'` e `base_desconto = 'VALOR_COMPRA'`.

## Campo `valor_desconto`

- **Modalidade PERCENTUAL:** use escala **0–100** (ex.: `10` = 10%, `5` = 5%).
- **Modalidade VALOR_FIXO:** valor em **R$** (ex.: `0.10` = R$ 0,10 por litro).
- Legado em fração (`0.05` para 5%) foi corrigido na migração `047` e normalizado no serviço ao salvar.

## Fluxo no voucher (app)

```
DESCONTO  → valor_final = compra − desconto
CASHBACK  → valor_final = compra (integral)
            → após PIX aprovado: crédito na carteira (cashback_previsto)
```

## Painel administrativo

Ao criar/editar campanha, escolha explicitamente **Tipo de benefício**. Não use só a palavra “cashback” no título — o tipo deve ser `CASHBACK` no banco.

## Migração de dados legados

Execute `banco/migracoes/047_campanha_corrigir_tipo_e_percentual_legado.sql` no Postgres de produção.

Ela:

1. Converte percentuais `0 < valor < 1` para escala 0–100.
2. Define `tipo_beneficio = 'CASHBACK'` onde o texto da campanha indica cashback mas o tipo estava `DESCONTO`.

Após a migração, a API lista o `tipo_beneficio` correto sem depender de heurística no nome.
