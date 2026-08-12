package servicos

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/utils"
)

const (
	usuarioRedeLimitePadrao  = 20
	usuarioRedeLimiteMaximo  = 100
	usuarioRedeLimiteMinimo  = 1
)

// ErrPresencaClienteNaoAplicavel conta nao e cliente na rede ou nao existe.
var ErrPresencaClienteNaoAplicavel = errors.New("presenca nao aplicavel a esta conta")

type ServicoUsuarioRede interface {
	ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, int, int, error)
	CriarUsuarioEquipe(in CriarUsuarioEquipeInput) (*modelos.UsuarioVinculoRede, error)
	EditarUsuarioEquipe(in EditarUsuarioEquipeInput) (*modelos.UsuarioVinculoRede, error)
	LoginPainel(email, senha string) (string, *modelos.UsuarioSessao, error)
	LoginPainelNaRede(email, senha, idRede string) (string, *modelos.UsuarioSessao, error)
	LoginPainelPorCodigo(codigo, senha, idRede string) (string, *modelos.UsuarioSessao, error)
	CadastrarClienteApp(in CadastroClienteAppInput) (string, *modelos.UsuarioSessao, error)
	ExcluirContaClienteApp(idUsuario, idRede string) error
	// EmailECPFPorUsuarioRede e-mail e CPF cadastrados (app / pagamento).
	EmailECPFPorUsuarioRede(idUsuario, idRede string) (email string, cpf string, err error)
	// ObterNivelCliente codigo do nivel (ex. bronze) para multiplicador de moeda no app.
	ObterNivelCliente(idUsuario, idRede string) (string, error)
	// RegistrarTokenFCM grava o token do Firebase Cloud Messaging (push) para o utilizador.
	RegistrarTokenFCM(idUsuario, token, plataforma string) error
	// ListarTokensFCM tokens guardados (para teste de push / envio).
	ListarTokensFCM(idUsuario string) ([]string, error)
	// ListarTokensFCMClientesRede tokens FCM de todos os clientes ativos da rede (push em massa).
	ListarTokensFCMClientesRede(idRede string) ([]string, error)
	DiagnosticoPushRede(idRede string) (*repositorios.DiagnosticoPushRedeStats, error)
	// RemoverTokensFCM apaga tokens invalidos (NotRegistered / SenderId mismatch, etc.).
	RemoverTokensFCM(tokens []string) (int64, error)
	// RemoverTokensFCMClientes apaga tokens FCM de clientes; idRede vazio = todas as redes.
	RemoverTokensFCMClientes(idRede string) (int64, error)
	// RegistrarPresencaAppCliente heartbeat do app (cliente): atualiza ultima atividade.
	RegistrarPresencaAppCliente(idUsuario, idRede, plataforma string) error
	// ListarPresencaClientesRede clientes com dados cadastrais e ultimo app heartbeat (painel gestor/gerente).
	ListarPresencaClientesRede(idRede string, limite, minutosOnline int) (totalClientes, totalComPresenca int, itens []repositorios.ClientePresencaAppItem, err error)
}

// CriarUsuarioEquipeInput cadastro de gerente de posto ou frentista pelo admin global.
type CriarUsuarioEquipeInput struct {
	IDRede         string
	IDPosto        string
	Papel          string
	Nome           string
	Email          string
	Codigo         string
	Senha          string
	ConfirmarSenha string
	Telefone       string
}

// CadastroClienteAppInput cadastro público de cliente final (app mobile) na rede.
type CadastroClienteAppInput struct {
	IDRede         string
	NomeCompleto   string
	Email          string
	Senha          string
	ConfirmarSenha string
	Telefone       string
	CPF            string
	// CodigoIndicacao opcional: codigo de outro cliente (indique e ganhe).
	CodigoIndicacao string
}

// EditarUsuarioEquipeInput atualizacao de gerente ou frentista; senhas vazias mantem a senha atual.
type EditarUsuarioEquipeInput struct {
	IDRede         string
	IDUsuario      string
	IDPosto        string
	Papel          string
	Nome           string
	Email          string
	Codigo         string
	Senha          string
	ConfirmarSenha string
	Telefone       string
	Ativo          bool
}

var papeisEquipePosto = map[string]struct{}{
	"gerente_posto": {},
	"frentista":     {},
}

type usuarioRedePostgresRepo interface {
	ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, error)
	CriarUsuarioEquipe(idRede, idPosto, papel, nome, email, senhaHash, telefone, codigoAcesso string) (*modelos.UsuarioVinculoRede, error)
	CriarClienteSelfCadastro(idRede, nome, email, senhaHash, telefone, cpf string) (*modelos.UsuarioVinculoRede, error)
	ExcluirContaClientePorID(idUsuario, idRede string) error
	AtualizarUsuarioEquipe(idRede, idUsuario string, nome, email, telefone string, ativo bool, papel, idPosto, senhaHashOuVazio, codigoAcesso string) (*modelos.UsuarioVinculoRede, error)
	BuscarPorEmailParaLoginPainel(email string) (*repositorios.UsuarioPainelLogin, error)
	BuscarPorEmailParaLoginPainelNaRede(idRede, email string) (*repositorios.UsuarioPainelLogin, error)
	ListarFrentistasPorCodigoAcesso(codigo, idRede string) ([]*repositorios.UsuarioPainelLogin, error)
	BuscarFrentistaAtivoPorCodigoNoPosto(idRede, idPosto, codigo string) (*repositorios.UsuarioPainelLogin, error)
	PostoPertenceARede(idPosto, idRede string) (bool, error)
	EmailECPFPorUsuarioRede(idUsuario, idRede string) (email string, cpf string, err error)
	ObterNivelCliente(idUsuario, idRede string) (string, error)
	UpsertFCMToken(idUsuario, token, plataforma string) error
	ListarTokensFCMPorUsuarioID(idUsuario string) ([]string, error)
	ListarTokensFCMPorRedeClientesAtivos(idRede string) ([]string, error)
	RemoverTokensFCM(tokens []string) (int64, error)
	RemoverTokensFCMClientes(idRede string) (int64, error)
	DiagnosticoPushRede(idRede string) (*repositorios.DiagnosticoPushRedeStats, error)
	DefinirCodigoIndicacao(idUsuario, idRede, codigo string) error
	ObterCodigoIndicacao(idUsuario, idRede string) (string, error)
	BuscarIdClientePorCodigoIndicacao(idRede, codigo string) (string, error)
	RegistrarPresencaAppCliente(idUsuario, idRede, plataforma string) error
	ListarClientesPresencaAppPorRede(idRede string, limite, minutosOnline int) (totalClientes, totalComPresenca int, itens []repositorios.ClientePresencaAppItem, err error)
}

type servicoUsuarioRede struct {
	repoUsuarios  usuarioRedePostgresRepo
	repoRede      repositorios.RedeRepositorio
	auth          *autenticadorToken
	indiqueGanhe  *ServicoIndiqueGanhe
}

func NovoServicoUsuarioRede(
	repoUsuarios usuarioRedePostgresRepo,
	repoRede repositorios.RedeRepositorio,
	auth Autenticador,
	indiqueGanhe *ServicoIndiqueGanhe,
) (ServicoUsuarioRede, error) {
	authToken, ok := auth.(*autenticadorToken)
	if !ok {
		return nil, errors.New("autenticador invalido para servico de usuario da rede")
	}
	return &servicoUsuarioRede{
		repoUsuarios: repoUsuarios,
		repoRede:     repoRede,
		auth:         authToken,
		indiqueGanhe: indiqueGanhe,
	}, nil
}

func (s *servicoUsuarioRede) ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, int, int, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, 0, 0, 0, ErrDadosInvalidos
	}
	if limite < usuarioRedeLimiteMinimo {
		limite = usuarioRedeLimitePadrao
	}
	if limite > usuarioRedeLimiteMaximo {
		limite = usuarioRedeLimiteMaximo
	}
	if offset < 0 {
		offset = 0
	}
	papeis := repositorios.SanitizarPapeisFiltro(papeisFiltro)
	idPostoFiltro = strings.TrimSpace(idPostoFiltro)
	if _, err := s.repoRede.BuscarPorID(idRede); err != nil {
		return nil, 0, 0, 0, err
	}
	itens, total, err := s.repoUsuarios.ListarPorRedeIDPaginado(idRede, limite, offset, papeis, idPostoFiltro)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	return itens, total, limite, offset, nil
}

func (s *servicoUsuarioRede) CriarUsuarioEquipe(in CriarUsuarioEquipeInput) (*modelos.UsuarioVinculoRede, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.IDPosto = strings.TrimSpace(in.IDPosto)
	in.Papel = strings.TrimSpace(in.Papel)
	in.Nome = strings.TrimSpace(in.Nome)
	in.Email = strings.TrimSpace(in.Email)
	in.Codigo = strings.TrimSpace(in.Codigo)
	in.Senha = strings.TrimSpace(in.Senha)
	in.ConfirmarSenha = strings.TrimSpace(in.ConfirmarSenha)
	in.Telefone = strings.TrimSpace(in.Telefone)

	if in.IDRede == "" || in.IDPosto == "" || in.Nome == "" || in.Senha == "" || in.Papel == "" {
		return nil, ErrDadosInvalidos
	}
	if _, ok := papeisEquipePosto[in.Papel]; !ok {
		return nil, fmt.Errorf("%w: papel deve ser gerente_posto ou frentista", ErrDadosInvalidos)
	}
	if in.Papel == "frentista" {
		if in.Codigo == "" {
			return nil, fmt.Errorf("%w: codigo de acesso obrigatorio para frentista", ErrDadosInvalidos)
		}
	} else if in.Email == "" {
		return nil, fmt.Errorf("%w: email obrigatorio para gerente de posto", ErrDadosInvalidos)
	}
	if in.Senha != in.ConfirmarSenha {
		return nil, fmt.Errorf("%w: senha e confirmar_senha devem ser iguais", ErrDadosInvalidos)
	}
	if len(in.Senha) < 6 {
		return nil, fmt.Errorf("%w: senha deve ter no minimo 6 caracteres", ErrDadosInvalidos)
	}
	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return nil, err
	}
	ok, err := s.repoUsuarios.PostoPertenceARede(in.IDPosto, in.IDRede)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, repositorios.ErrPostoNaoPertenceARede
	}

	codigo := ""
	if in.Papel == "frentista" {
		codigo = in.Codigo
	}

	return s.repoUsuarios.CriarUsuarioEquipe(
		in.IDRede,
		in.IDPosto,
		in.Papel,
		in.Nome,
		in.Email,
		utils.GerarHashSHA256(in.Senha),
		in.Telefone,
		codigo,
	)
}

func (s *servicoUsuarioRede) EditarUsuarioEquipe(in EditarUsuarioEquipeInput) (*modelos.UsuarioVinculoRede, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.IDUsuario = strings.TrimSpace(in.IDUsuario)
	in.IDPosto = strings.TrimSpace(in.IDPosto)
	in.Papel = strings.TrimSpace(in.Papel)
	in.Nome = strings.TrimSpace(in.Nome)
	in.Email = strings.TrimSpace(in.Email)
	in.Codigo = strings.TrimSpace(in.Codigo)
	in.Senha = strings.TrimSpace(in.Senha)
	in.ConfirmarSenha = strings.TrimSpace(in.ConfirmarSenha)
	in.Telefone = strings.TrimSpace(in.Telefone)

	if in.IDRede == "" || in.IDUsuario == "" || in.IDPosto == "" || in.Nome == "" || in.Papel == "" {
		return nil, ErrDadosInvalidos
	}
	if _, ok := papeisEquipePosto[in.Papel]; !ok {
		return nil, fmt.Errorf("%w: papel deve ser gerente_posto ou frentista", ErrDadosInvalidos)
	}
	if in.Papel == "frentista" {
		if in.Codigo == "" {
			return nil, fmt.Errorf("%w: codigo de acesso obrigatorio para frentista", ErrDadosInvalidos)
		}
	} else if in.Email == "" {
		return nil, fmt.Errorf("%w: email obrigatorio para gerente de posto", ErrDadosInvalidos)
	}
	if in.Senha != "" || in.ConfirmarSenha != "" {
		if in.Senha != in.ConfirmarSenha {
			return nil, fmt.Errorf("%w: senha e confirmar_senha devem ser iguais", ErrDadosInvalidos)
		}
		if len(in.Senha) < 6 {
			return nil, fmt.Errorf("%w: senha deve ter no minimo 6 caracteres", ErrDadosInvalidos)
		}
	}
	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return nil, err
	}

	senhaHash := ""
	if in.Senha != "" {
		senhaHash = utils.GerarHashSHA256(in.Senha)
	}
	codigo := ""
	if in.Papel == "frentista" {
		codigo = in.Codigo
	}
	u, err := s.repoUsuarios.AtualizarUsuarioEquipe(
		in.IDRede,
		in.IDUsuario,
		in.Nome,
		in.Email,
		in.Telefone,
		in.Ativo,
		in.Papel,
		in.IDPosto,
		senhaHash,
		codigo,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *servicoUsuarioRede) LoginPainel(email, senha string) (string, *modelos.UsuarioSessao, error) {
	email = strings.TrimSpace(email)
	senha = strings.TrimSpace(senha)
	if email == "" || senha == "" {
		return "", nil, ErrDadosInvalidos
	}

	u, err := s.repoUsuarios.BuscarPorEmailParaLoginPainel(email)
	if err != nil {
		if errors.Is(err, repositorios.ErrUsuarioPainelLoginNaoEncontrado) {
			return "", nil, ErrCredenciais
		}
		return "", nil, err
	}
	if !u.Ativo || u.SenhaHash != utils.GerarHashSHA256(senha) {
		return "", nil, ErrCredenciais
	}

	p := modelos.Papel(strings.TrimSpace(u.Papel))
	sessao := &modelos.UsuarioSessao{
		IDUsuario:    u.ID,
		NomeCompleto: u.Nome,
		IDRede:       u.IDRede,
		IDPosto:      u.IDPosto,
		Papel:        p,
	}
	token := s.auth.CriarSessao(sessao)
	return token, sessao, nil
}

func (s *servicoUsuarioRede) LoginPainelNaRede(email, senha, idRede string) (string, *modelos.UsuarioSessao, error) {
	email = strings.TrimSpace(email)
	senha = strings.TrimSpace(senha)
	idRede = strings.TrimSpace(idRede)
	if email == "" || senha == "" || idRede == "" {
		return "", nil, ErrDadosInvalidos
	}

	u, err := s.repoUsuarios.BuscarPorEmailParaLoginPainelNaRede(idRede, email)
	if err != nil {
		if errors.Is(err, repositorios.ErrUsuarioPainelLoginNaoEncontrado) {
			return "", nil, ErrCredenciais
		}
		return "", nil, err
	}
	if !u.Ativo || u.SenhaHash != utils.GerarHashSHA256(senha) {
		return "", nil, ErrCredenciais
	}

	p := modelos.Papel(strings.TrimSpace(u.Papel))
	sessao := &modelos.UsuarioSessao{
		IDUsuario:    u.ID,
		NomeCompleto: u.Nome,
		IDRede:       u.IDRede,
		IDPosto:      u.IDPosto,
		Papel:        p,
	}
	token := s.auth.CriarSessao(sessao)
	return token, sessao, nil
}

// ErrCodigoAcessoAmbiguo quando o mesmo codigo+senha bate em mais de um posto.
var ErrCodigoAcessoAmbiguo = errors.New("codigo de acesso ambiguo em mais de um posto; use o email")

// LoginPainelPorCodigo autentica frentista pelo codigo de acesso (+ senha).
func (s *servicoUsuarioRede) LoginPainelPorCodigo(codigo, senha, idRede string) (string, *modelos.UsuarioSessao, error) {
	codigo = strings.TrimSpace(codigo)
	senha = strings.TrimSpace(senha)
	idRede = strings.TrimSpace(idRede)
	if codigo == "" || senha == "" {
		return "", nil, ErrDadosInvalidos
	}

	lista, err := s.repoUsuarios.ListarFrentistasPorCodigoAcesso(codigo, idRede)
	if err != nil {
		return "", nil, err
	}
	if len(lista) == 0 {
		return "", nil, ErrCredenciais
	}

	hash := utils.GerarHashSHA256(senha)
	var matches []*repositorios.UsuarioPainelLogin
	for _, u := range lista {
		if u == nil || !u.Ativo {
			continue
		}
		if u.SenhaHash == hash {
			matches = append(matches, u)
		}
	}
	if len(matches) == 0 {
		return "", nil, ErrCredenciais
	}
	if len(matches) > 1 {
		return "", nil, ErrCodigoAcessoAmbiguo
	}

	u := matches[0]
	p := modelos.Papel(strings.TrimSpace(u.Papel))
	sessao := &modelos.UsuarioSessao{
		IDUsuario:    u.ID,
		NomeCompleto: u.Nome,
		IDRede:       u.IDRede,
		IDPosto:      u.IDPosto,
		Papel:        p,
	}
	token := s.auth.CriarSessao(sessao)
	return token, sessao, nil
}

func (s *servicoUsuarioRede) CadastrarClienteApp(in CadastroClienteAppInput) (string, *modelos.UsuarioSessao, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.NomeCompleto = strings.TrimSpace(in.NomeCompleto)
	in.Email = strings.TrimSpace(in.Email)
	in.Senha = strings.TrimSpace(in.Senha)
	in.ConfirmarSenha = strings.TrimSpace(in.ConfirmarSenha)
	in.Telefone = strings.TrimSpace(in.Telefone)
	in.CPF = utils.SomenteDigitosCPF(in.CPF)

	if in.IDRede == "" || in.NomeCompleto == "" || in.Email == "" || in.Senha == "" {
		return "", nil, ErrDadosInvalidos
	}
	if in.CPF == "" {
		return "", nil, fmt.Errorf("%w: cpf e obrigatorio", ErrDadosInvalidos)
	}
	if !utils.ValidarCPF(in.CPF) {
		return "", nil, fmt.Errorf("%w: cpf invalido", ErrDadosInvalidos)
	}
	if in.Senha != in.ConfirmarSenha {
		return "", nil, fmt.Errorf("%w: senha e confirmar_senha devem ser iguais", ErrDadosInvalidos)
	}
	if len(in.Senha) < 6 {
		return "", nil, fmt.Errorf("%w: senha deve ter no minimo 6 caracteres", ErrDadosInvalidos)
	}
	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return "", nil, err
	}

	u, err := s.repoUsuarios.CriarClienteSelfCadastro(
		in.IDRede,
		in.NomeCompleto,
		in.Email,
		utils.GerarHashSHA256(in.Senha),
		in.Telefone,
		in.CPF,
	)
	if err != nil {
		return "", nil, err
	}

	sessao := &modelos.UsuarioSessao{
		IDUsuario:    u.ID,
		NomeCompleto: u.Nome,
		IDRede:       u.IDRede,
		IDPosto:      u.IDPosto,
		Papel:        modelos.PapelCliente,
	}
	token := s.auth.CriarSessao(sessao)
	if s.indiqueGanhe != nil {
		s.indiqueGanhe.AposNovoCadastro(in.IDRede, u.ID, in.CodigoIndicacao)
	}
	return token, sessao, nil
}

func (s *servicoUsuarioRede) ExcluirContaClienteApp(idUsuario, idRede string) error {
	idUsuario = strings.TrimSpace(idUsuario)
	idRede = strings.TrimSpace(idRede)
	if idUsuario == "" || idRede == "" {
		return ErrDadosInvalidos
	}
	return s.repoUsuarios.ExcluirContaClientePorID(idUsuario, idRede)
}

func (s *servicoUsuarioRede) EmailECPFPorUsuarioRede(idUsuario, idRede string) (email string, cpf string, err error) {
	return s.repoUsuarios.EmailECPFPorUsuarioRede(idUsuario, idRede)
}

func (s *servicoUsuarioRede) ObterNivelCliente(idUsuario, idRede string) (string, error) {
	return s.repoUsuarios.ObterNivelCliente(idUsuario, idRede)
}

func (s *servicoUsuarioRede) RegistrarTokenFCM(idUsuario, token, plataforma string) error {
	idUsuario = strings.TrimSpace(idUsuario)
	if idUsuario == "" {
		return ErrDadosInvalidos
	}
	if plataforma == "" {
		plataforma = "android"
	}
	return s.repoUsuarios.UpsertFCMToken(idUsuario, token, plataforma)
}

func (s *servicoUsuarioRede) ListarTokensFCM(idUsuario string) ([]string, error) {
	return s.repoUsuarios.ListarTokensFCMPorUsuarioID(strings.TrimSpace(idUsuario))
}

func (s *servicoUsuarioRede) ListarTokensFCMClientesRede(idRede string) ([]string, error) {
	return s.repoUsuarios.ListarTokensFCMPorRedeClientesAtivos(strings.TrimSpace(idRede))
}

func (s *servicoUsuarioRede) DiagnosticoPushRede(idRede string) (*repositorios.DiagnosticoPushRedeStats, error) {
	return s.repoUsuarios.DiagnosticoPushRede(strings.TrimSpace(idRede))
}

func (s *servicoUsuarioRede) RemoverTokensFCM(tokens []string) (int64, error) {
	return s.repoUsuarios.RemoverTokensFCM(tokens)
}

func (s *servicoUsuarioRede) RemoverTokensFCMClientes(idRede string) (int64, error) {
	return s.repoUsuarios.RemoverTokensFCMClientes(idRede)
}

func (s *servicoUsuarioRede) RegistrarPresencaAppCliente(idUsuario, idRede, plataforma string) error {
	idUsuario = strings.TrimSpace(idUsuario)
	idRede = strings.TrimSpace(idRede)
	if idUsuario == "" || idRede == "" {
		return ErrDadosInvalidos
	}
	err := s.repoUsuarios.RegistrarPresencaAppCliente(idUsuario, idRede, plataforma)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPresencaClienteNaoAplicavel
	}
	return err
}

func (s *servicoUsuarioRede) ListarPresencaClientesRede(idRede string, limite, minutosOnline int) (totalClientes, totalComPresenca int, itens []repositorios.ClientePresencaAppItem, err error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return 0, 0, nil, ErrDadosInvalidos
	}
	return s.repoUsuarios.ListarClientesPresencaAppPorRede(idRede, limite, minutosOnline)
}
