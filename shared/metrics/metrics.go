package metrics

import (
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Counter struct {
	name   string
	value  int64
	mu     sync.RWMutex
	logger *zap.Logger
}

func NewCounter(name string, logger *zap.Logger) *Counter {
	return &Counter{
		name:   name,
		logger: logger,
	}
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Add(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += delta
}

func (c *Counter) Value() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

func (c *Counter) Log() {
	c.logger.Info("counter metric",
		zap.String("name", c.name),
		zap.Int64("value", c.Value()))
}

type Histogram struct {
	name    string
	count   int64
	sum     float64
	min     float64
	max     float64
	mu      sync.RWMutex
	logger  *zap.Logger
}

func NewHistogram(name string, logger *zap.Logger) *Histogram {
	return &Histogram{
		name:   name,
		min:    math.MaxFloat64,
		logger: logger,
	}
}

func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.sum += value
	if value < h.min {
		h.min = value
	}
	if value > h.max {
		h.max = value
	}
}

func (h *Histogram) Stats() HistogramStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	avg := 0.0
	if h.count > 0 {
		avg = h.sum / float64(h.count)
	}
	
	min := h.min
	if h.count == 0 {
		min = 0
	}
	
	return HistogramStats{
		Count: h.count,
		Sum:   h.sum,
		Avg:   avg,
		Min:   min,
		Max:   h.max,
	}
}

func (h *Histogram) Log() {
	stats := h.Stats()
	h.logger.Info("histogram metric",
		zap.String("name", h.name),
		zap.Int64("count", stats.Count),
		zap.Float64("sum", stats.Sum),
		zap.Float64("avg", stats.Avg),
		zap.Float64("min", stats.Min),
		zap.Float64("max", stats.Max))
}

type HistogramStats struct {
	Count int64   `json:"count"`
	Sum   float64 `json:"sum"`
	Avg   float64 `json:"avg"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

type Timer struct {
	histogram *Histogram
}

func NewTimer(name string, logger *zap.Logger) *Timer {
	return &Timer{
		histogram: NewHistogram(name+"_duration_ms", logger),
	}
}

func (t *Timer) Time(fn func()) {
	start := time.Now()
	fn()
	duration := time.Since(start)
	t.histogram.Observe(float64(duration.Milliseconds()))
}

func (t *Timer) TimeFunc(fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	t.histogram.Observe(float64(duration.Milliseconds()))
	return err
}

func (t *Timer) Log() {
	t.histogram.Log()
}

type Registry struct {
	counters   map[string]*Counter
	histograms map[string]*Histogram
	timers     map[string]*Timer
	mu         sync.RWMutex
	logger     *zap.Logger
}

func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		counters:   make(map[string]*Counter),
		histograms: make(map[string]*Histogram),
		timers:     make(map[string]*Timer),
		logger:     logger,
	}
}

func (r *Registry) Counter(name string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if counter, exists := r.counters[name]; exists {
		return counter
	}
	
	counter := NewCounter(name, r.logger)
	r.counters[name] = counter
	return counter
}

func (r *Registry) Histogram(name string) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if histogram, exists := r.histograms[name]; exists {
		return histogram
	}
	
	histogram := NewHistogram(name, r.logger)
	r.histograms[name] = histogram
	return histogram
}

func (r *Registry) Timer(name string) *Timer {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if timer, exists := r.timers[name]; exists {
		return timer
	}
	
	timer := NewTimer(name, r.logger)
	r.timers[name] = timer
	return timer
}

func (r *Registry) LogAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, counter := range r.counters {
		counter.Log()
	}
	
	for _, histogram := range r.histograms {
		histogram.Log()
	}
}

// ServiceMetrics provides common metrics for gRPC services
type ServiceMetrics struct {
	registry         *Registry
	requestCounter   *Counter
	errorCounter     *Counter
	requestTimer     *Timer
	circuitBreakerOp *Counter
}

func NewServiceMetrics(serviceName string, registry *Registry) *ServiceMetrics {
	return &ServiceMetrics{
		registry:         registry,
		requestCounter:   registry.Counter(serviceName + "_requests_total"),
		errorCounter:     registry.Counter(serviceName + "_errors_total"),
		requestTimer:     registry.Timer(serviceName + "_request"),
		circuitBreakerOp: registry.Counter(serviceName + "_circuit_breaker_operations"),
	}
}

func (sm *ServiceMetrics) RecordRequest(method string, fn func() error) error {
	sm.requestCounter.Inc()
	
	err := sm.requestTimer.TimeFunc(fn)
	if err != nil {
		sm.errorCounter.Inc()
	}
	
	return err
}

func (sm *ServiceMetrics) RecordCircuitBreakerOperation(operation string) {
	sm.circuitBreakerOp.Inc()
}

func (sm *ServiceMetrics) Log() {
	sm.registry.LogAll()
}