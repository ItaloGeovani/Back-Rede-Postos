package notificacoes

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

// WhatsAppTemplateDados campos usados nos templates.
type WhatsAppTemplateDados struct {
	Cabecalho string // posto ou rede
	Valor     string
	Quem      string
	DataHora  string
	Meio      string
	Status    string
	Codigo    string
	Titulo    string
	Extra     string
}

func VarianteTemplate(eventoID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(eventoID)))
	return int(h.Sum32() % 5)
}

func RenderWhatsAppTemplate(tipo string, variante int, d WhatsAppTemplateDados) string {
	if variante < 0 || variante > 4 {
		variante = 0
	}
	cab := strings.TrimSpace(d.Cabecalho)
	if cab == "" {
		cab = "REDE"
	}
	header := fmt.Sprintf("== [ %s ] =====", strings.ToUpper(cab))
	data := strings.TrimSpace(d.DataHora)
	if data == "" {
		data = time.Now().Format("02/01/2006 15:04")
	}

	switch tipo {
	case modelos.EventoVoucherGerado:
		return renderVoucherGerado(variante, header, d, data)
	case modelos.EventoVoucherPago:
		return renderVoucherPago(variante, header, d, data)
	case modelos.EventoVoucherBaixa:
		return renderVoucherBaixa(variante, header, d, data)
	case modelos.EventoCampanhaCriada:
		return renderCampanhaCriada(variante, header, d, data)
	case modelos.EventoCampanhaAtivada:
		return renderCampanhaAtivada(variante, header, d, data)
	default:
		return fmt.Sprintf("%s\n*%s*\n%s\n==================", header, strings.TrimSpace(d.Titulo), data)
	}
}

func renderVoucherGerado(v int, header string, d WhatsAppTemplateDados, data string) string {
	switch v {
	case 1:
		return fmt.Sprintf(`%s
*Voucher gerado*
R$ %s · %s
Meio: %s | Status: %s
%s
==================`, header, d.Valor, d.Quem, d.Meio, d.Status, data)
	case 2:
		return fmt.Sprintf(`%s
NOVO VOUCHER
Cliente: *%s*
Valor: *R$ %s*
Pagamento: %s (%s)
Horário: %s
==================`, header, d.Quem, d.Valor, d.Meio, d.Status, data)
	case 3:
		return fmt.Sprintf(`%s
*Geração de voucher*
• Valor: R$ %s
• Quem: %s
• Quando: %s
• Canal: %s
• Situação: %s
==================`, header, d.Valor, d.Quem, data, d.Meio, d.Status)
	case 4:
		return fmt.Sprintf(`%s
*%s*
Valor R$ %s para %s via %s
Status atual: %s
Registrado em %s
==================`, header, "VOUCHER CRIADO", d.Valor, d.Quem, d.Meio, d.Status, data)
	default:
		return fmt.Sprintf(`%s
*NOVO VOUCHER GERADO*
Valor: R$ %s
Quem: %s
Data: %s
Meio: %s
Status: %s
==================`, header, d.Valor, d.Quem, data, d.Meio, d.Status)
	}
}

func renderVoucherPago(v int, header string, d WhatsAppTemplateDados, data string) string {
	cod := ""
	if strings.TrimSpace(d.Codigo) != "" {
		cod = "\nCódigo: " + strings.TrimSpace(d.Codigo)
	}
	switch v {
	case 1:
		return fmt.Sprintf(`%s
*Pagamento confirmado*
R$ %s — %s
Meio %s · %s%s
==================`, header, d.Valor, d.Quem, d.Meio, data, cod)
	case 2:
		return fmt.Sprintf(`%s
*VOUCHER PAGO*
Cliente *%s* quitou R$ *%s*
Via: %s
%s%s
==================`, header, d.Quem, d.Valor, d.Meio, data, cod)
	case 3:
		return fmt.Sprintf(`%s
PIX/pagamento OK
Valor: R$ %s
Quem: %s
Status: %s
%s%s
==================`, header, d.Valor, d.Quem, d.Status, data, cod)
	case 4:
		return fmt.Sprintf(`%s
*Confirmado*
*%s* pagou *R$ %s* (%s)
Horário: %s%s
==================`, header, d.Quem, d.Valor, d.Meio, data, cod)
	default:
		return fmt.Sprintf(`%s
*VOUCHER PAGO*
Valor: R$ %s
Quem: %s
Data: %s
Meio: %s
Status: %s%s
==================`, header, d.Valor, d.Quem, data, d.Meio, d.Status, cod)
	}
}

func renderVoucherBaixa(v int, header string, d WhatsAppTemplateDados, data string) string {
	cod := ""
	if strings.TrimSpace(d.Codigo) != "" {
		cod = "\nCódigo: " + strings.TrimSpace(d.Codigo)
	}
	op := strings.TrimSpace(d.Extra)
	if op != "" {
		op = "\nOperador: " + op
	}
	switch v {
	case 1:
		return fmt.Sprintf(`%s
*Baixa no posto*
R$ %s · %s
%s%s%s
==================`, header, d.Valor, d.Quem, data, cod, op)
	case 2:
		return fmt.Sprintf(`%s
*VOUCHER UTILIZADO*
Cliente: *%s*
Valor: *R$ %s*
%s%s%s
==================`, header, d.Quem, d.Valor, data, cod, op)
	case 3:
		return fmt.Sprintf(`%s
Baixa registrada
• R$ %s
• %s
• %s%s%s
==================`, header, d.Valor, d.Quem, data, cod, op)
	case 4:
		return fmt.Sprintf(`%s
*%s*
%s usou voucher de R$ %s
%s%s%s
==================`, header, "BAIXA EFETUADA", d.Quem, d.Valor, data, cod, op)
	default:
		return fmt.Sprintf(`%s
*VOUCHER BAIXA*
Valor: R$ %s
Quem: %s
Data: %s
Status: %s%s%s
==================`, header, d.Valor, d.Quem, data, d.Status, cod, op)
	}
}

func renderCampanhaCriada(v int, header string, d WhatsAppTemplateDados, data string) string {
	nome := strings.TrimSpace(d.Titulo)
	if nome == "" {
		nome = "Campanha"
	}
	switch v {
	case 1:
		return fmt.Sprintf(`%s
*Nova campanha*
*%s*
Criada em %s
Status: %s
==================`, header, nome, data, d.Status)
	case 2:
		return fmt.Sprintf(`%s
CAMPANHA CRIADA
Nome: *%s*
%s · %s
==================`, header, nome, data, d.Status)
	case 3:
		return fmt.Sprintf(`%s
*Cadastro de campanha*
• %s
• Quando: %s
• Situação: %s
==================`, header, nome, data, d.Status)
	case 4:
		return fmt.Sprintf(`%s
*%s* entrou no sistema
Data: %s | Status: %s
==================`, header, nome, data, d.Status)
	default:
		return fmt.Sprintf(`%s
*CAMPANHA CRIADA*
Nome: %s
Data: %s
Status: %s
==================`, header, nome, data, d.Status)
	}
}

func renderCampanhaAtivada(v int, header string, d WhatsAppTemplateDados, data string) string {
	nome := strings.TrimSpace(d.Titulo)
	if nome == "" {
		nome = "Campanha"
	}
	switch v {
	case 1:
		return fmt.Sprintf(`%s
*Campanha ativada*
*%s* agora está *ATIVA*
%s
==================`, header, nome, data)
	case 2:
		return fmt.Sprintf(`%s
ATIVAÇÃO
Campanha *%s* liberada
Horário: %s
==================`, header, nome, data)
	case 3:
		return fmt.Sprintf(`%s
*Status → ATIVA*
%s
Atualizado em %s
==================`, header, nome, data)
	case 4:
		return fmt.Sprintf(`%s
*%s*
Passou a ATIVA em %s
==================`, header, nome, data)
	default:
		return fmt.Sprintf(`%s
*CAMPANHA ATIVADA*
Nome: %s
Data: %s
Status: ATIVA
==================`, header, nome, data)
	}
}
