package servicos

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"gaspass-servidor/interno/config"
)

// ErrUploadImagemNaoConfigurado quando API_IMGBB_KEY não está definida no servidor.
var ErrUploadImagemNaoConfigurado = errors.New("upload de imagens não configurado no servidor (defina API_IMGBB_KEY)")

const MaxImagemUploadBytes = 12 << 20

var mimeImagemPermitidos = map[string]struct{}{
	"image/jpeg": {}, "image/png": {}, "image/gif": {}, "image/webp": {},
}

type ServicoUploadImagem struct {
	cfg config.Config
}

func NovoServicoUploadImagem(cfg config.Config) *ServicoUploadImagem {
	return &ServicoUploadImagem{cfg: cfg}
}

// Upload valida tamanho/MIME e envia ao ImgBB. Retorna a URL pública (DisplayURL preferida).
func (s *ServicoUploadImagem) Upload(fh *multipart.FileHeader) (string, error) {
	if strings.TrimSpace(s.cfg.APIImgBBKey) == "" {
		return "", ErrUploadImagemNaoConfigurado
	}
	if fh == nil || fh.Size <= 0 {
		return "", errors.New("arquivo vazio")
	}
	if fh.Size > MaxImagemUploadBytes {
		return "", fmt.Errorf("arquivo muito grande (máximo %d MB)", MaxImagemUploadBytes>>20)
	}
	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	raw, err := io.ReadAll(io.LimitReader(src, MaxImagemUploadBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", errors.New("arquivo vazio")
	}
	if len(raw) > int(MaxImagemUploadBytes) {
		return "", fmt.Errorf("arquivo muito grande (máximo %d MB)", MaxImagemUploadBytes>>20)
	}
	mime := http.DetectContentType(raw)
	if _, ok := mimeImagemPermitidos[mime]; !ok {
		return "", fmt.Errorf("tipo não permitido: %s (use JPEG, PNG, GIF ou WebP)", mime)
	}
	name := strings.TrimSpace(fh.Filename)
	if name == "" {
		name = "imagem"
	}
	up, err := UploadImageToImgBB(s.cfg.APIImgBBKey, name, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	url := strings.TrimSpace(up.DisplayURL)
	if url == "" {
		url = strings.TrimSpace(up.URL)
	}
	if url == "" {
		return "", errors.New("upload falhou: sem URL retornada")
	}
	return url, nil
}
