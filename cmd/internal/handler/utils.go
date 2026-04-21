package handler

import (
	"encoding/json"
	"net/http"
	"time"
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

type requiredField struct {
	name  string
	value *string
}

func validateRequiredFields(w http.ResponseWriter, fields ...requiredField) bool {
	for _, field := range fields {
		if !isStringValid(field.value) {
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage(field.name))
			return false
		}
	}
	return true
}

func validateRFC3339Field(w http.ResponseWriter, field string, value *string) bool {
	if _, err := time.Parse(time.RFC3339, *value); err != nil {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage(field))
		return false
	}
	return true
}
