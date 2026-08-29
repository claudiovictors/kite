package kite

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// Map é um atalho para map[string]interface{}, usado para passar dados a
// views (res.Render), respostas avulsas e qualquer lugar que peça um payload
// dinâmico — evita escrever "map[string]interface{}" toda hora.
//
//	return res.Render("index", kite.Map{"Title": "Hello, World!"})
type Map map[string]interface{}

// Env lê uma variável de ambiente, devolvendo fallback quando ela não
// estiver definida (ou vier vazia) — atalho para os.Getenv com valor padrão.
//
//	port := kite.Env("PORT", "8080")
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Must faz panic se err não for nil, e devolve o próprio valor caso
// contrário. Pensado para inicializações no boot da aplicação que você já
// espera nunca falhar em produção (carregar config, gerar chave, etc.).
//
//	data := kite.Must(os.ReadFile("config.json"))
func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

// Ptr devolve um ponteiro para o valor informado — útil pra preencher
// campos opcionais (*string, *int, ...) sem precisar de uma variável
// intermediária.
//
//	user.Nickname = kite.Ptr("Vic")
func Ptr[T any](v T) *T {
	return &v
}

// Coalesce devolve o primeiro valor não vazio da lista, ou "" se todos
// vierem vazios — equivalente ao coalesce()/?? de outras linguagens.
//
//	nome := kite.Coalesce(req.Input("apelido"), req.Input("nome"), "Visitante")
func Coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Contains indica se slice contém item (comparação exata). Genérico:
// funciona com qualquer tipo comparável ([]string, []int, ...).
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Truncate corta s em length caracteres (runas), acrescentando "..." no
// final quando o corte de fato ocorrer. Não mexe em s se já for menor ou
// igual a length.
func Truncate(s string, length int) string {
	runes := []rune(s)
	if len(runes) <= length {
		return s
	}
	if length <= 3 {
		return string(runes[:length])
	}
	return string(runes[:length-3]) + "..."
}

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)
var slugTrimDashes = regexp.MustCompile(`^-+|-+$`)

// Slugify converte uma string livre num slug de URL: minúsculas, espaços e
// caracteres especiais viram "-", acentos comuns do português são
// removidos, e traços duplicados/nas pontas são limpos.
//
//	kite.Slugify("Título do Post!") // -> "titulo-do-post"
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = stripAccents(s)
	s = slugInvalidChars.ReplaceAllString(s, "-")
	s = slugTrimDashes.ReplaceAllString(s, "")
	return s
}

// stripAccents troca os acentos mais comuns do português pelas letras sem
// acento, numa tabela simples (sem depender de unicode/norm).
func stripAccents(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "õ", "o", "ô", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
	)
	return replacer.Replace(s)
}

// RandomString gera uma string aleatória (base64 URL-safe, sem padding)
// com aproximadamente n caracteres — útil para tokens, chaves de API,
// nomes de arquivo temporários, etc. Usa crypto/rand (seguro para segredos,
// ao contrário de math/rand).
func RandomString(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// Extremamente improvável (falha na fonte de entropia do SO);
		// preferimos um valor previsível a propagar pânico em produção.
		return strings.Repeat("x", n)
	}
	encoded := base64.RawURLEncoding.EncodeToString(buf)
	if len(encoded) > n {
		return encoded[:n]
	}
	return encoded
}

// ToJSON serializa v para uma string JSON, devolvendo "{}" em caso de erro
// — pensado para logging rápido, onde você não quer lidar com o erro.
func ToJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}