package config

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BaymaxRice/prometheus-nginxlog-exporter/log"
)

// NamespaceConfig is a struct describing single metric namespaces
type NamespaceConfig struct {
	Name string `hcl:",key"`

	NamespaceLabelName string `hcl:"namespace_label" yaml:"namespace_label"`
	NamespaceLabels    map[string]string

	MetricsOverride *struct {
		Prefix string `hcl:"prefix" yaml:"prefix"`
	} `hcl:"metrics_override" yaml:"metrics_override"`
	NamespacePrefix string

	SourceFiles      []string          `hcl:"source_files" yaml:"source_files"`
	SourceData       SourceData        `hcl:"source" yaml:"source"`
	Parser           string            `hcl:"parser" yaml:"parser"`
	Format           string            `hcl:"format" yaml:"format"`
	Labels           map[string]string `hcl:"labels" yaml:"labels"`
	RelabelConfigs   []RelabelConfig   `hcl:"relabel" yaml:"relabel"`
	HistogramBuckets []float64         `hcl:"histogram_buckets" yaml:"histogram_buckets"`

	PrintLog bool `hcl:"print_log" yaml:"print_log"`

	OrderedLabelNames  []string
	OrderedLabelValues []string

	// 可以增加额外的指标
	OthersMetrics map[string]*MetricsInfo `hcl:"others_metrics" yaml:"others_metrics"`
}

type MetricsInfo struct {
	// 指标名称
	MetricsName string `hcl:"metrics_name" yaml:"metrics_name"`

	// 指标帮助信息
	MetricsHelp string `hcl:"metrics_help" yaml:"metrics_help"`

	// 0: 设置默认指标为空 1: counter 2: gauge 4: histogram 8: summary，可以二进制组合
	MetricsType int `hcl:"metrics_type" yaml:"metrics_type"`

	// 仅当Type为counter时有效
	HistogramBuckets []float64 `hcl:"histogram_buckets" yaml:"histogram_buckets"`

	// 仅当Type为summary时有效
	MaxAge     time.Duration       `hcl:"max_age" yaml:"max_age"`
	Objectives map[float64]float64 `hcl:"objectives" yaml:"objectives"`
}

const (
	MetricsTypeCounter   = 1
	MetricsTypeGauge     = 2
	MetricsTypeHistogram = 4
	MetricsTypeSummary   = 8
)

type SourceData struct {
	Files  FileSource    `hcl:"files" yaml:"files"`
	Syslog *SyslogSource `hcl:"syslog" yaml:"syslog"`
}

type FileSource []string

type SyslogSource struct {
	ListenAddress string   `hcl:"listen_address" yaml:"listen_address"`
	Format        string   `hcl:"format" yaml:"format"`
	Tags          []string `hcl:"tags" yaml:"tags"`
}

// StabilityWarnings tests if the NamespaceConfig uses any configuration settings
// that are not yet declared "stable"
func (c *NamespaceConfig) StabilityWarnings() error {
	return nil
}

// DeprecationWarnings tests if the NamespaceConfig uses any deprecated
// configuration settings
func (c *NamespaceConfig) DeprecationWarnings() error {
	if len(c.SourceFiles) > 0 {
		return errors.New("you are using the 'source_files' configuration parameter")
	}

	return nil
}

// MustCompile compiles the configuration (mostly regular expressions that are used
// in configuration variables) for later use
func (c *NamespaceConfig) MustCompile() {
	err := c.Compile()
	if err != nil {
		panic(err)
	}
}

// ResolveDeprecations converts any values from depreated fields into the new
// structures
func (c *NamespaceConfig) ResolveDeprecations() {
	if len(c.SourceFiles) > 0 {
		c.SourceData.Files = FileSource(c.SourceFiles)
	}
}

// ResolveGlobs finds globs in file sources and expand them to the actual
// list of files
func (c *NamespaceConfig) ResolveGlobs(logger *log.Logger) error {
	if len(c.SourceData.Files) > 0 {
		resolvedFiles := make([]string, 0)
		for _, sf := range c.SourceData.Files {
			if strings.Contains(sf, "*") {
				matches, err := filepath.Glob(sf)
				if err != nil {
					return err
				}
				logger.Infof("Resolved globs %v to %v", sf, matches)
				resolvedFiles = append(resolvedFiles, matches...)
			} else {
				logger.Warnf("No globs for %v", sf)
				resolvedFiles = append(resolvedFiles, sf)
			}
		}

		// update fields with new list of files
		c.SourceData.Files = resolvedFiles
		c.SourceFiles = resolvedFiles
	}
	return nil
}

// Compile compiles the configuration (mostly regular expressions that are used
// in configuration variables) for later use
func (c *NamespaceConfig) Compile() error {
	for i := range c.RelabelConfigs {
		if err := c.RelabelConfigs[i].Compile(); err != nil {
			return err
		}
	}
	if c.NamespaceLabelName != "" {
		c.NamespaceLabels = make(map[string]string)
		c.NamespaceLabels[c.NamespaceLabelName] = c.Name
	}

	c.OrderLabels()
	c.NamespacePrefix = c.Name
	if c.MetricsOverride != nil {
		c.NamespacePrefix = c.MetricsOverride.Prefix
	}

	return nil
}

// OrderLabels builds two lists of label keys and values, ordered by label name
func (c *NamespaceConfig) OrderLabels() {
	keys := make([]string, 0, len(c.Labels))
	values := make([]string, len(c.Labels))

	for k := range c.Labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for i, k := range keys {
		values[i] = c.Labels[k]
	}

	c.OrderedLabelNames = keys
	c.OrderedLabelValues = values
}
