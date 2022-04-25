package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"sky/api/internal/model"

	"github.com/gorilla/mux"
)

// Handler is responsible for handling the API requests for returning the metrics
type Handler struct {
	store Store
}

// Store is an interface representing any timestories storage for metrics
type Store interface {
	GetSeries(ctx context.Context, filter model.Query) ([]model.Metric, error)
	GetAverage(ctx context.Context, filter model.Query) (*model.MetricAverage, error)
}

// NewMetricsHandler creates a handler with a storage
func NewMetricsHandler(store Store) *Handler {
	return &Handler{
		store: store,
	}
}

// GetTimeline should return a series of metrics for the given url
// type: can be cpu_load and concurrency;
// accepted query parameters:
// * start, end - being epoch time
// * frequency - possible values being "minutes", "hours", "days"
func (h *Handler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	filter := buildQueryFilter(w, r)
	if filter == nil {
		return
	}
	series, err := h.store.GetSeries(context.Background(), *filter)
	switch {
	case err != nil:
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	case len(series) == 0:
		writeError(w, fmt.Sprintf("data for specified filter does not exist; filter: %v", *filter), http.StatusNotFound)
		return
	default:
		jsonResp, err := json.Marshal(series)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)
	}
}

// GetAverage should return the stats for the given http params/filters
func (h *Handler) GetAverage(w http.ResponseWriter, r *http.Request) {
	filter := buildQueryFilter(w, r)
	if filter == nil {
		return
	}

	data, err := h.store.GetAverage(context.Background(), *filter)
	switch {
	case err != nil:
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	case data == nil:
		writeError(w, fmt.Sprintf("data for specified filter does not exist; filter: %v", *filter), http.StatusNotFound)
		return
	default:
		jsonResp, err := json.Marshal(data)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)
	}
}

func buildQueryFilter(w http.ResponseWriter, r *http.Request) *model.Query {
	// time range
	query := r.URL.Query()
	start := query.Get("start")
	end := query.Get("end")
	if start == "" || end == "" {
		writeError(w, "timerange wasn't specified", http.StatusBadRequest)
		return nil
	}

	var err error
	var startInt, endInt int
	startInt, err = strconv.Atoi(start)
	if err != nil {
		writeError(w, fmt.Sprintf("start timestamp is not valid; expected to be epoch format, but received %s", start), http.StatusBadRequest)
		return nil
	}
	endInt, err = strconv.Atoi(end)
	if err != nil {
		writeError(w, fmt.Sprintf("end timestamp is not valid; expected to be epoch format, but received %s", end), http.StatusBadRequest)
		return nil
	}

	// frequency
	var frequency model.Frequency
	switch query.Get("frequency") {
	case "seconds":
		frequency = model.FrequencyBySeconds
	case "minutes":
		frequency = model.FrequencyByMinutes
	case "hours":
		frequency = model.FrequencyByHours
	case "days":
		frequency = model.FrequencyByDays
	case "months":
		frequency = model.FrequencyByMonths
	case "years":
		frequency = model.FrequencyByYears
	case "":
		frequency = model.FrequencyNone
	default:
		writeError(w, fmt.Sprintf("frequency value is not valid; received %s", query.Get("frequency")), http.StatusBadRequest)
		return nil
	}

	// type
	vars := mux.Vars(r)
	var mType model.MetricType
	switch vars["type"] {
	case "cpu_load":
		mType = model.MetricTypeCPULoad
	case "concurrency":
		mType = model.MetricTypeConcurrency
	case "":
		mType = model.MetricTypeNone
	default:
		writeError(w, fmt.Sprintf("metric type is not valid; received %s", vars["type"]), http.StatusBadRequest)
		return nil
	}

	return &model.Query{
		StartAt:    time.Unix(int64(startInt), 0),
		EndAt:      time.Unix(int64(endInt), 0),
		MetricType: mType,
		Frequency:  frequency,
	}
}

func writeError(w http.ResponseWriter, message string, httpStatusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
}
