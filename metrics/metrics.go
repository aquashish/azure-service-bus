package metrics

import (
	"azure-service-bus/config"
	"fmt"
	"os"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"

	"github.com/rcrowley/go-metrics"
)

var (
	Client DDClient
)

type DDClient interface {
	// Gauge measures the value of a metric at a particular time.
	Gauge(name string, value float64, tags []string, rate float64) error
	// Count tracks how many times something happened per second.
	Count(name string, value int64, tags []string, rate float64) error
	// Decr is just Count of -1
	Decr(name string, tags []string, rate float64) error
	// Incr is just Count of 1
	Incr(name string, tags []string, rate float64) error
	// Set counts the number of unique elements in a group.
	Set(name string, value string, tags []string, rate float64) error
	// Histogram tracks the statistical distribution of a set of values on each host.
	Histogram(name string, value float64, tags []string, rate float64) error
	// Distribution tracks the statistical distribution of a set of values across your infrastructure.
	Distribution(name string, value float64, tags []string, rate float64) error
	// Timing sends timing information, it is an alias for TimeInMilliseconds
	Timing(name string, value time.Duration, tags []string, rate float64) error

	Close() error
}

func DefaultClient() DDClient {
	if Client == nil {
		return &LocalClient{}
	}
	return Client
}

// GaugeWithTags measures the value of a metric at a particular time.
func GaugeWithTags(name string, value float64, providedTags map[string]string) {
	tags := append(ConvertMapToSliceTags(providedTags), tags()...)
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Gauge(name, value, tags, sampleRate())
}

// Gauge measures the value of a metric at a particular time.
func Gauge(name string, value float64) {
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Gauge(name, value, tags(), sampleRate())
}

// CountWithTags tracks how many times something happened per second.
func CountWithTags(name string, value int64, providedTags map[string]string) {
	tags := append(ConvertMapToSliceTags(providedTags), tags()...)
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Count(name, value, tags, sampleRate())
}

// Count tracks how many times something happened per second.
func Count(name string, value int64) {
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Count(name, value, tags(), sampleRate())
}

// DecrWithTags is just Count of -1
func DecrWithTags(name string, providedTags map[string]string) {
	tags := append(ConvertMapToSliceTags(providedTags), tags()...)
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Decr(name, tags, sampleRate())
}

// Decr is just Count of -1
func Decr(name string) {
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Decr(name, tags(), sampleRate())
}

// IncrWithTags is just Count of 1
func IncrWithTags(name string, providedTags map[string]string) {
	tags := append(ConvertMapToSliceTags(providedTags), tags()...)
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Incr(name, tags, sampleRate())
}

// Incr is just Count of 1
func Incr(name string) {
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Incr(name, tags(), sampleRate())
}

// SetWithTags counts the number of unique elements in a group.
func SetWithTags(name string, value string, providedTags map[string]string) {
	tags := append(ConvertMapToSliceTags(providedTags), tags()...)
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Set(name, value, tags, sampleRate())
}

// Set counts the number of unique elements in a group.
func Set(name string, value string) {
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Set(name, value, tags(), sampleRate())
}

// HistogramWithTags tracks the statistical distribution of a set of values on each host.
func HistogramWithTags(name string, value float64, providedTags map[string]string) {
	tags := append(ConvertMapToSliceTags(providedTags), tags()...)
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Histogram(name, value, tags, sampleRate())
}

// Histogram tracks the statistical distribution of a set of values on each host.
func Histogram(name string, value float64) {
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Histogram(name, value, tags(), sampleRate())
}

// DistributionWithTags tracks the statistical distribution of a set of values across your infrastructure.
func DistributionWithTags(name string, value float64, providedTags map[string]string) {
	tags := append(ConvertMapToSliceTags(providedTags), tags()...)
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Distribution(name, value, tags, sampleRate())
}

// Distribution tracks the statistical distribution of a set of values across your infrastructure.
func Distribution(name string, value float64) {
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Distribution(name, value, tags(), sampleRate())
}

// TimingWithTags sends timing information, it is an alias for TimeInMilliseconds
func TimingWithTags(name string, value time.Duration, providedTags map[string]string) {
	tags := append(ConvertMapToSliceTags(providedTags), tags()...)
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Timing(name, value, tags, sampleRate())
}

// Timing sends timing information, it is an alias for TimeInMilliseconds
func Timing(name string, value time.Duration) {
	//  Metrics are best effort so we don't handle this error
	_ = DefaultClient().Timing(name, value, tags(), sampleRate())
}

func tags() []string {
	return []string{
		fmt.Sprintf("service_name:%s", os.Getenv("SERVICE_NAME")),
		fmt.Sprintf("pod_name:%s", os.Getenv("POD_NAME")),
	}
}

func sampleRate() float64 {
	return 1
}

func ConvertMapToSliceTags(providedTags map[string]string) []string {
	var tags []string

	for key, val := range providedTags {
		tags = append(tags, fmt.Sprintf("%s:%s", key, val))
	}

	return tags
}

func Start() error {
	if config.IsLocalEnvironment() {
		Client = &LocalClient{}
		return nil
	}

	s, err := statsd.New("127.0.0.1:8125", statsd.WithNamespace("application"))
	if err != nil {
		return err
	}
	Client = s
	reportClient := &Reporter{
		Registry: metrics.DefaultRegistry,
		Client:   s,
		interval: 10 * time.Second,
	}

	// start the client
	go reportClient.Flush()

	return nil
}
