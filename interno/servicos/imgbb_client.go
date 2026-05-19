package servicos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const imgbbUploadURL = "https://api.imgbb.com/1/upload"

// ImgBBUploadResult URLs retornadas pela API v1 (https://api.imgbb.com).
type ImgBBUploadResult struct {
	ID         string
	URL        string
	DisplayURL string
	ThumbURL   string
}

type imgbbResponse struct {
	Data      *imgbbData `json:"data"`
	Success   bool       `json:"success"`
	Status    int        `json:"status"`
	Error     imgbbError `json:"error"`
	StatusTxt string     `json:"status_txt"`
}

type imgbbData struct {
	ID         string       `json:"id"`
	URL        string       `json:"url"`
	DisplayURL string       `json:"display_url"`
	Thumb      *imgbbNested `json:"thumb"`
	Image      *imgbbNested `json:"image"`
}

type imgbbNested struct {
	URL string `json:"url"`
}

type imgbbError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// UploadImageToImgBB envia o arquivo no campo multipart "image" (POST). A chave vai na query string.
func UploadImageToImgBB(apiKey string, filename string, file io.Reader) (*ImgBBUploadResult, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, fmt.Errorf("API ImgBB não configurada (defina API_IMGBB_KEY)")
	}

	var body bytes.Buffer
	mp := multipart.NewWriter(&body)
	part, err := mp.CreateFormFile("image", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := mp.Close(); err != nil {
		return nil, err
	}

	endpoint, err := url.Parse(imgbbUploadURL)
	if err != nil {
		return nil, err
	}
	q := endpoint.Query()
	q.Set("key", key)
	endpoint.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodPost, endpoint.String(), &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mp.FormDataContentType())

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ImgBB: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	var env imgbbResponse
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("ImgBB: resposta inválida: %w", err)
	}
	if !env.Success || env.Data == nil {
		msg := strings.TrimSpace(env.Error.Message)
		if msg == "" {
			msg = strings.TrimSpace(env.StatusTxt)
		}
		if msg == "" {
			msg = string(respBody)
		}
		return nil, fmt.Errorf("ImgBB: %s", imgbbTruncate(msg, 500))
	}

	d := env.Data
	out := &ImgBBUploadResult{ID: d.ID}
	if d.DisplayURL != "" {
		out.DisplayURL = d.DisplayURL
	} else if d.URL != "" {
		out.DisplayURL = d.URL
	}
	if d.Image != nil && d.Image.URL != "" {
		out.URL = d.Image.URL
	} else {
		out.URL = d.URL
	}
	if d.Thumb != nil {
		out.ThumbURL = d.Thumb.URL
	}
	if out.URL == "" {
		out.URL = out.DisplayURL
	}
	if out.DisplayURL == "" {
		out.DisplayURL = out.URL
	}
	if out.URL == "" {
		return nil, fmt.Errorf("ImgBB: resposta sem URL de imagem")
	}
	return out, nil
}

func imgbbTruncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
