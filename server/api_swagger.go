package server

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "ttyweb/docs/swagger" // swagger generated docs
)

// @title ttyweb API
// @description Web-based terminal emulator API
// @version 1.0
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.basic BasicAuth

var swaggerHandler http.Handler

func init() {
	swaggerHandler = httpSwagger.Handler(httpSwagger.URL("/api/docs/swagger.json"))
}

// handleSwaggerDocs serves the Swagger UI for API documentation.
func handleSwaggerDocs(w http.ResponseWriter, r *http.Request) {
	swaggerHandler.ServeHTTP(w, r)
}
