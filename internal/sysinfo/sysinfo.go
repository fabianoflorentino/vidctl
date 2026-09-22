// Package sysinfo samples system resource usage (CPU, memory, GPU) so the
// UI can show live load while a conversion runs.
package sysinfo

import (
	"context"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/fabianoflorentino/vidctl/internal/cmdutil"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

// Snapshot is a point-in-time usage report exposed to the frontend.
type Snapshot struct {
	CPU        float64 `json:"cpu"`        // system-wide % (0–100, all cores normalized)
	MemUsedMB  float64 `json:"memUsedMB"`  //
	MemTotalMB float64 `json:"memTotalMB"` //
	FFmpegCPU  float64 `json:"ffmpegCpu"`  // ffmpeg processes as % of the whole machine
	GPU        float64 `json:"gpu"`        // NVIDIA utilization %; -1 when unavailable
}

// Collector samples usage on demand and caches GPU availability.
type Collector struct {
	mu        sync.Mutex
	gpuProbed bool
	gpuOK     bool
}

// NewCollector creates a collector.
func NewCollector() *Collector { return &Collector{} }

// Snapshot reads all available metrics; unavailable sources degrade to zero/-1.
func (c *Collector) Snapshot() Snapshot {
	ctx := context.Background()
	s := Snapshot{GPU: -1}

	if p, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(p) > 0 {
		s.CPU = clampPct(p[0])
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		s.MemTotalMB = float64(vm.Total) / (1024 * 1024)
		s.MemUsedMB = float64(vm.Used) / (1024 * 1024)
	}
	s.FFmpegCPU = ffmpegCPUPercent(ctx)
	if pct, ok := c.nvidiaPercent(ctx); ok {
		s.GPU = pct
	}
	return s
}

func ffmpegCPUPercent(ctx context.Context) float64 {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return 0
	}
	var sum float64
	for _, p := range procs {
		name, err := p.Name()
		if err != nil || !isFFmpeg(name) {
			continue
		}
		if pct, err := p.CPUPercent(); err == nil {
			sum += pct
		}
	}
	return clampPct(sum / float64(runtime.NumCPU()))
}

func isFFmpeg(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return strings.HasPrefix(base, "ffmpeg")
}

func (c *Collector) nvidiaPercent(ctx context.Context) (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gpuProbed && !c.gpuOK {
		return 0, false
	}
	bin, err := cmdutil.LookPath("nvidia-smi")
	if err == nil {
		var out []byte
		out, err = cmdutil.CommandContext(ctx, bin,
			"--query-gpu=utilization.gpu", "--format=csv,noheader,nounits").Output()
		if err == nil {
			if pct, ok := parseNvidia(out); ok {
				c.gpuProbed, c.gpuOK = true, true
				return pct, true
			}
		}
	}
	c.gpuProbed, c.gpuOK = true, false
	return 0, false
}

// parseNvidia reads nvidia-smi CSV output ("42\n55") and returns the max %.
func parseNvidia(out []byte) (float64, bool) {
	best := -1.0
	for _, line := range strings.Fields(string(out)) {
		line = strings.TrimRight(line, "%,")
		v, err := strconv.ParseFloat(line, 64)
		if err != nil {
			continue
		}
		if v > best {
			best = v
		}
	}
	if best < 0 {
		return 0, false
	}
	return clampPct(best), true
}

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
