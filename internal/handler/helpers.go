package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

type errBody struct {
	Error string `json:"error"`
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, errBody{Error: msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func urlID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func qStr(r *http.Request, k, fb string) string {
	if v := r.URL.Query().Get(k); v != "" {
		return v
	}
	return fb
}

func qInt(r *http.Request, k string, fb int) int {
	v := r.URL.Query().Get(k)
	if v == "" {
		return fb
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fb
	}
	return n
}

func qInt64Ptr(r *http.Request, k string) *int64 {
	v := r.URL.Query().Get(k)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func qBoolPtr(r *http.Request, k string) *bool {
	v := r.URL.Query().Get(k)
	if v == "" {
		return nil
	}
	b := v == "1" || v == "true"
	return &b
}

func isFKError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "FOREIGN KEY") || strings.Contains(msg, "foreign key")
}