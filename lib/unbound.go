package mpunbound

import (
	"bufio"
	"flag"
	mp "github.com/mackerelio/go-mackerel-plugin"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// UnboundPlugin mackerel plugin for Unbound
type UnboundPlugin struct {
	Prefix      string
	CommandPath string
	ConfPath    string
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p UnboundPlugin) MetricKeyPrefix() string {
	if p.Prefix != "" {
		return p.Prefix
	}
	return "unbound"
}

func parseUnboundStats(r io.Reader) (map[string]float64, error) {
	stat := make(map[string]float64)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		if body := strings.TrimPrefix(line, "total.num."); body != line {
			fields := strings.SplitN(body, "=", 2)
			if len(fields) < 2 {
				continue
			}
			n, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, err
			}
			stat[fields[0]] = n
		}
	}
	return stat, nil
}

// FetchMetrics fetch the metrics
func (p UnboundPlugin) FetchMetrics() (map[string]float64, error) {
	args := []string{}
	if p.ConfPath != "" {
		args = append(args, "-c", p.ConfPath)
	}
	args = append(args, "stats_noreset")
	cmd := exec.Command(p.CommandPath, args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	stats, err := parseUnboundStats(out)
	if err != nil {
		return nil, err
	}
	err = cmd.Wait()
	return stats, err
}

// GraphDefinition of UnboundPlugin
func (p UnboundPlugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := strings.Title(p.Prefix)

	graphdef := map[string]mp.Graphs{
		"num": {
			Label: (labelPrefix + " Traffics"),
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "recursivereplies", Label: "Received replies", Diff: true, Stacked: false},
				{Name: "prefetch", Label: "Cache prefetch", Diff: true, Stacked: false},
				{Name: "cachemiss", Label: "Cache miss", Diff: true, Stacked: true},
				{Name: "cachehits", Label: "Cache hits", Diff: true, Stacked: true},
				{Name: "queries", Label: "Total queries from clients", Diff: true, Stacked: false},
			},
		},
	}
	return graphdef
}

// Do the plugin
func Do() {
	optTempfile := flag.String("tempfile", "", "Temp file name")
	optCommandPath := flag.String("path", "/usr/sbin/unbound-control", "Path of unbound-control")
	optConfPath := flag.String("conf", "", "Path of Unbound config file")
	optPrefix := flag.String("metric-key-prefix", "unbound", "Metric key prefix")
	flag.Parse()

	var plugin UnboundPlugin

	plugin.Prefix = *optPrefix
	plugin.CommandPath = *optCommandPath
	plugin.ConfPath = *optConfPath

	helper := mp.NewMackerelPlugin(plugin)
	helper.Tempfile = *optTempfile

	helper.Run()
}
