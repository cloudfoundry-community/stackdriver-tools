package stackdriver

import (
	"fmt"
	"path"
	"time"

	"sync"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	EventTime time.Time
	Unit      string // TODO Should this be "1" if it's empty?
}

type MetricAdapter interface {
	PostMetrics([]Metric) error
}

type metricAdapter struct {
	projectID             string
	client                MetricClient
	descriptors           map[string]struct{}
	createDescriptorMutex *sync.Mutex
}

func NewMetricAdapter(projectID string, client MetricClient) (MetricAdapter, error) {
	ma := &metricAdapter{
		projectID:             projectID,
		client:                client,
		createDescriptorMutex: &sync.Mutex{},
		descriptors:           map[string]struct{}{},
	}

	err := ma.fetchMetricDescriptorNames()
	return ma, err
}

func (ma *metricAdapter) PostMetrics(metrics []Metric) error {
	projectName := path.Join("projects", ma.projectID)
	var timeSerieses []*monitoringpb.TimeSeries

	for _, metric := range metrics {
		err := ma.ensureMetricDescriptor(metric)
		if err != nil {
			return err
		}

		metricType := path.Join("custom.googleapis.com", metric.Name)
		timeSeries := monitoringpb.TimeSeries{
			Metric: &metricpb.Metric{
				Type:   metricType,
				Labels: metric.Labels,
			},
			Points: points(metric.Value, metric.EventTime),
		}
		timeSerieses = append(timeSerieses, &timeSeries)
	}

	request := &monitoringpb.CreateTimeSeriesRequest{
		Name:       projectName,
		TimeSeries: timeSerieses,
	}

	err := ma.client.Post(request)
	err = errors.Wrapf(err, "Request: %+v", request)
	return err
}

func (ma *metricAdapter) CreateMetricDescriptor(metric Metric) error {
	projectName := path.Join("projects", ma.projectID)
	metricType := path.Join("custom.googleapis.com", metric.Name)
	metricName := path.Join(projectName, "metricDescriptors", metricType)

	var labelDescriptors []*labelpb.LabelDescriptor
	for key := range metric.Labels {
		labelDescriptors = append(labelDescriptors, &labelpb.LabelDescriptor{
			Key:       key,
			ValueType: labelpb.LabelDescriptor_STRING,
		})
	}

	req := &monitoringpb.CreateMetricDescriptorRequest{
		Name: projectName,
		MetricDescriptor: &metricpb.MetricDescriptor{
			Name:        metricName,
			Type:        metricType,
			Labels:      labelDescriptors,
			MetricKind:  metricpb.MetricDescriptor_GAUGE,
			ValueType:   metricpb.MetricDescriptor_DOUBLE,
			Unit:        metric.Unit,
			Description: "stackdriver-nozzle created custom metric.",
			DisplayName: metric.Name, // TODO
		},
	}

	return ma.client.CreateMetricDescriptor(req)
}

func (ma *metricAdapter) fetchMetricDescriptorNames() error {
	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   fmt.Sprintf("projects/%s", ma.projectID),
		Filter: "metric.type = starts_with(\"custom.googleapis.com/\")",
	}

	descriptors, err := ma.client.ListMetricDescriptors(req)
	if err != nil {
		return err
	}

	for _, descriptor := range descriptors {
		ma.descriptors[descriptor.Name] = struct{}{}
	}
	return nil
}

func (ma *metricAdapter) ensureMetricDescriptor(metric Metric) error {
	if metric.Unit == "" {
		return nil
	}

	if _, ok := ma.descriptors[metric.Name]; ok {
		return nil
	}

	ma.createDescriptorMutex.Lock()
	defer ma.createDescriptorMutex.Unlock()

	err := ma.CreateMetricDescriptor(metric)
	if err != nil {
		return err
	}
	ma.descriptors[metric.Name] = struct{}{}
	return nil
}

func points(value float64, eventTime time.Time) []*monitoringpb.Point {
	timeStamp := timestamp.Timestamp{
		Seconds: eventTime.Unix(),
		Nanos:   int32(eventTime.Nanosecond()),
	}
	point := &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			EndTime:   &timeStamp,
			StartTime: &timeStamp,
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DoubleValue{
				DoubleValue: value,
			},
		},
	}
	return []*monitoringpb.Point{point}
}
