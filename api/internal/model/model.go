package model

import (
	"time"
)

// Query defines the parameters that metrics can be queried for
type Query struct {
	StartAt    time.Time
	EndAt      time.Time
	MetricType MetricType
	Frequency  Frequency
}

// MetricType represents the different metric categories
type MetricType int32

const (
	// MetricTypeNone defaults to no specified type; if used in filtering, it returns all the available metrics
	MetricTypeNone MetricType = 0
	// MetricTypeCPULoad is the type for cpu load metrics
	MetricTypeCPULoad MetricType = 1
	// MetricTypeConcurrency is the type for concurrency metrics
	MetricTypeConcurrency MetricType = 2
)

// Frequency determines in what ranged should the metrics be aggregated
type Frequency int32

const (
	// FrequencyNone defaults to no aggregation; the data will be returned with the frequency that it is stored in
	FrequencyNone Frequency = 0
	// FrequencyBySeconds defaults the avarage metric value for seconds
	FrequencyBySeconds Frequency = 1
	// FrequencyByMinutes defaults the avarage metric value for minutes
	FrequencyByMinutes Frequency = 2
	// FrequencyByHours defaults the avarage metric value for hours
	FrequencyByHours Frequency = 3
	// FrequencyByDays defaults the avarage metric value for days
	FrequencyByDays Frequency = 4
	// FrequencyByMonths defaults the avarage metric value for months
	FrequencyByMonths Frequency = 5
	// FrequencyByYears defaults the avarage metric value for years
	FrequencyByYears Frequency = 6
)

// Metric is the data structure with all the metric types saved in the store
type Metric struct {
	Timestamp   time.Time `bson:"timestamp" json:"timestamp"`
	CPULoad     float64   `bson:"cpu_load,omitempty" json:"cpu_load,omitempty"`
	Concurrency int32     `bson:"concurrency,omitempty,truncate" json:"concurrency,omitempty"`
}

// MetricAverage is an average for a given metric
type MetricAverage struct {
	StartTime   time.Time `bson:"start" json:"start"`
	EndTime     time.Time `bson:"end" json:"end"`
	CPULoad     float64   `bson:"cpu_load,omitempty" json:"cpu_load,omitempty"`
	Concurrency float64   `bson:"concurrency,omitempty" json:"concurrency,omitempty"`
}
