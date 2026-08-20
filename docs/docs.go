package docs

import "embed"

//go:embed openapi.yaml
var openAPIFile embed.FS

func OpenAPIYAML() []byte {
	b, err := openAPIFile.ReadFile("openapi.yaml")
	if err != nil {
		return nil
	}
	return b
}

const UIPath = "/api/docs"
const SpecPath = "/api/docs/openapi.yaml"

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Authx API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css"/>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({
        url: "/api/docs/openapi.yaml",
        dom_id: "#swagger-ui",
        presets: [SwaggerUIBundle.presets.apis],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>
`

func SwaggerHTML() []byte {
	return []byte(swaggerHTML)
}
