package servicos

import (
	"errors"
	"fmt"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/utils"
)

type ServicoGestorRede interface {
	Listar() ([]*modelos.GestorRede, error)
	CriarComPlano(req CriarGestorComPlanoInput) (*modelos.GestorRede, []string, error)
	EditarComPlano(req EditarGestorComPlanoInput) (*modelos.GestorRede, []string, error)
	Contar() (total int, ativos int, err error)
	Login(email, senha string) (string, *modelos.UsuarioSessao, error)
	ExisteGestorPorEmail(email string) (bool, error)
}

type CriarGestorComPlanoInput struct {
	IDRede         string
	Nome           string
	Email          string
	Senha          string
	ConfirmarSenha string
	Telefone       string
}

type EditarGestorComPlanoInput struct {
	ID             string
	Nome           string
	Email          string
	Telefone       string
	Ativo          bool
	Senha          string
	ConfirmarSenha string
}

type servicoGestorRede struct {
	repoGestor repositorios.GestorRedeRepositorio
	repoRede   repositorios.RedeRepositorio
	auth       *autenticadorToken
}

func NovoServicoGestorRede(repoGestor repositorios.GestorRedeRepositorio, repoRede repositorios.RedeRepositorio, auth Autenticador) (ServicoGestorRede, error) {
	authToken, ok := auth.(*autenticadorToken)
	if !ok {
		return nil, errors.New("autenticador invalido para servico de gestor da rede")
	}
	return &servicoGestorRede{
		repoGestor: repoGestor,
		repoRede:   repoRede,
		auth:       authToken,
	}, nil
}

func (s *servicoGestorRede) Listar() ([]*modelos.GestorRede, error) {
	return s.repoGestor.Listar()
}

func (s *servicoGestorRede) CriarComPlano(req CriarGestorComPlanoInput) (*modelos.GestorRede, []string, error) {
	req.Nome = strings.TrimSpace(req.Nome)
	req.Email = strings.TrimSpace(req.Email)
	req.Senha = strings.TrimSpace(req.Senha)
	req.ConfirmarSenha = strings.TrimSpace(req.ConfirmarSenha)
	req.Telefone = strings.TrimSpace(req.Telefone)
	req.IDRede = strings.TrimSpace(req.IDRede)

	if req.IDRede == "" || req.Nome == "" || req.Email == "" || req.Senha == "" || req.ConfirmarSenha == "" {
		return nil, nil, ErrDadosInvalidos
	}
	if len(req.Senha) < 6 {
		return nil, nil, fmt.Errorf("%w: senha deve ter no minimo 6 caracteres", ErrDadosInvalidos)
	}
	if req.Senha != req.ConfirmarSenha {
		return nil, nil, fmt.Errorf("%w: senha e confirmar_senha devem ser iguais", ErrDadosInvalidos)
	}

	rede, err := s.repoRede.BuscarPorID(req.IDRede)
	if err != nil {
		return nil, nil, err
	}

	gestor := &modelos.GestorRede{
		ID:                 utils.GerarToken("gestor"),
		IDRede:             req.IDRede,
		Nome:               req.Nome,
		Email:              req.Email,
		SenhaHash:          utils.GerarHashSHA256(req.Senha),
		Telefone:           req.Telefone,
		Ativo:              true,
		ValorImplantacao:   rede.ValorImplantacao,
		ValorMensalidade:   rede.ValorMensalidade,
		PrimeiroVencimento: rede.PrimeiroCobranca,
		DiaVencimento:      rede.DiaCobranca,
	}

	if err := s.repoGestor.Criar(gestor); err != nil {
		return nil, nil, err
	}

	return gestor, utils.ProximosVencimentosMensais(gestor.PrimeiroVencimento, 6), nil
}

func (s *servicoGestorRede) EditarComPlano(req EditarGestorComPlanoInput) (*modelos.GestorRede, []string, error) {
	req.ID = strings.TrimSpace(req.ID)
	req.Nome = strings.TrimSpace(req.Nome)
	req.Email = strings.TrimSpace(req.Email)
	req.Telefone = strings.TrimSpace(req.Telefone)
	req.Senha = strings.TrimSpace(req.Senha)
	req.ConfirmarSenha = strings.TrimSpace(req.ConfirmarSenha)

	if req.ID == "" || req.Nome == "" || req.Email == "" {
		return nil, nil, ErrDadosInvalidos
	}

	var novaSenhaHash string
	switch {
	case req.Senha == "" && req.ConfirmarSenha == "":
		break
	case req.Senha == "" || req.ConfirmarSenha == "":
		return nil, nil, fmt.Errorf("%w: informe senha e confirmar_senha para alterar a senha", ErrDadosInvalidos)
	default:
		if len(req.Senha) < 6 {
			return nil, nil, fmt.Errorf("%w: senha deve ter no minimo 6 caracteres", ErrDadosInvalidos)
		}
		if req.Senha != req.ConfirmarSenha {
			return nil, nil, fmt.Errorf("%w: senha e confirmar_senha devem ser iguais", ErrDadosInvalidos)
		}
		novaSenhaHash = utils.GerarHashSHA256(req.Senha)
	}

	gestor, err := s.repoGestor.Atualizar(req.ID, func(g *modelos.GestorRede) error {
		g.Nome = req.Nome
		g.Email = req.Email
		g.Telefone = req.Telefone
		g.Ativo = req.Ativo
		if novaSenhaHash != "" {
			g.NovaSenhaHash = novaSenhaHash
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	var proximos []string
	if !gestor.PrimeiroVencimento.IsZero() {
		proximos = utils.ProximosVencimentosMensais(gestor.PrimeiroVencimento, 6)
	}
	return gestor, proximos, nil
}

func (s *servicoGestorRede) Contar() (int, int, error) {
	return s.repoGestor.Contar()
}

func (s *servicoGestorRede) ExisteGestorPorEmail(email string) (bool, error) {
	return s.repoGestor.ExisteGestorPorEmail(strings.TrimSpace(email))
}

func (s *servicoGestorRede) Login(email, senha string) (string, *modelos.UsuarioSessao, error) {
	email = strings.TrimSpace(email)
	senha = strings.TrimSpace(senha)
	if email == "" || senha == "" {
		return "", nil, ErrDadosInvalidos
	}

	gestor, err := s.repoGestor.BuscarPorEmailParaLogin(email)
	if err != nil {
		if errors.Is(err, repositorios.ErrGestorNaoEncontrado) {
			return "", nil, ErrCredenciais
		}
		return "", nil, err
	}

	if !gestor.Ativo || gestor.SenhaHash != utils.GerarHashSHA256(senha) {
		return "", nil, ErrCredenciais
	}

	sessao := &modelos.UsuarioSessao{
		IDUsuario:    gestor.ID,
		NomeCompleto: gestor.Nome,
		IDRede:       gestor.IDRede,
		Papel:        modelos.PapelGestorRede,
	}

	token := s.auth.CriarSessao(sessao)
	return token, sessao, nil
}
