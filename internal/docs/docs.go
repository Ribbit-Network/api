// Package docs serves the OpenAPI specification and a Scalar-rendered API
// reference page.
package docs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var openAPISpec []byte

const referenceHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Ribbit Network API — Reference</title>
    <link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Ctext y='.9em' font-size='90'%3E🐸%3C/text%3E%3C/svg%3E" />
  </head>
  <body>
    <script id="api-reference" data-url="/openapi.yaml"></script>
    <script>
      var configuration = {
        theme: "purple",
        layout: "modern",
        hideDownloadButton: false,
        metaData: {
          title: "Ribbit Network API",
        },
      };
      document.getElementById("api-reference").dataset.configuration =
        JSON.stringify(configuration);
    </script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>
`

// HandleSpec serves the embedded OpenAPI document.
func HandleSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(openAPISpec)
}

// HandleReference serves the Scalar-rendered API reference page.
func HandleReference(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(referenceHTML))
}
