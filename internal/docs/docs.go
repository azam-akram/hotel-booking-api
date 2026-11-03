package docs

import _ "embed"

//go:embed openapi.yml
var Spec []byte

// UIPage is Swagger UI loaded from a CDN, pointed at /openapi.yaml.
const UIPage = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Hotel Booking API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({ url: "/openapi.yml", dom_id: "#swagger-ui" });
  </script>
</body>
</html>`
