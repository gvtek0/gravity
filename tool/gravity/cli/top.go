package cli

import (
	"context"
	"fmt"
	"time"

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
	prometheusClient, err := monitoring.NewPrometheus(fmt.Sprintf("http://%v", prometheusAddr))
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
			currentCPU, cpuRate, currentRAM, ramRate, err := getData(prometheusClient, interval)
			if err != nil {
				return trace.Wrap(err)
			}
			draw(currentCPU, cpuRate, currentRAM, ramRate)
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return nil
			}
		}
	}

	return nil
}

func draw(currentCPU int, cpuRate monitoring.Series, currentRAM int, ramRate monitoring.Series) {
	var cpuData []float64
	for _, point := range cpuRate {
		cpuData = append(cpuData, float64(point.Value))
	}

	var ramData []float64
	for _, point := range ramRate {
		ramData = append(ramData, float64(point.Value))
	}

	cpuGauge := widgets.NewGauge()
	cpuGauge.Title = "Current CPU Usage"
	cpuGauge.Percent = currentCPU
	cpuGauge.SetRect(0, 0, 20, 20)

	cpuPlot := widgets.NewPlot()
	cpuPlot.Title = fmt.Sprintf("CPU Usage")
	cpuPlot.Data = [][]float64{cpuData}
	cpuPlot.MaxVal = 100
	cpuPlot.Marker = widgets.MarkerDot
	cpuPlot.SetRect(21, 0, 141, 20)

	ramGauge := widgets.NewGauge()
	ramGauge.Title = "Current RAM Usage"
	ramGauge.Percent = currentRAM
	ramGauge.SetRect(0, 21, 20, 41)

	ramPlot := widgets.NewPlot()
	ramPlot.Title = fmt.Sprintf("RAM Usage")
	ramPlot.Data = [][]float64{ramData}
	ramPlot.MaxVal = 100
	ramPlot.SetRect(21, 21, 141, 41)

	termui.Clear()
	termui.Render(cpuGauge, cpuPlot, ramGauge, ramPlot)
}

func getData(prometheusClient monitoring.Metrics, interval time.Duration) (int, monitoring.Series, int, monitoring.Series, error) {
	currentCPU, err := prometheusClient.GetCurrentCPURate(context.TODO())
	if err != nil {
		return 0, nil, 0, nil, trace.Wrap(err)
	}
	cpuRate, err := prometheusClient.GetCPURate(context.TODO(), time.Now().Add(-interval), time.Now(), 10*time.Second)
	if err != nil {
		return 0, nil, 0, nil, trace.Wrap(err)
	}
	currentRAM, err := prometheusClient.GetCurrentMemoryRate(context.TODO())
	if err != nil {
		return 0, nil, 0, nil, trace.Wrap(err)
	}
	ramRate, err := prometheusClient.GetMemoryRate(context.TODO(), time.Now().Add(-interval), time.Now(), 10*time.Second)
	if err != nil {
		return 0, nil, 0, nil, trace.Wrap(err)
	}
	return currentCPU, cpuRate, currentRAM, ramRate, nil
}

const prometheusService = "prometheus-k8s.monitoring.svc.cluster.local:9090"
