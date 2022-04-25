package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"sky/api/internal/model"

	"net/http"
	"net/http/httptest"
)

func TestHandler(t *testing.T) {
	now := time.Now()
	cases := []struct {
		description string
		dbSeries    []model.Metric

		expectedRespStatus int
	}{
		{
			"empty db", nil, http.StatusNotFound,
		},
		{
			"some data exists",
			[]model.Metric{
				{
					Timestamp:   now,
					CPULoad:     48,
					Concurrency: 365984,
				},
				{
					Timestamp:   now.Add(time.Duration(-1) * time.Minute),
					CPULoad:     48,
					Concurrency: 365984,
				},
				{
					Timestamp:   now.Add(time.Duration(-2) * time.Minute),
					CPULoad:     76,
					Concurrency: 965945,
				},
			},
			http.StatusOK,
		},
	}

	for _, c := range cases {
		store := mockStore{c.dbSeries}
		router := createRouter(store)

		nowT := now.Unix()
		minAgoT := now.Add(time.Duration(-1) * time.Minute).Unix()

		req, err := http.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/metrics/cpu_load?start=%d&end=%d&frequency=%s", minAgoT, nowT, "minutes"), strings.NewReader(""))
		assert.Nil(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, c.expectedRespStatus, rr.Code)

		res := rr.Result()
		bytes, err := io.ReadAll(res.Body)
		defer res.Body.Close()
		assert.Nil(t, err)

		// compare message - for error cases
		if rr.Code != http.StatusOK {
			var m map[string]string
			if err := json.Unmarshal([]byte(bytes), &m); err != nil {
				assert.Nil(t, err)
			}
			continue
		}

		var series []model.Metric
		if err := json.Unmarshal([]byte(bytes), &series); err != nil {
			assert.Nil(t, err)
		}
		assert.Equal(t, len(c.dbSeries), len(series))
	}
}

type mockStore struct {
	series []model.Metric
}

func (m mockStore) GetSeries(ctx context.Context, filter model.Query) ([]model.Metric, error) {
	return m.series, nil
}

func (m mockStore) GetAverage(ctx context.Context, filter model.Query) (*model.MetricAverage, error) {
	return nil, nil
}
