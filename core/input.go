package kite

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

/**
 * jsonBody tenta decodificar o corpo da requisição formatado em JSON para um map.
 * Retorna nil quando o Content-Type não é JSON, o corpo está vazio ou ocorrem falhas de parsing.
 * O resultado dos bytes da requisição é reutilizado pelo método Body().
 *
 * @return map[string]interface{}
 */
func (r *Request) jsonBody() map[string]interface{} {
	if !strings.Contains(r.ContentType(), "json") {
		return nil
	}
	data, err := r.Body()
	if err != nil || len(data) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

/**
 * GetBody obtém os bytes brutos do corpo da requisição.
 * Atua como um método utilitário para a chamada nativa de Body().
 *
 * @return []byte
 * @return error
 */
func (r *Request) GetBody() ([]byte, error) {
	return r.Body()
}

/**
 * Input recupera o valor de um parâmetro inspecionando sequencialmente:
 * corpo JSON, dados de formulário (urlencoded/multipart) e parâmetros de query string.
 *
 * Exemplo:
 *  nome := req.Input("nome")
 *
 * @param name string
 * @return string
 */
func (r *Request) Input(name string) string {
	if m := r.jsonBody(); m != nil {
		if v, ok := m[name]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
	}
	if v := r.FormValue(name); v != "" {
		return v
	}
	return r.Query(name)
}

/**
 * InputDefault recupera o valor de um parâmetro via Input(), retornando o valor padrão (def)
 * caso o parâmetro não exista ou esteja vazio.
 *
 * @param name string
 * @param def string
 * @return string
 */
func (r *Request) InputDefault(name, def string) string {
	if v := r.Input(name); v != "" {
		return v
	}
	return def
}

/**
 * InputInt recupera o valor de um parâmetro via Input() e o converte para inteiro.
 * Retorna o valor padrão (def) se o parâmetro estiver ausente ou for inválido.
 *
 * @param name string
 * @param def int
 * @return int
 */
func (r *Request) InputInt(name string, def int) int {
	val, err := strconv.Atoi(r.Input(name))
	if err != nil {
		return def
	}
	return val
}

/**
 * InputBool recupera o valor de um parâmetro via Input() e o converte para booleano.
 * Aceita as representações "1", "true", "t", "yes", "y" como verdadeiro.
 *
 * @param name string
 * @param def bool
 * @return bool
 */
func (r *Request) InputBool(name string, def bool) bool {
	val := strings.ToLower(strings.TrimSpace(r.Input(name)))
	if val == "" {
		return def
	}
	switch val {
	case "1", "true", "t", "yes", "y":
		return true
	case "0", "false", "f", "no", "n":
		return false
	default:
		return def
	}
}

/**
 * Has verifica se um determinado parâmetro está presente na requisição (JSON, formulário ou query string),
 * independente do seu valor estar vazio ou não.
 *
 * @param name string
 * @return bool
 */
func (r *Request) Has(name string) bool {
	if m := r.jsonBody(); m != nil {
		if _, ok := m[name]; ok {
			return true
		}
	}
	if r.HasQuery(name) {
		return true
	}
	values, err := r.FormValues()
	if err == nil && values.Has(name) {
		return true
	}
	return false
}

/**
 * All consolida e retorna todos os dados de entrada da requisição:
 * corpo JSON, formulário e query string, priorizando chaves vindas do JSON em caso de sobreposição.
 *
 * @return map[string]interface{}
 */
func (r *Request) All() map[string]interface{} {
	out := map[string]interface{}{}

	if values, err := r.FormValues(); err == nil {
		for key := range values {
			out[key] = values.Get(key)
		}
	}
	for key, values := range r.Queries() {
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	if m := r.jsonBody(); m != nil {
		for key, value := range m {
			out[key] = value
		}
	}

	return out
}

/**
 * Only filtra e retorna apenas as chaves especificadas obtidas através de All().
 *
 * Exemplo:
 *  dados := req.Only("nome", "email")
 *
 * @param keys ...string
 * @return map[string]interface{}
 */
func (r *Request) Only(keys ...string) map[string]interface{} {
	all := r.All()
	out := map[string]interface{}{}
	for _, k := range keys {
		if v, ok := all[k]; ok {
			out[k] = v
		}
	}
	return out
}

/**
 * Except retorna todos os parâmetros obtidos de All(), excluindo as chaves informadas.
 * Útil para remoção de dados sensíveis antes de operações de log ou processamentos adicionais.
 *
 * @param keys ...string
 * @return map[string]interface{}
 */
func (r *Request) Except(keys ...string) map[string]interface{} {
	excluded := make(map[string]bool, len(keys))
	for _, k := range keys {
		excluded[k] = true
	}

	all := r.All()
	out := map[string]interface{}{}
	for k, v := range all {
		if !excluded[k] {
			out[k] = v
		}
	}
	return out
}

/**
 * InputFloat64 recupera o valor de um parâmetro via Input() e o converte para float64.
 * Retorna o valor padrão (def) se o parâmetro estiver ausente ou for inválido.
 *
 * Exemplo:
 *  preco := req.InputFloat64("preco", 0.0)
 *
 * @param name string
 * @param def float64
 * @return float64
 */
func (r *Request) InputFloat64(name string, def float64) float64 {
	raw := r.Input(name)
	if raw == "" {
		return def
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def
	}
	return val
}

/**
 * InputSlice recupera parâmetros com múltiplos valores presentes em requisições de formulário,
 * query strings no formato de lista (?tags=a&tags=b) ou arrays provenientes do corpo JSON.
 *
 * Exemplo:
 *  tags := req.InputSlice("tags") // ["go", "web"]
 *
 * @param name string
 * @return []string
 */
func (r *Request) InputSlice(name string) []string {
	if m := r.jsonBody(); m != nil {
		if arr, ok := m[name]; ok {
			if items, ok := arr.([]interface{}); ok {
				result := make([]string, 0, len(items))
				for _, item := range items {
					result = append(result, fmt.Sprintf("%v", item))
				}
				return result
			}
		}
	}

	if values, err := r.FormValues(); err == nil {
		if vals, ok := values[name]; ok && len(vals) > 0 {
			return vals
		}
	}

	if vals, ok := r.Queries()[name]; ok && len(vals) > 0 {
		return vals
	}

	return nil
}

/**
 * Filled indica se um parâmetro está presente na requisição E possui um valor não vazio.
 *
 * Exemplo:
 *  if req.Filled("nome") {
 *      // parâmetro presente e preenchido
 *  }
 *
 * @param name string
 * @return bool
 */
func (r *Request) Filled(name string) bool {
	return r.Input(name) != ""
}

/**
 * Missing indica se um determinado parâmetro NÃO foi enviado na requisição.
 *
 * @param name string
 * @return bool
 */
func (r *Request) Missing(name string) bool {
	return !r.Has(name)
}

/**
 * Merge combina um mapa de dados adicionais com os dados de entrada já existentes na requisição.
 * Valores definidos no parâmetro extra sobrescrevem chaves duplicadas.
 *
 * Exemplo:
 *  dados := req.Merge(map[string]interface{}{
 *      "status": "ativo",
 *      "role":   "user",
 *  })
 *
 * @param extra map[string]interface{}
 * @return map[string]interface{}
 */
func (r *Request) Merge(extra map[string]interface{}) map[string]interface{} {
	all := r.All()
	for k, v := range extra {
		all[k] = v
	}
	return all
}

/**
 * InputJSON faz a decodificação direta do corpo JSON da requisição para um map.
 * Retorna um mapa vazio se o corpo for ausente ou nulo.
 *
 * Exemplo:
 *  dados, err := req.InputJSON()
 *  if err != nil {
 *      return res.Status(400).WithJson(kite.ErrorResponse{Error: "JSON inválido", Status: 400})
 *  }
 *  nome := dados["nome"]
 *
 * @return map[string]interface{}
 * @return error
 */
func (r *Request) InputJSON() (map[string]interface{}, error) {
	data, err := r.Body()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}