# Metrics

The metrics library exposes a set of functions which allow custom metrics to be emitted for services.

These metrics are sent via the datadog agent to Datadog where they can be used for custom dashboards + alertings.

The metrics client is automatically initialised on service startup, so you won't need to do anything else to use the library.

Metrics are best effort, but won't error if they fail to be emitted. They shouldn't be used where we need precise numbers (say for API billing etc).

## Metric Tags

- Each metric has two sets of functions i.e. `Gauge` and `GaugeWithTags`
- The second of these allows you to specify a set of custom tags which are sent alongside the metric
- This allows you to add granularity to a metric
- For example, if we wanted to track the total number of calls to a HTTP server by the method of the request, we could do:
  - a single metric for each method `metrics.Count("HTTP_REQUEST_METHOD_GET")`
  - one metric for all requests tagged with the method `metrics.CountWithTags("HTTP_REQUEST", map[string]string{"METHOD": "GET"})`
- The second of these is preferable as it makes it much easier to graph all requests as well as requests per method.

## Metric Types

The metrics types are wrappers around StatsD primitives. Read more [here](https://statsd.readthedocs.io/en/v3.3/types.html). The Datadog docs are also useful [here](https://docs.datadoghq.com/metrics/types/?tab=distribution#metric-types).

### Gauge

```go
// Gauge measures the value of a metric at a particular time.
Gauge(name string, value float64)
```

- The Gauge represents a metric which has a particular value at a moment in time.
- For example the speedometer on your car shows your speed at a particular moment in time

### Count

```go
// Count tracks how many times something happened per second.
Count(name string, value int64) 
```

- The Count metric keeps a counter of things that happen, i.e a number of server hits that have happened.

### Decr + Incr

```go
// Decr is just Count of -1
Decr(name string)
// Incr is just Count of 1
Incr(name string)
```

- Decr + Incr are just wrappers around a counter that decrease/increase a counter by 1.

### Timing

```go
// Timing sends timing information, it is an alias for TimeInMilliseconds
Timing(name string, value time.Duration)
```

- Timing measures how long something takes to complete. You'll automatically get access to percentiles based on the values.
- A normal implementation would look like:

```go
start := time.Now()
// do thing we want to measure
exec()

metrics.Timing("exec_duration", time.Since(start))
```

### Set

```go
// Set counts the number of unique elements in a group.
Set(name string, value string)
```

- Counts the number of unique elements in a group.
- For example `metrics.Set("users", user.UserID)` could be used to track the total number of unique users in a period. Calls with the same `user.UserID` value are ignored.

### Histogram

```go
// Histogram tracks the statistical distribution of a set of values on each host.
Histogram(name string, value float64) 
```

- The histogram metric submission type represents the statistical distribution of a set of values calculated Agent-side in one time interval.

### Distribution

```go
// Distribution tracks the statistical distribution of a set of values across your infrastructure.
Distribution(name string, value float64)
```

- The distribution metric submission type represents the global statistical distribution of a set of values calculated across your entire distributed infrastructure in one time interval.
