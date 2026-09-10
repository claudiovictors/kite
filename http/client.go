// Package http provides a fluent HTTP client for consuming external APIs,
// inspired by Laravel's Http:: facade. It allows building requests in a
// chainable manner and returning responses with helpers for JSON, text,
// status codes, etc.
//
// Quick example:
//
//	resp, err := http.Get("https://api.example.com/users")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Status())   // 200
//	fmt.Println(resp.Body())     // body as string
//
//	var users []User
//	resp.Json(&users)            // decode JSON
//
// Builder example:
//
//	resp, err := http.NewRequest().
//	    WithToken("my-jwt-token").
//	    WithHeader("X-Custom", "value").
//	    Timeout(10 * time.Second).
//	    Post("https://api.example.com/users", map[string]interface{}{
//	        "name":  "Ana",
//	        "email": "ana@example.com",
//	    })
package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ----------------------------------------------------------------------
// PendingRequest: fluent builder for HTTP requests
// ----------------------------------------------------------------------

/**
 * PendingRequest accumulates the configuration of an HTTP request before
 * sending it. Use NewRequest() to create one, or the global shortcuts Get(),
 * Post(), etc., for simple requests.
 */
type PendingRequest struct {
	client  *http.Client
	headers map[string]string
	query   url.Values
	timeout time.Duration
	baseURL string
	cookies []*http.Cookie

	// retry
	retries int
	retryMs time.Duration
}

/**
 * NewRequest creates a PendingRequest with sane defaults (30s default timeout).
 *
 * Example:
 *  req := http.NewRequest()
 *
 * @return *PendingRequest
 */
func NewRequest() *PendingRequest {
	return &PendingRequest{
		client:  &http.Client{Timeout: 30 * time.Second},
		headers: map[string]string{},
		query:   url.Values{},
		timeout: 30 * time.Second,
	}
}

/**
 * BaseURL sets a base URL that will be prefixed to every request,
 * useful when all endpoints share the same domain.
 *
 * Example:
 *  client := http.NewRequest().BaseURL("https://api.example.com")
 *  resp, _ := client.Get("/users")       // GET https://api.example.com/users
 *
 * @param base string
 * @return *PendingRequest
 */
func (pr *PendingRequest) BaseURL(base string) *PendingRequest {
	pr.baseURL = strings.TrimRight(base, "/")
	return pr
}

/**
 * Timeout sets the maximum time for the full request (connection + body read).
 * Default is 30 seconds.
 *
 * Example:
 *  req.Timeout(10 * time.Second)
 *
 * @param d time.Duration
 * @return *PendingRequest
 */
func (pr *PendingRequest) Timeout(d time.Duration) *PendingRequest {
	pr.timeout = d
	pr.client.Timeout = d
	return pr
}

/**
 * WithHeader adds a header to the request. It is chainable.
 *
 * Example:
 *  req.WithHeader("X-Custom-Header", "value")
 *
 * @param key string
 * @param value string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithHeader(key, value string) *PendingRequest {
	pr.headers[key] = value
	return pr
}

/**
 * WithHeaders adds multiple headers at once via a map.
 *
 * Example:
 *  req.WithHeaders(map[string]string{"Accept": "application/json", "X-App": "Mobile"})
 *
 * @param headers map[string]string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithHeaders(headers map[string]string) *PendingRequest {
	for k, v := range headers {
		pr.headers[k] = v
	}
	return pr
}

/**
 * WithToken adds an Authorization: Bearer <token> header.
 * Equivalent to Laravel's Http::withToken().
 *
 * Example:
 *  req.WithToken("eyJhbGciOi...")
 *
 * @param token string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithToken(token string) *PendingRequest {
	pr.headers["Authorization"] = "Bearer " + token
	return pr
}

/**
 * WithBasicAuth adds HTTP Basic authentication to the Authorization header.
 * Equivalent to Laravel's Http::withBasicAuth().
 *
 * Example:
 *  req.WithBasicAuth("user", "password123")
 *
 * @param user string
 * @param password string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithBasicAuth(user, password string) *PendingRequest {
	pr.headers["Authorization"] = "Basic " + basicAuth(user, password)
	return pr
}

/**
 * Accept sets the Accept header of the request.
 *
 * Example:
 *  req.Accept("text/xml")
 *
 * @param contentType string
 * @return *PendingRequest
 */
func (pr *PendingRequest) Accept(contentType string) *PendingRequest {
	pr.headers["Accept"] = contentType
	return pr
}

/**
 * AcceptJSON is a shortcut for Accept("application/json").
 *
 * Example:
 *  req.AcceptJSON()
 *
 * @return *PendingRequest
 */
func (pr *PendingRequest) AcceptJSON() *PendingRequest {
	return pr.Accept("application/json")
}

/**
 * ContentType sets the Content-Type header of the request.
 *
 * Example:
 *  req.ContentType("application/x-www-form-urlencoded")
 *
 * @param ct string
 * @return *PendingRequest
 */
func (pr *PendingRequest) ContentType(ct string) *PendingRequest {
	pr.headers["Content-Type"] = ct
	return pr
}

/**
 * WithQuery adds a single parameter to the URL query string.
 *
 * Example:
 *  req.WithQuery("page", "2")
 *
 * @param key string
 * @param value string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithQuery(key, value string) *PendingRequest {
	pr.query.Set(key, value)
	return pr
}

/**
 * WithQueryParams adds multiple query string parameters via a map.
 *
 * Example:
 *  req.WithQueryParams(map[string]string{"page": "1", "limit": "20"})
 *
 * @param params map[string]string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithQueryParams(params map[string]string) *PendingRequest {
	for k, v := range params {
		pr.query.Set(k, v)
	}
	return pr
}

/**
 * WithCookie adds a cookie to the request.
 *
 * Example:
 *  req.WithCookie("session_id", "abc123xyz")
 *
 * @param name string
 * @param value string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithCookie(name, value string) *PendingRequest {
	pr.cookies = append(pr.cookies, &http.Cookie{Name: name, Value: value})
	return pr
}

/**
 * WithCookies adds multiple cookies to the request via a map.
 *
 * Example:
 *  req.WithCookies(map[string]string{"theme": "dark", "lang": "en"})
 *
 * @param cookies map[string]string
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithCookies(cookies map[string]string) *PendingRequest {
	for name, value := range cookies {
		pr.cookies = append(pr.cookies, &http.Cookie{Name: name, Value: value})
	}
	return pr
}

/**
 * Retry configures the number of retry attempts and the delay between them
 * in case of network failures or 5xx errors. The request will be attempted
 * up to retries+1 times in total.
 *
 * Example:
 *  resp, err := http.NewRequest().
 *      Retry(3, 500*time.Millisecond).
 *      Get("https://api.unstable.com/data")
 *
 * @param retries int
 * @param delay time.Duration
 * @return *PendingRequest
 */
func (pr *PendingRequest) Retry(retries int, delay time.Duration) *PendingRequest {
	pr.retries = retries
	pr.retryMs = delay
	return pr
}

/**
 * WithoutRedirects disables automatic following of redirects (3xx).
 *
 * Example:
 *  req.WithoutRedirects()
 *
 * @return *PendingRequest
 */
func (pr *PendingRequest) WithoutRedirects() *PendingRequest {
	pr.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return pr
}

// ----------------------------------------------------------------------
// Send methods (HTTP verbs)
// ----------------------------------------------------------------------

/**
 * Get sends an HTTP GET request to the given URL.
 *
 * Example:
 *  resp, err := req.Get("/users")
 *
 * @param url string
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Get(url string) (*ClientResponse, error) {
	return pr.send(http.MethodGet, url, nil)
}

/**
 * Post sends an HTTP POST request with a JSON body.
 * The data parameter is automatically serialized to JSON. Pass nil if the request has no body.
 *
 * Example:
 *  resp, err := req.Post("/users", map[string]string{"name": "Ana"})
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Post(url string, data interface{}) (*ClientResponse, error) {
	return pr.sendJSON(http.MethodPost, url, data)
}

/**
 * Put sends an HTTP PUT request with a JSON body.
 *
 * Example:
 *  resp, err := req.Put("/users/1", userStruct)
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Put(url string, data interface{}) (*ClientResponse, error) {
	return pr.sendJSON(http.MethodPut, url, data)
}

/**
 * Patch sends an HTTP PATCH request with a JSON body.
 *
 * Example:
 *  resp, err := req.Patch("/users/1", map[string]interface{}{"status": "active"})
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Patch(url string, data interface{}) (*ClientResponse, error) {
	return pr.sendJSON(http.MethodPatch, url, data)
}

/**
 * Delete sends an HTTP DELETE request. The data parameter is optional (may be omitted or nil).
 *
 * Example:
 *  resp, err := req.Delete("/users/1")
 *
 * @param url string
 * @param data ...interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Delete(url string, data ...interface{}) (*ClientResponse, error) {
	var body interface{}
	if len(data) > 0 {
		body = data[0]
	}
	if body != nil {
		return pr.sendJSON(http.MethodDelete, url, body)
	}
	return pr.send(http.MethodDelete, url, nil)
}

/**
 * Head sends an HTTP HEAD request to the given URL.
 *
 * Example:
 *  resp, err := req.Head("/files/download.zip")
 *
 * @param url string
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Head(url string) (*ClientResponse, error) {
	return pr.send(http.MethodHead, url, nil)
}

/**
 * Options sends an HTTP OPTIONS request to the given URL.
 *
 * Example:
 *  resp, err := req.Options("/api/v1")
 *
 * @param url string
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) Options(url string) (*ClientResponse, error) {
	return pr.send(http.MethodOptions, url, nil)
}

/**
 * PostForm sends an HTTP POST request encoded as application/x-www-form-urlencoded.
 *
 * Example:
 *  resp, _ := http.NewRequest().PostForm("https://api.com/login", map[string]string{
 *      "email":    "ana@example.com",
 *      "password": "123456",
 *  })
 *
 * @param reqURL string
 * @param data map[string]string
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) PostForm(reqURL string, data map[string]string) (*ClientResponse, error) {
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}
	body := strings.NewReader(form.Encode())
	pr.headers["Content-Type"] = "application/x-www-form-urlencoded"
	return pr.send(http.MethodPost, reqURL, body)
}

/**
 * PostMultipart sends an HTTP POST multipart/form-data request, ideal for uploading
 * files combined with text fields.
 *
 * Example:
 *  resp, _ := http.NewRequest().PostMultipart("https://api.com/upload",
 *      map[string]string{"description": "profile photo"},
 *      map[string][]byte{"avatar": avatarBytes},
 *  )
 *
 * @param reqURL string
 * @param fields map[string]string
 * @param files map[string][]byte
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) PostMultipart(reqURL string, fields map[string]string, files map[string][]byte) (*ClientResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return nil, fmt.Errorf("http: error writing field %q: %w", key, err)
		}
	}

	for name, data := range files {
		part, err := writer.CreateFormFile(name, name)
		if err != nil {
			return nil, fmt.Errorf("http: error creating multipart part %q: %w", name, err)
		}
		if _, err := part.Write(data); err != nil {
			return nil, fmt.Errorf("http: error writing file %q: %w", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("http: error closing multipart writer: %w", err)
	}

	pr.headers["Content-Type"] = writer.FormDataContentType()
	return pr.send(http.MethodPost, reqURL, &buf)
}

// ----------------------------------------------------------------------
// Internal send helpers
// ----------------------------------------------------------------------

/**
 * sendJSON serializes the body to JSON and sets the Content-Type before calling send.
 *
 * @param method string
 * @param reqURL string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) sendJSON(method, reqURL string, data interface{}) (*ClientResponse, error) {
	var body io.Reader
	if data != nil {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("http: error serializing JSON: %w", err)
		}
		body = bytes.NewReader(jsonBytes)
		if _, ok := pr.headers["Content-Type"]; !ok {
			pr.headers["Content-Type"] = "application/json; charset=utf-8"
		}
	}
	return pr.send(method, reqURL, body)
}

/**
 * resolveURL builds the final URL by joining the baseURL (if set) and appending
 * any query string parameters.
 *
 * @param rawURL string
 * @return string
 */
func (pr *PendingRequest) resolveURL(rawURL string) string {
	if pr.baseURL != "" && !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = pr.baseURL + "/" + strings.TrimLeft(rawURL, "/")
	}
	if len(pr.query) > 0 {
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}
		rawURL += sep + pr.query.Encode()
	}
	return rawURL
}

/**
 * send executes the native HTTP request, handling retry attempts, headers,
 * cookies, and reading the response.
 *
 * @param method string
 * @param rawURL string
 * @param body io.Reader
 * @return (*ClientResponse, error)
 */
func (pr *PendingRequest) send(method, rawURL string, body io.Reader) (*ClientResponse, error) {
	fullURL := pr.resolveURL(rawURL)

	var lastErr error
	maxAttempts := 1 + pr.retries

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 && pr.retryMs > 0 {
			time.Sleep(pr.retryMs)
		}

		req, err := http.NewRequest(method, fullURL, body)
		if err != nil {
			return nil, fmt.Errorf("http: error creating request: %w", err)
		}

		for k, v := range pr.headers {
			req.Header.Set(k, v)
		}
		for _, c := range pr.cookies {
			req.AddCookie(c)
		}

		resp, err := pr.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		cr := &ClientResponse{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			body:       respBody,
			cookies:    resp.Cookies(),
		}

		// Only retry on server errors (5xx)
		if resp.StatusCode >= 500 && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("http: status %d", resp.StatusCode)
			continue
		}

		return cr, nil
	}

	return nil, fmt.Errorf("http: all %d attempts failed: %w", maxAttempts, lastErr)
}

// ----------------------------------------------------------------------
// ClientResponse: response received from an external API
// ----------------------------------------------------------------------

/**
 * ClientResponse wraps the response from an HTTP request made by the
 * client, with helpers to read the body as JSON, text, check status,
 * etc. — equivalent to Laravel's Http:: Response.
 */
type ClientResponse struct {
	statusCode int
	headers    http.Header
	body       []byte
	cookies    []*http.Cookie
}

/**
 * Status returns the HTTP status code of the response (e.g. 200, 404, 500).
 *
 * @return int
 */
func (r *ClientResponse) Status() int {
	return r.statusCode
}

/**
 * Ok reports whether the status is in the success range (200-299).
 *
 * @return bool
 */
func (r *ClientResponse) Ok() bool {
	return r.statusCode >= 200 && r.statusCode < 300
}

/**
 * Successful is an alias for Ok(). Reports whether the request succeeded (2xx).
 *
 * @return bool
 */
func (r *ClientResponse) Successful() bool {
	return r.Ok()
}

/**
 * Failed reports whether the response has an error status code (>= 400).
 *
 * @return bool
 */
func (r *ClientResponse) Failed() bool {
	return r.statusCode >= 400
}

/**
 * ServerError reports whether an internal error occurred on the remote server (5xx).
 *
 * @return bool
 */
func (r *ClientResponse) ServerError() bool {
	return r.statusCode >= 500
}

/**
 * ClientError reports whether the error was caused by a client-side request issue (4xx).
 *
 * @return bool
 */
func (r *ClientResponse) ClientError() bool {
	return r.statusCode >= 400 && r.statusCode < 500
}

/**
 * Redirect reports whether the response is a redirect (3xx).
 *
 * @return bool
 */
func (r *ClientResponse) Redirect() bool {
	return r.statusCode >= 300 && r.statusCode < 400
}

/**
 * Unauthorized reports whether the request was not authorized (status 401).
 *
 * @return bool
 */
func (r *ClientResponse) Unauthorized() bool {
	return r.statusCode == 401
}

/**
 * Forbidden reports whether access to the resource was forbidden (status 403).
 *
 * @return bool
 */
func (r *ClientResponse) Forbidden() bool {
	return r.statusCode == 403
}

/**
 * NotFound reports whether the requested resource was not found (status 404).
 *
 * @return bool
 */
func (r *ClientResponse) NotFound() bool {
	return r.statusCode == 404
}

/**
 * Body returns the raw response body as a string.
 *
 * @return string
 */
func (r *ClientResponse) Body() string {
	return string(r.body)
}

/**
 * Bytes returns the raw byte slice of the response body.
 *
 * @return []byte
 */
func (r *ClientResponse) Bytes() []byte {
	return r.body
}

/**
 * Json decodes the JSON response body into the provided destination (pointer).
 *
 * Example:
 *  var user User
 *  resp.Json(&user)
 *
 * @param dest interface{}
 * @return error
 */
func (r *ClientResponse) Json(dest interface{}) error {
	if len(r.body) == 0 {
		return fmt.Errorf("http: empty body, cannot decode JSON")
	}
	return json.Unmarshal(r.body, dest)
}

/**
 * Map decodes the JSON body into a map[string]interface{}, useful for responses
 * without a defined struct.
 *
 * Example:
 *  data, _ := resp.Map()
 *  fmt.Println(data["name"])
 *
 * @return (map[string]interface{}, error)
 */
func (r *ClientResponse) Map() (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := r.Json(&m); err != nil {
		return nil, err
	}
	return m, nil
}

/**
 * Collect decodes the JSON body into a slice of maps, useful when the API
 * returns a JSON array of objects.
 *
 * Example:
 *  items, _ := resp.Collect()
 *  for _, item := range items {
 *      fmt.Println(item["name"])
 *  }
 *
 * @return ([]map[string]interface{}, error)
 */
func (r *ClientResponse) Collect() ([]map[string]interface{}, error) {
	var items []map[string]interface{}
	if err := r.Json(&items); err != nil {
		return nil, err
	}
	return items, nil
}

/**
 * Header returns the first value associated with the given response header name.
 *
 * @param name string
 * @return string
 */
func (r *ClientResponse) Header(name string) string {
	return r.headers.Get(name)
}

/**
 * Headers returns all headers received in the HTTP response.
 *
 * @return http.Header
 */
func (r *ClientResponse) Headers() http.Header {
	return r.headers
}

/**
 * Cookies returns all cookies set by the server in the response.
 *
 * @return []*http.Cookie
 */
func (r *ClientResponse) Cookies() []*http.Cookie {
	return r.cookies
}

/**
 * Cookie returns the value of a specific cookie by name, or an empty string if not found.
 *
 * @param name string
 * @return string
 */
func (r *ClientResponse) Cookie(name string) string {
	for _, c := range r.cookies {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// ----------------------------------------------------------------------
// Global shortcuts (package-level functions)
// ----------------------------------------------------------------------

/**
 * Get is a global shortcut for NewRequest().Get(url).
 *
 * Example:
 *  resp, err := http.Get("https://api.example.com/users")
 *
 * @param url string
 * @return (*ClientResponse, error)
 */
func Get(url string) (*ClientResponse, error) {
	return NewRequest().Get(url)
}

/**
 * Post is a global shortcut for NewRequest().Post(url, data).
 *
 * Example:
 *  resp, err := http.Post("https://api.example.com/users", payload)
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func Post(url string, data interface{}) (*ClientResponse, error) {
	return NewRequest().Post(url, data)
}

/**
 * Put is a global shortcut for NewRequest().Put(url, data).
 *
 * Example:
 *  resp, err := http.Put("https://api.example.com/users/1", payload)
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func Put(url string, data interface{}) (*ClientResponse, error) {
	return NewRequest().Put(url, data)
}

/**
 * Patch is a global shortcut for NewRequest().Patch(url, data).
 *
 * Example:
 *  resp, err := http.Patch("https://api.example.com/users/1", payload)
 *
 * @param url string
 * @param data interface{}
 * @return (*ClientResponse, error)
 */
func Patch(url string, data interface{}) (*ClientResponse, error) {
	return NewRequest().Patch(url, data)
}

/**
 * Delete is a global shortcut for NewRequest().Delete(url, data...).
 *
 * Example:
 *  resp, err := http.Delete("https://api.example.com/users/1")
 *
 * @param url string
 * @param data ...interface{}
 * @return (*ClientResponse, error)
 */
func Delete(url string, data ...interface{}) (*ClientResponse, error) {
	return NewRequest().Delete(url, data...)
}

// ----------------------------------------------------------------------
// Internal helpers
// ----------------------------------------------------------------------

/**
 * basicAuth encodes user:password credentials in Base64 for use in the
 * HTTP Basic Auth header.
 *
 * @param user string
 * @param password string
 * @return string
 */
func basicAuth(user, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
}
