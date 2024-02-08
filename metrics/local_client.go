package metrics

import "time"

type LocalClient struct {
}

// Gauge measures the value of a metric at a particular time.
func (c *LocalClient) Gauge(name string, value float64, tags []string, rate float64) error {
	return nil
}

// Count tracks how many times something happened per second.
func (c *LocalClient) Count(name string, value int64, tags []string, rate float64) error {
	return nil
}

// Decr is just Count of -1
func (c *LocalClient) Decr(name string, tags []string, rate float64) error {
	return nil
}

// Incr is just Count of 1
func (c *LocalClient) Incr(name string, tags []string, rate float64) error {
	return nil
}

// Set counts the number of unique elements in a group.
func (c *LocalClient) Set(name string, value string, tags []string, rate float64) error {
	return nil
}

// Histogram tracks the statistical distribution of a set of values on each host.
func (c *LocalClient) Histogram(name string, value float64, tags []string, rate float64) error {
	return nil
}

// Distribution tracks the statistical distribution of a set of values across your infrastructure.
func (c *LocalClient) Distribution(name string, value float64, tags []string, rate float64) error {
	return nil
}
func (c *LocalClient) Timing(name string, value time.Duration, tags []string, rate float64) error {
	return nil
}

func (c *LocalClient) Close() error {
	return nil
}
