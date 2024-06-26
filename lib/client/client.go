package client

import (
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jainlokesh2/sonarcloud-exporter/internal"
	sonar "github.com/jainlokesh2/sonarcloud-exporter/lib/sonar"
)

// Stats struct is the list of expected to results to export.
type Stats struct {
	Projects     *[]ProjectStats
	Measurements *[]MeasurementsStats
	QualityGate  *[]QualityGateStats
}

// ProjectStats is the struct for SonarCloud projects data we want.
type ProjectStats struct {
	Organization string
	Key          string
	Name         string
	Qualifier    string
	LastAnalysis *time.Time
}

// MeasurementsStats is the struct for SonarCloud measurements we want.
type MeasurementsStats struct {
	Key       string
	Metric    string
	Value     string
	BestValue string
}

// QualityGateStats is the struct for SonarCloud quality gate status we want.
type QualityGateStats struct {
	Organization string
	Key          string
	Metric       string
	Value        string
	BestValue    string
}

// ExporterClient contains SonarCloud information for connecting
type ExporterClient struct {
	sqc *sonar.Client
}

// New returns a new Client connection to SonarCloud
func New(c internal.Config) *ExporterClient {
	return &ExporterClient{
		sqc: sonar.NewClient(c.Token, c.Organization),
	}
}

// GetStats retrieves data from API to create metrics from.
func (c *ExporterClient) GetStats() (*Stats, error) {

	projects, err := getProjects(c.sqc)
	if err != nil {
		return nil, err
	}

	measurements, err := getMeasurements(c.sqc, projects)
	if err != nil {
		return nil, err
	}

	qualityGate, err := QualityGateStatus(c.sqc, projects)
	if err != nil {
		return nil, err
	}

	return &Stats{
		Projects:     projects,
		Measurements: measurements,
		QualityGate:  qualityGate,
	}, nil
}

func getProjects(c *sonar.Client) (*[]ProjectStats, error) {
	var result []ProjectStats

	page := 1

	for {
		projects, err := c.ListProjects(&sonar.ListOptions{
			Page: page,
		})
		if err != nil {
			return nil, err
		}

		for _, project := range projects.Components {
			result = append(result, ProjectStats{
				Name:         project.Name,
				Qualifier:    project.Qualifier,
				Key:          project.Key,
				Organization: project.Organization,
			})
		}

		if len(projects.Components) == 0 {
			break
		}

		page++
	}

	log.Info("Found a total of: ", len(result), " projects")

	return &result, nil
}

func getMeasurements(c *sonar.Client, projects *[]ProjectStats) (*[]MeasurementsStats, error) {
	var result []MeasurementsStats

	for _, project := range *projects {
		data, err := c.ProjectMeasurements(project.Key)
		if err != nil {
			return nil, err
		}
		for _, measurement := range data.Component.Measures {
			result = append(result, MeasurementsStats{
				Key:       data.Component.Key,
				BestValue: strconv.FormatBool(measurement.BestValue),
				Metric:    measurement.Metric,
				Value:     measurement.Value,
			})
		}
	}

	return &result, nil
}

// QualityGateStatus retrieves the quality gate status for a project.
func QualityGateStatus(c *sonar.Client, projects *[]ProjectStats) (*[]QualityGateStats, error) {
	var result []QualityGateStats

	for _, project := range *projects {
		data, err := c.QualityGateMeasurement(project.Key)
		if err != nil {
			return nil, err
		}
		BestValue := 0
		if data.ProjectStatus.Status == "OK" {
			BestValue = 1
		}

		result = append(result, QualityGateStats{
			Key:          project.Key,
			BestValue:    strconv.Itoa(BestValue),
			Metric:       "sonar.quality.gate",
			Value:        strconv.Itoa(BestValue),
			Organization: project.Organization,
		})

	}

	return &result, nil
}
