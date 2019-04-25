/*
Copyright 2019 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package monitoring

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/gravitational/trace"
)

// Metrics defines an interface for cluster metrics.
type Metrics interface {
	// GetTotalCPU returns total number of CPU cores in the cluster.
	GetTotalCPU(context.Context) (int, error)
	// GetTotalMemory returns total amount of RAM in the cluster in bytes.
	GetTotalMemory(context.Context) (int64, error)
	// GetCPURate returns CPU usage rate for the specified interval.
	GetCPURate(ctx context.Context, start, end time.Time, step time.Duration) (Series, error)
	// GetMemoryRate returns RAM usage rate for the specified interval.
	GetMemoryRate(ctx context.Context, start, end time.Time, step time.Duration) (Series, error)
	// GetCurrentCPURate returns instantaneous CPU usage rate.
	GetCurrentCPURate(context.Context) (int, error)
	// GetCurrentMemoryRate returns instantaneous RAM usage rate.
	GetCurrentMemoryRate(context.Context) (int, error)
	// GetMaxCPURate returns highest CPU usage rate on the specified interval.
	GetMaxCPURate(ctx context.Context, start, end time.Time) (int, error)
	// GetMaxMemoryRate returns highest RAM usage rate on the specified interval.
	GetMaxMemoryRate(ctx context.Context, start, end time.Time) (int, error)
}

// Series represents a time series, collection of data points.
type Series []Point

// Point represents a single data point in a time series.
type Point struct {
	// Time is the metric timestamp.
	Time time.Time
	// Value is the metric value.
	Value int
}

// prometheus retrieves cluster metrics by querying in-cluster Prometheus.
//
// Implements Metrics interface.
type prometheus struct {
	v1.API
}

// NewPrometheus returns a new Prometheus-backed metrics collector.
func NewPrometheus(address string) (*prometheus, error) {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return &prometheus{
		API: v1.NewAPI(client),
	}, nil
}

// GetTotalCPU returns total number of CPU cores in the cluster.
func (p *prometheus) GetTotalCPU(ctx context.Context) (int, error) {
	value, err := p.Query(ctx, queryTotalCPU, time.Time{})
	if err != nil {
		return 0, trace.Wrap(err)
	}
	if value.Type() != model.ValVector {
		return 0, trace.BadParameter("expected vector: %v %v", value.Type(), value.String())
	}
	vector := value.(model.Vector)
	if len(vector) != 1 {
		return 0, trace.BadParameter("expected single-element vector: %v", value.String())
	}
	return int(vector[0].Value), nil
}

// GetTotalMemory returns total amount of RAM in the cluster in bytes.
func (p *prometheus) GetTotalMemory(ctx context.Context) (int64, error) {
	return 0, nil
}

// GetCPURate returns CPU usage rate for the specified interval.
func (p *prometheus) GetCPURate(ctx context.Context, start, end time.Time, step time.Duration) (Series, error) {
	value, err := p.QueryRange(ctx, queryCPURate, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if value.Type() != model.ValMatrix {
		return nil, trace.BadParameter("expected matrix: %v %v", value.Type(), value.String())
	}
	matrix := value.(model.Matrix)
	if len(matrix) != 1 {
		return nil, trace.BadParameter("expected single-element matrix: %v", value.String())
	}
	var result Series
	for _, v := range matrix[0].Values {
		result = append(result, Point{
			Value: int(v.Value),
			Time:  v.Timestamp.Time(),
		})
	}

	return result, nil
}

// GetMemoryRate returns RAM usage rate for the specified interval.
func (p *prometheus) GetMemoryRate(ctx context.Context, start, end time.Time, step time.Duration) (Series, error) {
	value, err := p.QueryRange(ctx, queryMemoryRate, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if value.Type() != model.ValMatrix {
		return nil, trace.BadParameter("expected matrix: %v %v", value.Type(), value.String())
	}
	matrix := value.(model.Matrix)
	if len(matrix) != 1 {
		return nil, trace.BadParameter("expected single-element matrix: %v", value.String())
	}
	var result Series
	for _, v := range matrix[0].Values {
		result = append(result, Point{
			Value: int(v.Value),
			Time:  v.Timestamp.Time(),
		})
	}
	return result, nil
}

// GetCurrentCPURate returns instantaneous CPU usage rate.
func (p *prometheus) GetCurrentCPURate(ctx context.Context) (int, error) {
	value, err := p.Query(ctx, queryCPURate, time.Time{})
	if err != nil {
		return 0, trace.Wrap(err)
	}
	if value.Type() != model.ValVector {
		return 0, trace.BadParameter("expected vector: %v", value.String())
	}
	vector := value.(model.Vector)
	if len(vector) != 1 {
		return 0, trace.BadParameter("expected single-element vector: %v", value.String())
	}
	return int(vector[0].Value), nil
}

// GetCurrentMemoryRate returns instantaneous RAM usage rate.
func (p *prometheus) GetCurrentMemoryRate(ctx context.Context) (int, error) {
	value, err := p.Query(ctx, queryMemoryRate, time.Time{})
	if err != nil {
		return 0, trace.Wrap(err)
	}
	if value.Type() != model.ValVector {
		return 0, trace.BadParameter("expected vector: %v", value.String())
	}
	vector := value.(model.Vector)
	if len(vector) != 1 {
		return 0, trace.BadParameter("expected single-element vector: %v", value.String())
	}
	return int(vector[0].Value), nil
}

// GetMaxCPURate returns highest CPU usage rate on the specified interval.
func (p *prometheus) GetMaxCPURate(ctx context.Context, start, end time.Time) (int, error) {
	return 0, nil
}

// GetMaxMemoryRate returns highest RAM usage rate on the specified interval.
func (p *prometheus) GetMaxMemoryRate(ctx context.Context, start, end time.Time) (int, error) {
	return 0, nil
}

var (
	queryTotalCPU    = "cluster:cpu_total"
	queryTotalMemory = "cluster:memory_total_bytes"
	queryCPURate     = "cluster:cpu_usage_rate"
	queryMemoryRate  = "cluster:memory_usage_rate"
)
