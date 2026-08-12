package notificacoes

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var (
	fcmClientMu sync.Mutex
	fcmClient   *messaging.Client
	credPathUso string
)

func fcmMensageria(ctx context.Context, cred string) (*messaging.Client, error) {
	if cred == "" {
		return nil, nil
	}
	fcmClientMu.Lock()
	defer fcmClientMu.Unlock()
	if fcmClient != nil && credPathUso == cred {
		return fcmClient, nil
	}
	b, err := os.ReadFile(cred)
	if err != nil {
		return nil, err
	}
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON(b))
	if err != nil {
		return nil, err
	}
	c, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	fcmClient = c
	credPathUso = cred
	return c, nil
}

// EnviarVoucherAprovado push quando o pagamento do voucher no Mercado Pago é aprovado.
// [cred] é o caminho do JSON da conta de serviço Firebase (env FCM_SA).
func EnviarVoucherAprovado(ctx context.Context, cred string, tokens []string, idCompra, codigo, valorReais string) {
	if cred == "" || len(tokens) == 0 {
		return
	}
	c, err := fcmMensageria(ctx, cred)
	if err != nil {
		log.Printf("fcm: abrir credenciais: %v", err)
		return
	}
	if c == nil {
		return
	}
	for i := 0; i < len(tokens); i += 500 {
		j := i + 500
		if j > len(tokens) {
			j = len(tokens)
		}
		batch := tokens[i:j]
		corpoVoucher := fmt.Sprintf("Seu pagamento de R$ %s foi confirmado. Abra o app para resgatar.", valorReais)
		req := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title: "Voucher aprovado",
				Body:  corpoVoucher,
			},
			Data: map[string]string{
				"tipo":       "voucher_ativo",
				"id":         idCompra,
				"codigo":     codigo,
				"valor":      valorReais,
				"abrir_tela": "vouchers",
				"titulo":     "Voucher aprovado",
				"corpo":      corpoVoucher,
			},
		}
		br, err := c.SendEachForMulticast(ctx, req)
		if err != nil {
			log.Printf("fcm: SendEachForMulticast: %v", err)
			return
		}
		if br.FailureCount > 0 {
			log.Printf("fcm: lote: falhas=%d de %d (tokens invalidos ou desinstalacoes antigas)", br.FailureCount, len(batch))
		}
	}
}

// EnviarVoucherUsadoNoPosto push quando o frentista registra a baixa (dinheiro ou ATIVO → USADO).
func EnviarVoucherUsadoNoPosto(ctx context.Context, cred string, tokens []string, idCompra, codigo, valorReais string, dinheiro bool) {
	if cred == "" || len(tokens) == 0 {
		return
	}
	c, err := fcmMensageria(ctx, cred)
	if err != nil {
		log.Printf("fcm: abrir credenciais: %v", err)
		return
	}
	if c == nil {
		return
	}
	titulo := "Voucher utilizado"
	corpo := fmt.Sprintf("Uso de R$ %s registrado no posto. Pode abastecer.", valorReais)
	if dinheiro {
		titulo = "Pagamento confirmado"
		corpo = fmt.Sprintf("Pagamento em dinheiro de R$ %s confirmado no posto.", valorReais)
	}
	for i := 0; i < len(tokens); i += 500 {
		j := i + 500
		if j > len(tokens) {
			j = len(tokens)
		}
		batch := tokens[i:j]
		req := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title: titulo,
				Body:  corpo,
			},
			Data: map[string]string{
				"tipo":       "voucher_usado",
				"id":         idCompra,
				"codigo":     codigo,
				"valor":      valorReais,
				"abrir_tela": "vouchers",
				"titulo":     titulo,
				"corpo":      corpo,
				"status":     "USADO",
			},
		}
		br, err := c.SendEachForMulticast(ctx, req)
		if err != nil {
			log.Printf("fcm: SendEachForMulticast uso: %v", err)
			return
		}
		if br.FailureCount > 0 {
			log.Printf("fcm: lote uso: falhas=%d de %d", br.FailureCount, len(batch))
		}
	}
}

// EnviarNovaCampanhaNoApp push para clientes da rede quando o gestor cria campanha ativa no app.
func EnviarNovaCampanhaNoApp(ctx context.Context, cred string, tokens []string, idCampanha, tituloExibicao, idRede string) {
	if cred == "" {
		log.Printf("fcm campanha: EnviarNovaCampanhaNoApp cred vazio")
		return
	}
	if len(tokens) == 0 {
		return
	}
	c, err := fcmMensageria(ctx, cred)
	if err != nil {
		log.Printf("fcm campanha: abrir credenciais: %v", err)
		return
	}
	if c == nil {
		return
	}
	tit := strings.TrimSpace(tituloExibicao)
	if tit == "" {
		tit = "Nova promocao"
	}
	cid := strings.TrimSpace(idCampanha)
	rid := strings.TrimSpace(idRede)
	sucesso := 0
	for i := 0; i < len(tokens); i += 500 {
		j := i + 500
		if j > len(tokens) {
			j = len(tokens)
		}
		batch := tokens[i:j]
		req := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title: "Nova promocao",
				Body:  tit,
			},
			Data: map[string]string{
				"tipo":        "nova_campanha_app",
				"id_campanha": cid,
				"id_rede":     rid,
				"abrir_tela":  "promocoes",
				"titulo":      "Nova promocao",
				"corpo":       tit,
			},
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					ChannelID: "lucena_plus_fcm",
					Priority:  messaging.PriorityHigh,
					Sound:     "default",
				},
			},
			APNS: &messaging.APNSConfig{
				Headers: map[string]string{"apns-priority": "10"},
				Payload: &messaging.APNSPayload{
					Aps: &messaging.Aps{
						Sound: "default",
					},
				},
			},
		}
		br, err := c.SendEachForMulticast(ctx, req)
		if err != nil {
			log.Printf("fcm campanha: SendEachForMulticast: %v", err)
			return
		}
		sucesso += br.SuccessCount
		if br.FailureCount > 0 {
			log.Printf("fcm campanha: lote: falhas=%d de %d", br.FailureCount, len(batch))
			for i, resp := range br.Responses {
				if resp != nil && !resp.Success && resp.Error != nil && i < len(batch) {
					log.Printf("fcm campanha: token[%d] falhou: %v", i, resp.Error)
				}
			}
		}
	}
	log.Printf("fcm campanha: fcm concluido id_campanha=%s sucesso=%d de %d token(s) id_rede=%s", cid, sucesso, len(tokens), rid)
}

// EnviarTeste notificacao simples (endpoint /v1/eu/push/fcm/teste) para validar FCM no dispositivo.
// Devolve o numero de envios com sucesso no lote; pode ser < len(tokens) se algum token for invalido.
func EnviarTeste(ctx context.Context, cred string, tokens []string) (int, int, error) {
	if cred == "" {
		return 0, 0, fmt.Errorf("credenciais fcm nao configuradas (defina FCM_SA)")
	}
	if len(tokens) == 0 {
		return 0, 0, nil
	}
	c, err := fcmMensageria(ctx, cred)
	if err != nil {
		return 0, 0, err
	}
	if c == nil {
		return 0, 0, fmt.Errorf("cliente fcm nulo")
	}
	ok := 0
	fal := 0
	for i := 0; i < len(tokens); i += 500 {
		j := i + 500
		if j > len(tokens) {
			j = len(tokens)
		}
		batch := tokens[i:j]
		req := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title: "Teste de notificacao",
				Body:  "Se recebeu isto, o push (FCM) esta a funcionar.",
			},
			Data: map[string]string{
				"tipo":       "fcm_teste",
				"abrir_tela": "modal",
				"titulo":     "Teste de notificacao",
				"corpo":      "Se recebeu isto, o push (FCM) esta a funcionar.",
			},
		}
		br, err := c.SendEachForMulticast(ctx, req)
		if err != nil {
			return ok, fal, err
		}
		ok += br.SuccessCount
		fal += br.FailureCount
	}
	return ok, fal, nil
}

// EnviarTesteRede envia notificacao de teste a todos os clientes (tokens FCM) da rede — titulo/corpo personalizaveis.
func EnviarTesteRede(ctx context.Context, cred string, tokens []string, idRede, titulo, corpo string) (int, int, error) {
	if cred == "" {
		return 0, 0, fmt.Errorf("credenciais fcm nao configuradas (defina FCM_SA)")
	}
	if len(tokens) == 0 {
		return 0, 0, nil
	}
	c, err := fcmMensageria(ctx, cred)
	if err != nil {
		return 0, 0, err
	}
	if c == nil {
		return 0, 0, fmt.Errorf("cliente fcm nulo")
	}
	tit := strings.TrimSpace(titulo)
	if tit == "" {
		tit = "Teste de notificacao"
	}
	corp := strings.TrimSpace(corpo)
	if corp == "" {
		corp = "Mensagem de teste do painel."
	}
	rid := strings.TrimSpace(idRede)
	ok := 0
	fal := 0
	for i := 0; i < len(tokens); i += 500 {
		j := i + 500
		if j > len(tokens) {
			j = len(tokens)
		}
		batch := tokens[i:j]
		req := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title: tit,
				Body:  corp,
			},
			Data: map[string]string{
				"tipo":         "fcm_teste_painel",
				"abrir_tela":   "modal",
				"titulo":       tit,
				"corpo":        corp,
				"id_rede":      rid,
				"origem":       "painel",
			},
		}
		br, err := c.SendEachForMulticast(ctx, req)
		if err != nil {
			return ok, fal, err
		}
		ok += br.SuccessCount
		fal += br.FailureCount
	}
	return ok, fal, nil
}
