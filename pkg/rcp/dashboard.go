package rcp

import (
	"context"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

const (
	chanSize = 10
)

// UIIface ui interface
type UIIface interface {
	Init() error
	Close()
	TerminalDimensions() (int, int)
	Render(items ...ui.Drawable)
	PollEvents() <-chan ui.Event
}

type tui struct{}

func (t *tui) Init() error                    { return ui.Init() }
func (t *tui) Close()                         { ui.Close() }
func (t *tui) TerminalDimensions() (int, int) { return ui.TerminalDimensions() }
func (t *tui) Render(items ...ui.Drawable)    { ui.Render(items...) }
func (t *tui) PollEvents() <-chan ui.Event    { return ui.PollEvents() }

type dummyui struct{}

func (t *dummyui) Init() error                    { return nil }
func (t *dummyui) Close()                         {}
func (t *dummyui) TerminalDimensions() (int, int) { return 10, 10 }
func (t *dummyui) Render(items ...ui.Drawable)    {}
func (t *dummyui) PollEvents() <-chan ui.Event    { return make(chan ui.Event) }

// SpeedDashboard status dash board
type SpeedDashboard struct {
	UIIface
	InputName    string
	OutputName   string
	ProgressSize int
	TotalSize    int64

	Title    *widgets.Paragraph
	Output   *widgets.Sparkline
	Input    *widgets.Sparkline
	Buffer   *widgets.Sparkline
	Progress *widgets.Gauge
	Buffers  *widgets.SparklineGroup
	Speeds   *widgets.SparklineGroup
	Metrics
	Ch chan Metrics
}

//Metrics progres metrics
type Metrics struct {
	Size             uint64
	AvgByteSec       uint64
	InputByteSec     uint64
	InputMaxByteSec  uint64
	OutputByteSec    uint64
	OutputMaxByteSec uint64
	BufferUsed       uint64
	BufferMaxUsed    uint64
}

func (s *SpeedDashboard) updateTitle() {
	s.Progress.Title = fmt.Sprintf("Progress:[%s / %s Byte], Average speed:[%syte/sec]",
		humanize.Comma(int64(s.Size)), humanize.Comma(s.TotalSize), humanize.Bytes(s.AvgByteSec))
	s.Input.Title = fmt.Sprintf("Input [%s] %syte/sec (max: %syte/sec)",
		s.InputName, humanize.Bytes(s.InputByteSec), humanize.Bytes(s.InputMaxByteSec))
	s.Output.Title = fmt.Sprintf("Output [%s] %syte/sec (max: %syte/sec)",
		s.OutputName, humanize.Bytes(s.OutputByteSec), humanize.Bytes(s.OutputMaxByteSec))
	s.Buffer.Title = fmt.Sprintf("Buffer used: %syte (max: %syte)",
		humanize.Bytes(s.BufferUsed), humanize.Bytes(s.BufferMaxUsed))
}

func percent(total int64, curr uint64) int {
	if total == 0 {
		return 0
	}
	return int(float64(curr) / float64(total) * 100)
}

func (s *SpeedDashboard) updateData() {
	s.Progress.Percent = percent(s.TotalSize, s.Size)
	s.Buffer.Data = append(s.Buffer.Data, float64(s.BufferUsed))
	s.Output.Data = append(s.Output.Data, float64(s.OutputByteSec))
	s.Input.Data = append(s.Input.Data, float64(s.InputByteSec))
}

// NewSpeedDashboard create SpeedDashboard struct
func NewSpeedDashboard() *SpeedDashboard {
	tu := &tui{}
	s := &SpeedDashboard{
		UIIface:      tu,
		ProgressSize: 3,
		Title:        widgets.NewParagraph(),
		Output:       widgets.NewSparkline(),
		Input:        widgets.NewSparkline(),
		Buffer:       widgets.NewSparkline(),
		Progress:     widgets.NewGauge(),
		Ch:           make(chan Metrics, chanSize),
	}

	s.Title.Text = "PRESS ctrl+[c] TO QUIT"
	s.Title.TextStyle.Fg = ui.ColorWhite
	s.Title.Border = false

	s.Progress.Percent = 0
	s.Progress.BarColor = ui.ColorGreen
	s.Progress.BorderStyle.Fg = ui.ColorWhite
	s.Progress.TitleStyle.Fg = ui.ColorCyan

	s.Input.LineColor = ui.ColorGreen
	s.Input.Data = []float64{0}

	s.Output.LineColor = ui.ColorRed
	s.Output.Data = []float64{0}

	s.Speeds = widgets.NewSparklineGroup(s.Input, s.Output)
	s.Speeds.Title = "Speed"

	s.Buffer.LineColor = ui.ColorYellow
	s.Buffer.Data = []float64{0}

	s.Buffers = widgets.NewSparklineGroup(s.Buffer)
	s.Buffers.Title = "Buffer used"
	return s
}

func (s *SpeedDashboard) resize() {
	speedY := 1
	tw, th := s.TerminalDimensions()
	speedSize := (th - 1 - s.ProgressSize) / 3 * 2
	bufferSize := (th - 1 - s.ProgressSize) / 3 * 1
	bufferY := speedY + speedSize
	progressY := bufferY + bufferSize
	s.Title.SetRect(0, 0, 50, 1)
	s.Progress.SetRect(0, progressY, tw, progressY+s.ProgressSize)
	s.Speeds.SetRect(0, speedY, tw, speedY+speedSize)
	s.Buffers.SetRect(0, bufferY, tw, bufferY+bufferSize)
	s.Output.Data = resizeData(s.Output.Data, tw)
	s.Input.Data = resizeData(s.Input.Data, tw)
	s.Buffer.Data = resizeData(s.Buffer.Data, tw)
}

func resizeData(data []float64, tw int) []float64 {
	s := len(data)
	if s > tw-2 {
		s -= tw - 2
		return data[s:]
	}
	return data
}

// Run speed dashboard
func (s *SpeedDashboard) Run(ctx context.Context) error {
	if err := s.Init(); err != nil {
		return err
	}
	defer s.Close()

	uiEvents := s.PollEvents()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case s.Metrics = <-s.Ch:
		case e := <-uiEvents:
			if e.ID == "<C-c>" {
				return nil
			}
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.updateData()
			s.resize()
			s.updateTitle()
			s.Render(s.Title, s.Progress, s.Speeds, s.Buffers)
		}
	}
}
