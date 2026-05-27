// Package jobs contiene los handlers de los jobs internos invocados por Cloud Scheduler.
package jobs

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// TokenCleaner define la operación de limpieza de tokens que el handler necesita.
type TokenCleaner interface {
	LimpiarTokensExpirados() (int64, error)
}

// LimpiarTokensHandler maneja POST /internal/jobs/limpiar_tokens_revocados.
//
// Solo acepta requests con header X-CloudScheduler: true (T024).
// Registra en log estructurado JSON: tipo, iniciado_en, completado_en, resultado, eliminados.
func LimpiarTokensHandler(cleaner TokenCleaner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// T024 — Validación header X-CloudScheduler.
		if r.Header.Get("X-CloudScheduler") != "true" {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":   "no_autorizado",
				"mensaje": "Acceso restringido a jobs internos",
			})
			return
		}

		iniciadoEn := time.Now().UTC()
		eliminados, err := cleaner.LimpiarTokensExpirados()
		completadoEn := time.Now().UTC()

		resultado := "ok"
		if err != nil {
			resultado = "error"
		}

		// Log estructurado JSON (RF-AUTH-06, constitución §jobs).
		log.Printf(`{"job":"limpiar_tokens_revocados","iniciado_en":"%s","completado_en":"%s","resultado":"%s","eliminados":%d}`,
			iniciadoEn.Format(time.RFC3339),
			completadoEn.Format(time.RFC3339),
			resultado,
			eliminados,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"eliminados":    int64(0),
				"iniciado_en":   iniciadoEn.Format(time.RFC3339),
				"completado_en": completadoEn.Format(time.RFC3339),
				"resultado":     "error",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"eliminados":    eliminados,
			"iniciado_en":   iniciadoEn.Format(time.RFC3339),
			"completado_en": completadoEn.Format(time.RFC3339),
			"resultado":     "ok",
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
