package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
)

var (
	cpuUsage    float64
	memoryUsage float64
	diskUsage   float64
	networkIn   int
	networkOut  int
	processList []Process
)

type Process struct {
	PID    int
	Name   string
	CPU    float64
	Memory float64
	Status string
}

const (
	graphWidth  = 76
	graphHeight = 10
)

var (
	cpuUsageHistory []float64
)

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			updateData()
			g.Update(func(g *gocui.Gui) error {
				drawDashboard(g)
				return nil
			})
		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func updateData() {

	cpuUsage = rand.Float64() * 100
	memoryUsage = rand.Float64() * 100
	diskUsage = rand.Float64() * 100
	networkIn = rand.Intn(1000) // KB/s
	networkOut = rand.Intn(500) // KB/s

	processList = []Process{
		{PID: 1, Name: "docker", CPU: rand.Float64() * 0.5, Memory: rand.Float64() * 1.0, Status: "Running"},
		{PID: 123, Name: "go-app", CPU: rand.Float64() * 15.0, Memory: rand.Float64() * 5.0, Status: "Running"},
		{PID: 456, Name: "nginx", CPU: rand.Float64() * 2.0, Memory: rand.Float64() * 1.5, Status: "Running"},
		{PID: 789, Name: "db-server", CPU: rand.Float64() * 10.0, Memory: rand.Float64() * 8.0, Status: "Running"},
	}

	cpuUsageHistory = append(cpuUsageHistory, cpuUsage)
	if len(cpuUsageHistory) > graphWidth {
		cpuUsageHistory = cpuUsageHistory[1:]
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("hardware", 0, 0, maxX/4-1, maxY/3-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = " Hardware "
		v.Autoscroll = false
		v.Wrap = true
	}

	if v, err := g.SetView("network", maxX/4, 0, maxX/2-1, maxY/3-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = " Network (KB/s) "
		v.Autoscroll = true
		v.Wrap = true
	}

	if v, err := g.SetView("processes", 0, maxY/3, maxX/2-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = " Running Processes "
		v.Clear()
		v.Autoscroll = false
		v.Wrap = false
		v.Highlight = true
		//v.SelBgColor = gocui.ColorGreen
		//v.SelFgColor = gocui.ColorBlack
	}

	if v, err := g.SetView("cpugraph", maxX/2+1, 0, maxX-1, maxY/3-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "CPU history"
	}

	return nil
}

func drawDashboard(g *gocui.Gui) {
	if v, err := g.View("hardware"); err == nil {
		v.Clear()
		fmt.Fprintf(v, "CPU:    %s %.2f%%\n", getColorBar(cpuUsage), cpuUsage)
		fmt.Fprintf(v, "Memory: %s %.2f%%\n", getColorBar(memoryUsage), memoryUsage)
		fmt.Fprintf(v, "Disk:   %s %.2f%%\n", getColorBar(diskUsage), diskUsage)
	}

	if v, err := g.View("network"); err == nil {
		v.Clear()
		fmt.Fprintf(v, "In:  %d KB/s\n", networkIn)
		fmt.Fprintf(v, "Out: %d KB/s\n", networkOut)
	}

	if v, err := g.View("processes"); err == nil {
		v.Clear()
		fmt.Fprintf(v, "%-8s %-20s %-8s %-8s %-10s\n", "PID", "Name", "CPU%", "Mem%", "Status")
		fmt.Fprintln(v, strings.Repeat("-", 60))

		for _, p := range processList {
			fmt.Fprintf(v, "%-8d %-20s %-8.2f %-8.2f %-10s\n", p.PID, p.Name, p.CPU, p.Memory, p.Status)
		}
	}

	if v, err := g.View("cpugraph"); err == nil {
		v.Clear()
		out := renderGraph(cpuUsageHistory, graphWidth, graphHeight)
		fmt.Fprintf(v, "%s\n", out)
	}

	g.Update(func(g *gocui.Gui) error {
		return nil
	})
}

func getColorBar(percentage float64) string {
	barLen := 10
	filled := int(percentage / 100 * float64(barLen))

	var color gocui.Attribute
	switch {
	case percentage > 80:
		color = gocui.ColorRed | gocui.AttrBold
	case percentage > 50:
		color = gocui.ColorYellow | gocui.AttrBold
	default:
		color = gocui.ColorGreen | gocui.AttrBold
	}

	bar := strings.Repeat("█", filled) + strings.Repeat(" ", barLen-filled)

	return fmt.Sprintf("\033[38;5;%dm%s\033[0m", colorCode(color), bar)
}

func colorCode(c gocui.Attribute) int {
	switch c & 0xFF { // Only the base color
	case gocui.ColorRed:
		return 31 // Red
	case gocui.ColorYellow:
		return 33 // Yellow
	case gocui.ColorGreen:
		return 32 // Green
	default:
		return 0 // White
	}
}

func renderGraph(history []float64, width, height int) string {
	lines := make([]string, height)
	historyLen := len(history)

	for y := 0; y < height; y++ {
		thresh := float64(height-y-1) * 100 / float64(height-1)
		var row strings.Builder

		for x := 0; x < width; x++ {

			histIdx := historyLen - width + x
			var val float64

			if histIdx >= 0 && histIdx < historyLen {
				val = history[histIdx]
			} else {
				val = 0
			}
			if val >= thresh {
				row.WriteString("█")
			} else {
				row.WriteString(" ")
			}
		}
		lines[y] = row.String()
	}
	return strings.Join(lines, "\n")
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
