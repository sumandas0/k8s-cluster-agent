package openapi

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func SwaggerHandler() http.Handler {
	return httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
		httpSwagger.InstanceName("swagger"),
		httpSwagger.PersistAuthorization(true),
	)
}
