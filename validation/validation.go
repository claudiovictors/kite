package validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

/**
 * RuleFunc representa a assinatura de uma função de validação individual.
 *
 * @param field Nome do campo que está sendo validado.
 * @param value Valor bruto recebido do campo (pode ser nil caso ausente).
 * @param param Argumentos ou parâmetros passados para a regra (ex.: em "min:3", o valor é "3").
 * @param data Mapa completo contendo a estrutura de dados de entrada da requisição.
 *
 * @return Retorna a mensagem de erro formatada em caso de falha ou string vazia se o valor for válido.
 */
type RuleFunc func(field string, value interface{}, param string, data map[string]interface{}) string

/**
 * Mutex e registro global de regras suportadas pelo validador.
 */
var (
	mu       sync.RWMutex
	registry = map[string]RuleFunc{
		"required":  ruleRequired,
		"email":     ruleEmail,
		"min":       ruleMin,
		"max":       ruleMax,
		"numeric":   ruleNumeric,
		"integer":   ruleInteger,
		"boolean":   ruleBoolean,
		"string":    ruleString,
		"alpha":     ruleAlpha,
		"alpha_num": ruleAlphaNum,
		"in":        ruleIn,
		"same":      ruleSame,
		"confirmed": ruleConfirmed,
		"url":       ruleURL,
		"uuid":      ruleUUID,
		"regex":     ruleRegex,
		"array":     ruleArray,
		"nullable":  ruleNullable,
	}
)

/**
 * RegisterRule permite estender o validador adicionando regras customizadas globalmente.
 * Esta função utiliza controle de concorrência thread-safe para registro em tempo de execução.
 *
 * @param name Nome identificador da regra a ser registrada.
 * @param fn Função implementando a interface RuleFunc.
 */
func RegisterRule(name string, fn RuleFunc) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = fn
}

/**
 * Recupe uma regra do registro global sob bloqueio de leitura concorrente.
 */
func getRule(name string) (RuleFunc, bool) {
	mu.RLock()
	defer mu.RUnlock()
	fn, ok := registry[name]
	return fn, ok
}

/**
 * Validator gerencia o estado da validação, armazenando dados de entrada,
 * regras ativas, mensagens customizadas e erros acumulados.
 */
type Validator struct {
	data     map[string]interface{}
	rules    map[string]string
	messages map[string]map[string]string
	errors   map[string][]string
	checked  bool
}

/**
 * Make inicializa e retorna uma nova instância do Validator.
 *
 * @param data Mapa de dados brutos a serem analisados.
 * @param rules Mapeamento dos campos e suas respectivas cadeias de regras (ex.: "nome": "required|min:3").
 *
 * @return Ponteiro para o objeto Validator configurado.
 */
func Make(data map[string]interface{}, rules map[string]string) *Validator {
	if data == nil {
		data = map[string]interface{}{}
	}
	return &Validator{
		data:     data,
		rules:    rules,
		messages: map[string]map[string]string{},
	}
}

/**
 * WithMessage registra uma mensagem de erro customizada para um campo e regra específicos.
 *
 * @param field Nome do campo alvo.
 * @param rule Nome da regra para substituir a mensagem padrão.
 * @param message Texto customizado a ser exibido em caso de erro.
 *
 * @return Retorna a própria instância do Validator para chamada encadeada.
 */
func (v *Validator) WithMessage(field, rule, message string) *Validator {
	if v.messages[field] == nil {
		v.messages[field] = map[string]string{}
	}
	v.messages[field][rule] = message
	return v
}

/**
 * Validate executa o processamento de todas as regras associadas aos campos declarados.
 *
 * @return Retorna true se nenhuma regra falhou em nenhum campo, ou false caso contrário.
 */
func (v *Validator) Validate() bool {
	v.errors = map[string][]string{}

	for field, ruleStr := range v.rules {
		value, exists := v.data[field]
		rulesList := strings.Split(ruleStr, "|")

		isNullable := false
		for _, part := range rulesList {
			if strings.TrimSpace(part) == "nullable" {
				isNullable = true
				break
			}
		}

		if isNullable && (!exists || isBlank(value)) {
			continue
		}

		for _, part := range rulesList {
			part = strings.TrimSpace(part)
			if part == "" || part == "nullable" {
				continue
			}

			ruleName, param := parseRule(part)
			fn, ok := getRule(ruleName)
			if !ok {
				v.addError(field, fmt.Sprintf("regra de validação desconhecida: %q", ruleName))
				continue
			}

			if msg := fn(field, value, param, v.data); msg != "" {
				if custom, ok := v.messages[field][ruleName]; ok {
					msg = custom
				}
				v.addError(field, msg)
			}
		}
	}

	v.checked = true
	return len(v.errors) == 0
}

/**
 * Fails verifica se o conjunto de dados possui alguma inconsistência com as regras fornecidas.
 *
 * @return Retorna true se houver falhas de validação.
 */
func (v *Validator) Fails() bool {
	if !v.checked {
		v.Validate()
	}
	return len(v.errors) > 0
}

/**
 * Passes verifica se todos os dados fornecidos atendem plenamente às regras declaradas.
 *
 * @return Retorna true se não houver nenhum erro.
 */
func (v *Validator) Passes() bool {
	return !v.Fails()
}

/**
 * Errors retorna o mapa contendo todas as mensagens de erro geradas, indexadas por campo.
 *
 * @return Mapeamento dos campos para fatias contendo as mensagens de erro correspondentes.
 */
func (v *Validator) Errors() map[string][]string {
	if !v.checked {
		v.Validate()
	}
	return v.errors
}

/**
 * FirstError retorna a primeira mensagem de erro encontrada na estrutura de validação.
 *
 * @return String contendo a primeira mensagem de falha ou string vazia caso a validação passe.
 */
func (v *Validator) FirstError() string {
	if !v.checked {
		v.Validate()
	}
	keys := sortedKeys(v.errors)
	for _, field := range keys {
		if len(v.errors[field]) > 0 {
			return v.errors[field][0]
		}
	}
	return ""
}

/**
 * Adiciona uma mensagem de erro ao mapa interno de falhas.
 */
func (v *Validator) addError(field, msg string) {
	v.errors[field] = append(v.errors[field], msg)
}

/**
 * Decompõe a sintaxe da regra extraindo o nome identificador e seus parâmetros.
 */
func parseRule(part string) (name, param string) {
	idx := strings.Index(part, ":")
	if idx == -1 {
		return part, ""
	}
	return part[:idx], part[idx+1:]
}

/**
 * Ordena as chaves do mapa de erros alfabeticamente para garantir ordem determinística.
 */
func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ----------------------------------------------------------------------
// Regras Internas da Engine de Validação
// ----------------------------------------------------------------------

/**
 * Avalia se um valor arbitrário está vazio ou não inicializado.
 */
func isBlank(value interface{}) bool {
	if value == nil {
		return true
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	}
	return false
}

/**
 * Tenta converter um valor numérico genérico para float64 com segurança de tipo.
 */
func toFloat(value interface{}) (float64, bool) {
	if value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}

/**
 * Regra: Valida a presença obrigatória do campo.
 */
func ruleRequired(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return fmt.Sprintf("o campo %s é obrigatório", field)
	}
	return ""
}

/**
 * Regra: Permite que campos nulos ou nulos/vazios ignorem validações posteriores.
 */
func ruleNullable(_ string, _ interface{}, _ string, _ map[string]interface{}) string {
	return ""
}

/**
 * Regra: Valida o formato de endereço de e-mail.
 */
func ruleEmail(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	if _, err := mail.ParseAddress(fmt.Sprint(value)); err != nil {
		return fmt.Sprintf("o campo %s deve ser um e-mail válido", field)
	}
	return ""
}

/**
 * Regra: Valida o comprimento mínimo (strings) ou valor numérico mínimo.
 */
func ruleMin(field string, value interface{}, param string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	limit, err := strconv.Atoi(param)
	if err != nil {
		return ""
	}

	if s, ok := value.(string); ok {
		if len([]rune(s)) < limit {
			return fmt.Sprintf("o campo %s deve ter no mínimo %d caracteres", field, limit)
		}
		return ""
	}

	if num, ok := toFloat(value); ok {
		if num < float64(limit) {
			return fmt.Sprintf("o campo %s deve ser no mínimo %d", field, limit)
		}
	}
	return ""
}

/**
 * Regra: Valida o comprimento máximo (strings) ou valor numérico máximo.
 */
func ruleMax(field string, value interface{}, param string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	limit, err := strconv.Atoi(param)
	if err != nil {
		return ""
	}

	if s, ok := value.(string); ok {
		if len([]rune(s)) > limit {
			return fmt.Sprintf("o campo %s deve ter no máximo %d caracteres", field, limit)
		}
		return ""
	}

	if num, ok := toFloat(value); ok {
		if num > float64(limit) {
			return fmt.Sprintf("o campo %s deve ser no máximo %d", field, limit)
		}
	}
	return ""
}

/**
 * Regra: Valida se a entrada representa um valor numérico válido.
 */
func ruleNumeric(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	if _, ok := toFloat(value); !ok {
		return fmt.Sprintf("o campo %s deve ser numérico", field)
	}
	return ""
}

/**
 * Regra: Valida se a entrada é um número inteiro válido.
 */
func ruleInteger(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	num, ok := toFloat(value)
	if !ok || num != float64(int64(num)) {
		return fmt.Sprintf("o campo %s deve ser um número inteiro", field)
	}
	return ""
}

/**
 * Regra: Valida se a entrada é um tipo ou representação booleana.
 */
func ruleBoolean(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	switch v := value.(type) {
	case bool:
		return ""
	case string:
		switch strings.ToLower(v) {
		case "1", "0", "true", "false", "t", "f", "yes", "no":
			return ""
		}
	}
	return fmt.Sprintf("o campo %s deve ser verdadeiro ou falso", field)
}

/**
 * Regra: Valida se a entrada é do tipo string.
 */
func ruleString(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	if _, ok := value.(string); !ok {
		return fmt.Sprintf("o campo %s deve ser texto", field)
	}
	return ""
}

var (
	alphaRegex    = regexp.MustCompile(`^[a-zA-ZÀ-ÿ\s]+$`)
	alphaNumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	uuidRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

/**
 * Regra: Valida se a entrada contém apenas caracteres alfabéticos.
 */
func ruleAlpha(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	if !alphaRegex.MatchString(fmt.Sprint(value)) {
		return fmt.Sprintf("o campo %s deve conter apenas letras", field)
	}
	return ""
}

/**
 * Regra: Valida se a entrada contém apenas caracteres alfanuméricos.
 */
func ruleAlphaNum(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	if !alphaNumRegex.MatchString(fmt.Sprint(value)) {
		return fmt.Sprintf("o campo %s deve conter apenas letras e números", field)
	}
	return ""
}

/**
 * Regra: Valida se o valor do campo está contido em uma lista permitida.
 */
func ruleIn(field string, value interface{}, param string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	options := strings.Split(param, ",")
	current := fmt.Sprint(value)
	for _, opt := range options {
		if strings.TrimSpace(opt) == current {
			return ""
		}
	}
	return fmt.Sprintf("o campo %s deve ser um dos seguintes valores: %s", field, param)
}

/**
 * Regra: Valida se o valor do campo é idêntico ao valor de outro campo do payload.
 */
func ruleSame(field string, value interface{}, param string, data map[string]interface{}) string {
	other, exists := data[param]
	if !exists || fmt.Sprint(value) != fmt.Sprint(other) {
		return fmt.Sprintf("o campo %s deve ser igual a %s", field, param)
	}
	return ""
}

/**
 * Regra: Valida confirmação de correspondência com o sufixo _confirmation.
 */
func ruleConfirmed(field string, value interface{}, _ string, data map[string]interface{}) string {
	confirmField := field + "_confirmation"
	other, exists := data[confirmField]
	if !exists || fmt.Sprint(value) != fmt.Sprint(other) {
		return fmt.Sprintf("a confirmação do campo %s não confere", field)
	}
	return ""
}

/**
 * Regra: Valida se a string é uma estrutura de URL válida.
 */
func ruleURL(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	u, err := url.ParseRequestURI(fmt.Sprint(value))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Sprintf("o campo %s deve ser uma URL válida", field)
	}
	return ""
}

/**
 * Regra: Valida se a string atende ao padrão RFC de um UUID válido.
 */
func ruleUUID(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	if !uuidRegex.MatchString(fmt.Sprint(value)) {
		return fmt.Sprintf("o campo %s deve ser um UUID válido", field)
	}
	return ""
}

/**
 * Regra: Avalia a entrada com base em uma expressão regular informada no parâmetro.
 */
func ruleRegex(field string, value interface{}, param string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	re, err := regexp.Compile(param)
	if err != nil {
		return fmt.Sprintf("regex inválida na regra do campo %s", field)
	}
	if !re.MatchString(fmt.Sprint(value)) {
		return fmt.Sprintf("o campo %s tem um formato inválido", field)
	}
	return ""
}

/**
 * Regra: Valida se a estrutura informada é um Slice ou Array nativo em Go.
 */
func ruleArray(field string, value interface{}, _ string, _ map[string]interface{}) string {
	if isBlank(value) {
		return ""
	}
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		return ""
	}
	return fmt.Sprintf("o campo %s deve ser uma lista", field)
}