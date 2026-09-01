// Package apiutil reúne helpers de JSON HTTP sem depender de nenhum pacote
// de domínio. Fica separado de httpserver deliberadamente: httpserver
// importa todos os pacotes de domínio (para montar as rotas), então se
// esses helpers vivessem lá, todo handler que os usasse criaria um import
// cycle (domínio → httpserver → domínio).
package apiutil

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
