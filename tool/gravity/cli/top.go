package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gravitational/gravity/lib/constants"
	"github.com/gravitational/gravity/lib/localenv"
	"github.com/gravitational/gravity/lib/ops/monitoring"

	"github.com/gizak/termui"
	"github.com/gravitational/trace"
)

func top(env *localenv.LocalEnvironment, interval time.Duration) error {
	// prometheusAddr, err := utils.ResolveAddr(env.DNS.Addr(), prometheusService)
	// if err != nil {
	// 	return trace.Wrap(err)
	// }
	prometheusAddr := "192.168.121.97:30198"
	prometheusClient, err := monitoring.NewPrometheus(prometheusAddr)
	if err != nil {
		return trace.Wrap(err)
	}

	if err := termui.Init(); err != nil {
		return trace.Wrap(err)
	}
	defer termui.Close()

	// uiEvents := termui.PollEvents()

	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
				totalCPU, currentCPU, maxCPU, cpuRate, totalRAM, currentRAM, maxRAM, ramRate, err := getData(context.TODO(), prometheusClient, interval)
				if err != nil {
					continue
				}
				draw(totalCPU, currentCPU, maxCPU, cpuRate, totalRAM, currentRAM, maxRAM, ramRate)
				// case e := <-uiEvents:
				// 	switch e.ID {
				// 	case "q", "<C-c>":
				// 		return nil
				// 	}
			}
		}
	}()

	termui.Handle("/sys/kbd/q", func(termui.Event) {
		termui.StopLoop()
	})
	termui.Loop()

	return nil
}

// func draw2(currentCPU, maxCPU int, cpuRate monitoring.Series, currentRAM, maxRAM int, ramRate monitoring.Series) {
// 	chart := goterm.NewLineChart()
// }

func draw(totalCPU, currentCPU, maxCPU int, cpuRate monitoring.Series, totalRAM int64, currentRAM, maxRAM int, ramRate monitoring.Series) {
	var cpuData []float64
	var cpuLabels []string
	for _, point := range cpuRate {
		cpuData = append(cpuData, float64(point.Value))
		cpuLabels = append(cpuLabels, point.Time.Format("15:04"))
	}
	// if len(cpuData) > 140 {
	// 	cpuData = cpuData[len(cpuData)-140:]
	// }

	var ramData []float64
	var ramLabels []string
	for _, point := range ramRate {
		ramData = append(ramData, float64(point.Value))
		ramLabels = append(ramLabels, point.Time.Format("15:04"))
	}
	// if len(ramData) > 140 {
	// 	ramData = ramData[len(ramData)-140:]
	// }

	color := func(percent int) termui.Attribute {
		if percent <= 25 {
			return termui.ColorGreen
		} else if percent > 75 {
			return termui.ColorRed
		} else {
			return termui.ColorYellow
		}
	}

	title := termui.NewPar(fmt.Sprintf("Total Nodes: %v\nTotal CPU Cores: %v\nTotal Memory: %v",
		1, totalCPU, humanize.Bytes(uint64(totalRAM))))
	title.BorderLabel = fmt.Sprintf("Cluster Monitoring - Last Updated: %v",
		time.Now().Format(constants.HumanDateFormatSeconds))
	title.Height = 5
	title.Width = 230

	cpuGauge := termui.NewGauge()
	cpuGauge.BorderLabel = "Current CPU"
	cpuGauge.BorderLabelFg = termui.ColorRed
	cpuGauge.Percent = currentCPU
	cpuGauge.BarColor = color(currentCPU)
	cpuGauge.Height = 10
	cpuGauge.Width = 30
	cpuGauge.X = 0
	cpuGauge.Y = 5

	maxCPUGauge := termui.NewGauge()
	maxCPUGauge.BorderLabel = "Peak CPU"
	maxCPUGauge.BorderLabelFg = termui.ColorRed
	maxCPUGauge.Percent = maxCPU
	maxCPUGauge.BarColor = color(maxCPU)
	maxCPUGauge.Height = 10
	maxCPUGauge.Width = 30
	maxCPUGauge.X = 0
	maxCPUGauge.Y = 15

	cpuPlot := termui.NewLineChart()
	cpuPlot.BorderLabel = fmt.Sprintf("CPU")
	cpuPlot.BorderLabelFg = termui.ColorRed
	cpuPlot.Data = cpuData
	cpuPlot.DataLabels = cpuLabels
	cpuPlot.Mode = "dot"
	cpuPlot.LineColor = termui.ColorRed
	// cpuPlot.AxesColor = termui.ColorCyan
	// cpuPlot.DotStyle = '⦁'
	cpuPlot.Height = 20
	cpuPlot.Width = 200
	cpuPlot.X = 30
	cpuPlot.Y = 5

	ramGauge := termui.NewGauge()
	ramGauge.BorderLabel = "Current RAM"
	ramGauge.BorderLabelFg = termui.ColorBlue
	ramGauge.Percent = currentRAM
	ramGauge.BarColor = color(currentRAM)
	ramGauge.Height = 10
	ramGauge.Width = 30
	ramGauge.X = 0
	ramGauge.Y = 25

	maxRAMGauge := termui.NewGauge()
	maxRAMGauge.BorderLabel = "Peak RAM"
	maxRAMGauge.BorderLabelFg = termui.ColorBlue
	maxRAMGauge.Percent = maxRAM
	maxRAMGauge.BarColor = color(maxRAM)
	maxRAMGauge.Height = 10
	maxRAMGauge.Width = 30
	maxRAMGauge.X = 0
	maxRAMGauge.Y = 35

	ramPlot := termui.NewLineChart()
	ramPlot.BorderLabel = fmt.Sprintf("RAM")
	ramPlot.BorderLabelFg = termui.ColorBlue
	ramPlot.Data = ramData
	ramPlot.DataLabels = ramLabels
	ramPlot.Mode = "dot"
	// ramPlot.DotStyle = '⦁'
	ramPlot.LineColor = termui.ColorBlue
	// ramPlot.AxesColor = termui.ColorCyan
	ramPlot.Height = 20
	ramPlot.Width = 200
	ramPlot.X = 30
	ramPlot.Y = 25

	termui.Clear()
	termui.Render(title, cpuGauge, maxCPUGauge, cpuPlot, ramGauge, maxRAMGauge, ramPlot)
}

func getData(ctx context.Context, prometheusClient monitoring.Metrics, interval time.Duration) (int, int, int, monitoring.Series, int64, int, int, monitoring.Series, error) {
	totalCPU, err := prometheusClient.GetTotalCPU(ctx)
	if err != nil {
		return 0, 0, 0, nil, 0, 0, 9, nil, trace.Wrap(err)
	}
	currentCPU, err := prometheusClient.GetCurrentCPURate(ctx)
	if err != nil {
		return 0, 0, 0, nil, 0, 0, 0, nil, trace.Wrap(err)
	}
	maxCPU, err := prometheusClient.GetMaxCPURate(ctx, interval)
	if err != nil {
		return 0, 0, 0, nil, 0, 0, 0, nil, trace.Wrap(err)
	}
	cpuRate, err := prometheusClient.GetCPURate(ctx, time.Now().Add(-interval), time.Now(), 15*time.Second)
	if err != nil {
		return 0, 0, 0, nil, 0, 0, 0, nil, trace.Wrap(err)
	}
	totalRAM, err := prometheusClient.GetTotalMemory(ctx)
	if err != nil {
		return 0, 0, 0, nil, 0, 0, 0, nil, trace.Wrap(err)
	}
	currentRAM, err := prometheusClient.GetCurrentMemoryRate(ctx)
	if err != nil {
		return 0, 0, 0, nil, 0, 0, 0, nil, trace.Wrap(err)
	}
	maxRAM, err := prometheusClient.GetMaxMemoryRate(ctx, interval)
	if err != nil {
		return 0, 0, 0, nil, 0, 0, 0, nil, trace.Wrap(err)
	}
	ramRate, err := prometheusClient.GetMemoryRate(ctx, time.Now().Add(-interval), time.Now(), 15*time.Second)
	if err != nil {
		return 0, 0, 0, nil, 0, 0, 0, nil, trace.Wrap(err)
	}
	return totalCPU, currentCPU, maxCPU, cpuRate, totalRAM, currentRAM, maxRAM, ramRate, nil
}

const prometheusService = "prometheus-k8s.monitoring.svc.cluster.local:9090"
