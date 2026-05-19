package data

import (
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	"fdi_data_board/internal/data/orm"
)

const slowQueryThreshold = 3 * time.Second

type slowQueryMeta struct {
	QueryType  string
	StartDt    string
	EndDt      string
	EventNames []string
	FilterName string
}

type slowQueryLogger struct {
	db *gorm.DB
}

func newSlowQueryLogger(db *gorm.DB) *slowQueryLogger {
	return &slowQueryLogger{db: db}
}

// Observe 计时执行 fn，超阈值时异步写 MySQL，不阻塞响应。
func (l *slowQueryLogger) Observe(meta slowQueryMeta, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	if duration >= slowQueryThreshold {
		record := &orm.SlowQueryLogDo{
			QueryType:     meta.QueryType,
			DurationMs:    duration.Milliseconds(),
			StartDt:       meta.StartDt,
			EndDt:         meta.EndDt,
			DateRangeDays: calcDateRangeDays(meta.StartDt, meta.EndDt),
			EventNames:    strings.Join(meta.EventNames, ","),
			FilterName:    meta.FilterName,
		}
		go func() {
			if dbErr := l.db.Create(record).Error; dbErr != nil {
				log.Warnf("[slow_query] write failed: %v", dbErr)
			}
		}()
	}
	return err
}

func calcDateRangeDays(startDt, endDt string) int {
	if startDt == "" || endDt == "" {
		return 0
	}
	const layout = "2006-01-02"
	start, err1 := time.Parse(layout, startDt)
	end, err2 := time.Parse(layout, endDt)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(end.Sub(start).Hours()/24) + 1
}
