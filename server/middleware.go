package server

import (
	"net/http"

	"go.uber.org/zap"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO - Will add request-id in log
		zap.L().Info("RequestURI: ", zap.String("URI",r.RequestURI))
		next.ServeHTTP(w,r)
	})
}