package handler

import (
	"encoding/json"
	"net/http"
)

func writeJSONMessage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func invalidFieldMessage(field string) string {
	return "invalid \"" + field + "\" field"
}

func invalidParamMessage(field string) string {
	return "invalid \"" + field + "\" parameter"
}

func isStringValid(s *string) bool {
	if s == nil {
		return false
	}
	return *s != ""
}
