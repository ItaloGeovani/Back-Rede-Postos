package handlers

import (
	"errors"
	"net/http"

	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// PostUploadImagem recebe multipart campo "file", envia ao ImgBB e devolve { "url": "..." }.
func (h *Handlers) PostUploadImagem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.uploadImagem == nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "servico de upload indisponivel")
		return
	}

	if err := r.ParseMultipartForm(servicos.MaxImagemUploadBytes + (1 << 20)); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "formulario invalido")
		return
	}
	_, fh, err := r.FormFile("file")
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "arquivo nao enviado")
		return
	}

	url, err := h.uploadImagem.Upload(fh)
	if err != nil {
		if errors.Is(err, servicos.ErrUploadImagemNaoConfigurado) {
			utils.ResponderErro(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]string{"url": url})
}
