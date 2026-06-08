package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gocourier/pkg/apperrors"
)

func (s *Server) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	summary, err := s.dashboard.Summary(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleDashboardDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, apperrors.ErrValidation)
		return
	}
	detail, err := s.dashboard.Detail(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, apperrors.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
