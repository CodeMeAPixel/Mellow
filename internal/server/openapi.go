package server

import "net/http"

const openAPISpec = `{
  "openapi": "3.1.0",
  "info": {
    "title": "Mellow API",
    "version": "1.0.0",
    "description": "REST API for Mellow, an AI mental health companion for Discord."
  },
  "servers": [{ "url": "/" }],
  "paths": {
    "/healthz": {
      "get": {
        "summary": "Health check",
        "tags": ["Base"],
        "responses": {
          "200": {
            "description": "Service is up",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": { "status": { "type": "string", "example": "ok" } }
                }
              }
            }
          }
        }
      }
    },
    "/v1/stats": {
      "get": {
        "summary": "Community statistics",
        "tags": ["Stats"],
        "responses": {
          "200": {
            "description": "Aggregate counts",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "Users": { "type": "integer" },
                    "Guilds": { "type": "integer" },
                    "Conversations": { "type": "integer" },
                    "CrisisEvents": { "type": "integer" },
                    "MoodCheckIns": { "type": "integer" }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/v1/testimonials": {
      "get": {
        "summary": "Public testimonials",
        "tags": ["Testimonials"],
        "responses": {
          "200": {
            "description": "Approved, public feedback",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "message": { "type": "string" },
                      "createdAt": { "type": "string", "format": "date-time" },
                      "featured": { "type": "boolean" }
                    }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/v1/feedback": {
      "post": {
        "summary": "Submit feedback",
        "tags": ["Testimonials"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["userId", "message"],
                "properties": {
                  "userId": { "type": "string" },
                  "message": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "202": { "description": "Feedback received" },
          "400": { "description": "Missing userId or message" },
          "429": { "description": "Rate limited" }
        }
      }
    },
    "/v1/chat": {
      "post": {
        "summary": "Chat with Mellow",
        "tags": ["Chat"],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["message"],
                "properties": { "message": { "type": "string" } }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "AI reply",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": { "reply": { "type": "string" } }
                }
              }
            }
          },
          "400": { "description": "Missing message" },
          "429": { "description": "Rate limited" },
          "503": { "description": "AI not configured" }
        }
      }
    }
  }
}`

const docsHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Mellow API</title>
</head>
<body>
  <script id="api-reference" data-url="/openapi.json"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISpec))
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(docsHTML))
}
