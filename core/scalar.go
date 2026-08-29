package kite

import (
	"encoding/json"
	"fmt"
)

// ScalarTheme defines available color themes for Scalar API Reference.
type ScalarTheme string

const (
	ThemeDefault   ScalarTheme = "default"
	ThemeAlternate ScalarTheme = "alternate"
	ThemeMoon      ScalarTheme = "moon"
	ThemePurple    ScalarTheme = "purple"
	ThemeSolarized ScalarTheme = "solarized"
	ThemeSaturn    ScalarTheme = "saturn"
	ThemeKepler    ScalarTheme = "kepler"
	ThemeMars      ScalarTheme = "mars"
	ThemeDeepSpace ScalarTheme = "deepSpace"
	ThemeNone      ScalarTheme = "none"
)

/**
 * ScalarConfig holds visual and functional settings for Scalar API Reference.
 */
type ScalarConfig struct {
	Title        string            `json:"title,omitempty"`
	SpecURL      string            `json:"specURL,omitempty"`
	Theme        ScalarTheme       `json:"theme,omitempty"`
	Layout       string            `json:"layout,omitempty"` // "modern" or "classic"
	DarkMode     *bool             `json:"darkMode,omitempty"`
	ShowSidebar  *bool             `json:"showSidebar,omitempty"`
	ProxyURL     string            `json:"proxyUrl,omitempty"`
	CustomCSS    string            `json:"customCss,omitempty"`
	SearchHotKey string            `json:"searchHotKey,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

/**
 * RenderScalarHTML generates the standalone HTML page for Scalar API Reference.
 */
func RenderScalarHTML(cfg ScalarConfig) (string, error) {
	title := cfg.Title
	if title == "" {
		title = "API Reference"
	}

	theme := cfg.Theme
	if theme == "" {
		theme = ThemePurple
	}

	layout := cfg.Layout
	if layout == "" {
		layout = "modern"
	}

	scalarOptions := map[string]any{
		"theme":  theme,
		"layout": layout,
	}

	if cfg.ProxyURL != "" {
		scalarOptions["proxyUrl"] = cfg.ProxyURL
	}
	if cfg.DarkMode != nil {
		scalarOptions["darkMode"] = *cfg.DarkMode
	}
	if cfg.ShowSidebar != nil {
		scalarOptions["showSidebar"] = *cfg.ShowSidebar
	}
	if cfg.SearchHotKey != "" {
		scalarOptions["searchHotKey"] = cfg.SearchHotKey
	}
	if cfg.CustomCSS != "" {
		scalarOptions["customCss"] = cfg.CustomCSS
	}

	optionsJSON, err := json.Marshal(scalarOptions)
	if err != nil {
		return "", err
	}

	html := fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head>
    <title>%s</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="icon" type="image/svg+xml" href="https://scalar.com/favicon.svg" />
    <style>
      body {
        margin: 0;
        padding: 0;
        min-height: 100vh;
      }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="%s"
      data-configuration='%s'
      src="https://cdn.jsdelivr.net/npm/@scalar/api-reference">
    </script>
  </body>
</html>`, title, cfg.SpecURL, string(optionsJSON))

	return html, nil
}
