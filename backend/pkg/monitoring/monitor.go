package monitoring

import (
	"context"
	"runtime"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// MonitorConfig 监控配置
type MonitorConfig struct {
	// 性能阈值
	ResponseTimeP95Threshold float64 // API响应时间P95阈值（秒）
	ResponseTimeP99Threshold float64 // API响应时间P99阈值（秒）
	DBQueryTimeThreshold     float64 // 数据库查询时间阈值（秒）

	// 资源阈值
	CPUThreshold    float64 // CPU使用率阈值（百分比）
	MemoryThreshold float64 // 内存使用率阈值（百分比）
	DiskThreshold   float64 // 磁盘使用率阈值（百分比）

	// 检查间隔
	CheckInterval time.Duration // 检查间隔
}

// DefaultMonitorConfig 默认监控配置
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		ResponseTimeP95Threshold: 0.2,   // 200ms
		ResponseTimeP99Threshold: 0.5,   // 500ms
		DBQueryTimeThreshold:     0.1,   // 100ms
		CPUThreshold:             80.0,  // 80%
		MemoryThreshold:          85.0,  // 85%
		DiskThreshold:            90.0,  // 90%
		CheckInterval:            1 * time.Minute,
	}
}

// Monitor 监控守护进程
type Monitor struct {
	config         MonitorConfig
	healthService  *HealthService
	metricsService *MetricsService
	alertService   *AlertService
	log            *log.Helper
	ticker         *time.Ticker
	done           chan struct{}
}

// NewMonitor 创建监控守护进程
func NewMonitor(
	config MonitorConfig,
	healthService *HealthService,
	metricsService *MetricsService,
	alertService *AlertService,
	logger log.Logger,
) *Monitor {
	return &Monitor{
		config:         config,
		healthService:  healthService,
		metricsService: metricsService,
		alertService:   alertService,
		log:            log.NewHelper(log.With(logger, "module", "monitor")),
		ticker:         time.NewTicker(config.CheckInterval),
		done:           make(chan struct{}),
	}
}

// Start 启动监控
func (m *Monitor) Start(ctx context.Context) error {
	m.log.Info("starting monitor")

	go func() {
		for {
			select {
			case <-m.ticker.C:
				m.check(ctx)
			case <-m.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop 停止监控
func (m *Monitor) Stop(ctx context.Context) error {
	m.log.Info("stopping monitor")
	m.ticker.Stop()
	close(m.done)
	return nil
}

// check 执行检查
func (m *Monitor) check(ctx context.Context) {
	// 检查系统健康
	m.checkHealth(ctx)

	// 检查性能指标
	m.checkPerformance(ctx)

	// 检查资源使用
	m.checkResources(ctx)
}

// checkHealth 检查系统健康
func (m *Monitor) checkHealth(ctx context.Context) {
	health := m.healthService.Check(ctx)

	if health.Status == HealthStatusDOWN {
		// 发送系统异常告警
		for name, component := range health.Components {
			if component.Status == HealthStatusDOWN {
				m.alertService.SendSystemError(ctx,
					"系统组件异常: "+name,
					"组件 "+name+" 状态异常: "+component.Error,
				)
			}
		}
	}
}

// checkPerformance 检查性能指标
func (m *Monitor) checkPerformance(ctx context.Context) {
	stats := GetResponseTimeStats("POST", "/api")

	// 检查P95响应时间
	if stats.P95 > m.config.ResponseTimeP95Threshold {
		m.alertService.SendPerformanceAlert(ctx,
			"API响应时间P95",
			stats.P95,
			m.config.ResponseTimeP95Threshold,
		)
	}

	// 检查P99响应时间
	if stats.P99 > m.config.ResponseTimeP99Threshold {
		m.alertService.SendPerformanceAlert(ctx,
			"API响应时间P99",
			stats.P99,
			m.config.ResponseTimeP99Threshold,
		)
	}
}

// checkResources 检查资源使用
func (m *Monitor) checkResources(ctx context.Context) {
	// 检查CPU使用率
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		if cpuPercent[0] > m.config.CPUThreshold {
			m.alertService.SendResourceAlert(ctx,
				"CPU",
				cpuPercent[0],
				m.config.CPUThreshold,
			)
		}
	}

	// 检查内存使用率
	if memInfo, err := mem.VirtualMemory(); err == nil {
		if memInfo.UsedPercent > m.config.MemoryThreshold {
			m.alertService.SendResourceAlert(ctx,
				"内存",
				memInfo.UsedPercent,
				m.config.MemoryThreshold,
			)
		}
	}

	// 检查磁盘使用率
	if diskInfo, err := disk.Usage("/"); err == nil {
		if diskInfo.UsedPercent > m.config.DiskThreshold {
			m.alertService.SendResourceAlert(ctx,
				"磁盘",
				diskInfo.UsedPercent,
				m.config.DiskThreshold,
			)
		}
	}
}

// GetSystemInfo 获取系统信息
func GetSystemInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// CPU信息
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		info["cpu_percent"] = cpuPercent[0]
	}
	info["cpu_count"] = runtime.NumCPU()

	// 内存信息
	if memInfo, err := mem.VirtualMemory(); err == nil {
		info["memory_total"] = memInfo.Total
		info["memory_used"] = memInfo.Used
		info["memory_percent"] = memInfo.UsedPercent
	}

	// 磁盘信息
	if diskInfo, err := disk.Usage("/"); err == nil {
		info["disk_total"] = diskInfo.Total
		info["disk_used"] = diskInfo.Used
		info["disk_percent"] = diskInfo.UsedPercent
	}

	// Goroutine数量
	info["goroutine_count"] = runtime.NumGoroutine()

	return info
}
