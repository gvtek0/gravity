package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/gravitational/gravity/lib/constants"
	"github.com/gravitational/gravity/lib/localenv"
	"github.com/gravitational/gravity/lib/ops/monitoring"

	"github.com/gizak/termui"
	"github.com/gizak/termui/widgets"
	"github.com/gravitational/trace"
)

func top(env *localenv.LocalEnvironment, interval time.Duration) error {
	// prometheusAddr, err := utils.ResolveAddr(env.DNS.Addr(), prometheusService)
	// if err != nil {
	// 	return trace.Wrap(err)
	// }
	prometheusAddr := "192.168.121.133:32634"
	prometheusClient, err := monitoring.NewPrometheus(prometheusAddr)
	if err != nil {
		return trace.Wrap(err)
	}

	if err := termui.Init(); err != nil {
		return trace.Wrap(err)
	}
	defer termui.Close()
	uiEvents := termui.PollEvents()

	for {
		select {
		case <-time.After(2 * time.Second):
			currentCPU, maxCPU, cpuRate, currentRAM, maxRAM, ramRate, err := getData(context.TODO(), prometheusClient, interval)
			if err != nil {
				return trace.Wrap(err)
			}
			draw(currentCPU, maxCPU, cpuRate, currentRAM, maxRAM, ramRate)
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return nil
			}
		}
	}

	return nil
}

func draw(currentCPU, maxCPU int, cpuRate monitoring.Series, currentRAM, maxRAM int, ramRate monitoring.Series) {
	var cpuData []float64
	for _, point := range cpuRate {
		cpuData = append(cpuData, float64(point.Value))
	}
	if len(cpuData) > 140 {
		cpuData = cpuData[len(cpuData)-140:]
	}

	var ramData []float64
	for _, point := range ramRate {
		ramData = append(ramData, float64(point.Value))
	}
	if len(ramData) > 140 {
		ramData = ramData[len(ramData)-140:]
	}

	color := func(percent int) termui.Color {
		if percent <= 25 {
			return termui.ColorGreen
		} else if percent > 75 {
			return termui.ColorRed
		} else {
			return termui.ColorYellow
		}
	}

	title := widgets.NewParagraph()
	title.Title = "Cluster Monitoring"
	title.Text = fmt.Sprintf("Last Updated: %v",
		time.Now().Format(constants.HumanDateFormatSeconds))
	title.SetRect(0, 0, 182, 5)

	cpuGauge := widgets.NewGauge()
	cpuGauge.Title = "Current CPU"
	cpuGauge.Percent = currentCPU
	cpuGauge.BarColor = color(currentCPU)
	cpuGauge.SetRect(0, 5, 15, 25)

	maxCPUGauge := widgets.NewGauge()
	maxCPUGauge.Title = "Peak CPU"
	maxCPUGauge.Percent = maxCPU
	maxCPUGauge.BarColor = color(maxCPU)
	maxCPUGauge.SetRect(16, 5, 31, 25)

	cpuPlot := widgets.NewPlot()
	cpuPlot.Title = fmt.Sprintf("CPU Usage")
	cpuPlot.Data = [][]float64{cpuData}
	cpuPlot.MaxVal = 100
	cpuPlot.Marker = widgets.MarkerDot
	cpuPlot.DotMarkerRune = '•'
	cpuPlot.SetRect(32, 5, 182, 25)
	//cpuPlot.SetRect(0, 0, 100, 20)

	_ = termui.DOT

	ramGauge := widgets.NewGauge()
	ramGauge.Title = "Current RAM"
	ramGauge.Percent = currentRAM
	ramGauge.BarColor = color(currentRAM)
	ramGauge.SetRect(0, 25, 15, 45)

	maxRAMGauge := widgets.NewGauge()
	maxRAMGauge.Title = "Peak RAM"
	maxRAMGauge.Percent = maxRAM
	maxRAMGauge.BarColor = color(maxRAM)
	maxRAMGauge.SetRect(16, 25, 31, 45)

	ramPlot := widgets.NewPlot()
	ramPlot.Title = fmt.Sprintf("RAM Usage")
	ramPlot.Data = [][]float64{ramData}
	ramPlot.Marker = widgets.MarkerDot
	ramPlot.DotMarkerRune = '•'
	ramPlot.MaxVal = 100
	ramPlot.SetRect(32, 25, 182, 45)
	// ramPlot.SetRect(0, 21, 100, 41)
	// ramPlot.SetRect(0, 0, 230, 54)

	termui.Clear()
	// termui.Render(ramPlot)
	termui.Render(title, cpuGauge, maxCPUGauge, cpuPlot, ramGauge, maxRAMGauge, ramPlot)
}

func getData(ctx context.Context, prometheusClient monitoring.Metrics, interval time.Duration) (int, int, monitoring.Series, int, int, monitoring.Series, error) {
	currentCPU, err := prometheusClient.GetCurrentCPURate(ctx)
	if err != nil {
		return 0, 0, nil, 0, 0, nil, trace.Wrap(err)
	}
	maxCPU, err := prometheusClient.GetMaxCPURate(ctx, interval)
	if err != nil {
		return 0, 0, nil, 0, 0, nil, trace.Wrap(err)
	}
	cpuRate, err := prometheusClient.GetCPURate(ctx, time.Now().Add(-interval), time.Now(), 15*time.Second)
	if err != nil {
		return 0, 0, nil, 0, 0, nil, trace.Wrap(err)
	}
	currentRAM, err := prometheusClient.GetCurrentMemoryRate(ctx)
	if err != nil {
		return 0, 0, nil, 0, 0, nil, trace.Wrap(err)
	}
	maxRAM, err := prometheusClient.GetMaxMemoryRate(ctx, interval)
	if err != nil {
		return 0, 0, nil, 0, 0, nil, trace.Wrap(err)
	}
	ramRate, err := prometheusClient.GetMemoryRate(ctx, time.Now().Add(-interval), time.Now(), 15*time.Second)
	if err != nil {
		return 0, 0, nil, 0, 0, nil, trace.Wrap(err)
	}
	return currentCPU, maxCPU, cpuRate, currentRAM, maxRAM, ramRate, nil
}

const prometheusService = "prometheus-k8s.monitoring.svc.cluster.local:9090"
