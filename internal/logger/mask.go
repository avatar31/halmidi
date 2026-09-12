package logger

import (
	"reflect"
	"strings"
)

var DefaultMaskedFields = []string{"password", "token", "secret", "access_token", "refresh_token", "api_key", "pin"}

const MaskValue = "***"

// Mask attempts to mask sensitive fields in a struct or map.
// Supports nested structures.
func Mask(input any, fieldsToMask ...string) any {
	if len(fieldsToMask) == 0 {
		fieldsToMask = DefaultMaskedFields
	}
	return maskValue(reflect.ValueOf(input), fieldsToMask)
}

func maskValue(val reflect.Value, fieldsToMask []string) any {
	if !val.IsValid() {
		return nil
	}

	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		return maskValue(val.Elem(), fieldsToMask)

	case reflect.Struct:
		masked := make(map[string]any)
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" { // unexported
				continue
			}
			jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
			if jsonTag == "" {
				jsonTag = field.Name
			}
			if field.Tag.Get("mask") == "true" || isFieldSensitive(jsonTag, fieldsToMask) {
				masked[jsonTag] = MaskValue
			} else {
				masked[jsonTag] = maskValue(val.Field(i), fieldsToMask)
			}
		}
		return masked

	case reflect.Map:
		if val.Type().Key().Kind() != reflect.String {
			return val.Interface() // unsupported map key
		}
		masked := make(map[string]any)
		iter := val.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			value := iter.Value()
			if isFieldSensitive(key, fieldsToMask) {
				masked[key] = MaskValue
			} else {
				masked[key] = maskValue(value, fieldsToMask)
			}
		}
		return masked

	case reflect.Slice, reflect.Array:
		result := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = maskValue(val.Index(i), fieldsToMask)
		}
		return result

	default:
		return val.Interface()
	}
}

func isFieldSensitive(field string, fields []string) bool {
	field = strings.ToLower(field)
	for _, target := range fields {
		if strings.EqualFold(field, target) {
			return true
		}
	}
	return false
}

// func main() {
// 	req := LoginRequest{
// 		Username: "admin",
// 		Password: "secret123",
// 		Token:    "abc-123-token",
// 	}

// 	log.Info().
// 		Interface("login", masking.Mask(req)).
// 		Msg("Incoming login request")

// 	payload := map[string]any{
// 		"username": "john",
// 		"password": "secret",
// 		"profile": map[string]any{
// 			"token": "nested-token",
// 		},
// 	}

// 	log.Info().
// 		Interface("payload", masking.Mask(payload)).
// 		Msg("Sanitized payload")
// }
