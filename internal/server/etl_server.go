package server

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"

	"fdi_data_board/internal/biz"
	"fdi_data_board/internal/conf"
)

const (
	defaultEtlBackfillDays = 180
	defaultEtlRefreshDays  = 7
	defaultEtlLockKey      = "fdi_data_board:etl:scheduler"
	defaultEtlLockTTL      = 2 * time.Minute
	defaultEtlLockRefresh  = 30 * time.Second
)

type ETLServer struct {
	uc         *biz.ETLUseCase
	conf       *conf.Data
	log        *log.Helper
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	redis      redis.UniversalClient
	lockScript *redis.Script
	delScript  *redis.Script
}

type etlRuntimeConfig struct {
	on           bool
	backfillOn   bool
	backfillDays int
	refreshDays  int
	lockKey      string
	lockTTL      time.Duration
	lockRefresh  time.Duration
}

func NewETLServer(uc *biz.ETLUseCase, cd *conf.Data, redisClient redis.UniversalClient, logger log.Logger) *ETLServer {
	return &ETLServer{
		uc:    uc,
		conf:  cd,
		log:   log.NewHelper(logger),
		redis: redisClient,
		lockScript: redis.NewScript(`
					if redis.call("get", KEYS[1]) == ARGV[1] then
						return redis.call("pexpire", KEYS[1], ARGV[2])
				end
				return 0
			`),
		delScript: redis.NewScript(`
				if redis.call("get", KEYS[1]) == ARGV[1] then
					return redis.call("del", KEYS[1])
				end
				return 0
			`),
	}
}

func (s *ETLServer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.log.Info("[etl] server starting")

	cfg := s.runtimeConfig()
	if !cfg.on {
		s.log.Info("[etl] scheduler disabled by config")
		return nil
	}

	if s.redis == nil {
		s.log.Warn("[etl] redis client unavailable, scheduler disabled")
		return nil
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runScheduler(ctx, cfg)
	}()

	return nil
}

func (s *ETLServer) Stop(_ context.Context) error {
	s.log.Info("[etl] server stopping")
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	if s.redis != nil {
		_ = s.redis.Close()
	}
	return nil
}

func (s *ETLServer) runScheduler(ctx context.Context, cfg etlRuntimeConfig) {
	if cfg.backfillOn {
		s.runBackfill(ctx, cfg)
	}

	s.runDailyRefreshLoop(ctx, cfg)
}

func (s *ETLServer) runBackfill(ctx context.Context, cfg etlRuntimeConfig) {
	lockKey := cfg.lockKey + ":backfill"
	acquired, err := s.withTaskLock(ctx, lockKey, cfg, func(taskCtx context.Context) {
		s.log.Infof("[etl] backfill lock acquired, starting backfill for %d days", cfg.backfillDays)
		s.uc.Backfill(taskCtx, cfg.backfillDays)
		s.log.Info("[etl] backfill done")
	})
	if err != nil {
		s.log.Errorf("[etl] backfill failed: %v", err)
		return
	}
	if !acquired {
		s.log.Infof("[etl] backfill skipped, lock is held by another instance: key=%s", lockKey)
	}
}

func (s *ETLServer) runDailyRefreshLoop(ctx context.Context, cfg etlRuntimeConfig) {
	for {
		if ctx.Err() != nil {
			s.log.Info("[etl] scheduler stopped")
			return
		}

		next := nextRunTime("Asia/Shanghai", 10, 0)

		select {
		case <-ctx.Done():
			s.log.Info("[etl] scheduler stopped")
			return
		case <-time.After(time.Until(next)):
		}

		runDate := next.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
		lockKey := fmt.Sprintf("%s:refresh:%s", cfg.lockKey, runDate)
		acquired, err := s.withTaskLock(ctx, lockKey, cfg, func(taskCtx context.Context) {
			s.uc.RunWindow(taskCtx, cfg.refreshDays)
		})
		if err != nil {
			s.log.Errorf("[etl] daily refresh failed for %s: %v", runDate, err)
			continue
		}
		if !acquired {
			s.log.Infof("[etl] daily refresh skipped for %s, lock is held by another instance", runDate)
		}
	}
}

func (s *ETLServer) withTaskLock(ctx context.Context, lockKey string, cfg etlRuntimeConfig, fn func(context.Context)) (bool, error) {
	token, acquired, err := s.acquireLock(ctx, lockKey, cfg.lockTTL)
	if err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}

	taskCtx, cancel := context.WithCancel(ctx)
	renewDone := make(chan struct{})
	// 续租goroutine
	go s.renewLockLoop(taskCtx, cancel, lockKey, token, cfg.lockTTL, cfg.lockRefresh, renewDone)

	fn(taskCtx)

	cancel()
	<-renewDone
	if err := s.releaseLock(context.Background(), lockKey, token); err != nil {
		s.log.Warnf("[etl] release lock failed for key=%s: %v", lockKey, err)
	}
	return true, nil
}

func (s *ETLServer) renewLockLoop(
	ctx context.Context,
	cancel context.CancelFunc,
	lockKey string,
	token string,
	lockTTL time.Duration,
	refreshInterval time.Duration,
	done chan struct{},
) {
	defer close(done)

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ok, err := s.refreshLock(ctx, lockKey, token, lockTTL)
			if err != nil {
				s.log.Errorf("[etl] refresh lock failed for key=%s: %v", lockKey, err)
				cancel()
				return
			}
			if !ok {
				s.log.Warnf("[etl] redis lock lost for key=%s, task will stop", lockKey)
				cancel()
				return
			}
		}
	}
}

func (s *ETLServer) acquireLock(ctx context.Context, lockKey string, lockTTL time.Duration) (string, bool, error) {
	token := uuid.NewString()
	ok, err := s.redis.SetNX(ctx, lockKey, token, lockTTL).Result()
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	return token, true, nil
}

func (s *ETLServer) refreshLock(ctx context.Context, lockKey, token string, lockTTL time.Duration) (bool, error) {
	res, err := s.lockScript.Run(ctx, s.redis, []string{lockKey}, token, lockTTL.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

func (s *ETLServer) releaseLock(ctx context.Context, lockKey, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.delScript.Run(ctx, s.redis, []string{lockKey}, token).Int64()
	return err
}

func (s *ETLServer) runtimeConfig() etlRuntimeConfig {
	cfg := etlRuntimeConfig{
		on:           true,
		backfillOn:   true,
		backfillDays: defaultEtlBackfillDays,
		refreshDays:  defaultEtlRefreshDays,
		lockKey:      defaultEtlLockKey,
		lockTTL:      defaultEtlLockTTL,
		lockRefresh:  defaultEtlLockRefresh,
	}

	if etl := s.conf.GetEtl(); etl != nil {
		cfg.on = parseConfigBool(etl.GetOn(), cfg.on)
		cfg.backfillOn = parseConfigBool(etl.GetBackfillOn(), cfg.backfillOn)
		if etl.GetBackfillDays() > 0 {
			cfg.backfillDays = int(etl.GetBackfillDays())
		}
		if etl.GetRefreshDays() > 0 {
			cfg.refreshDays = int(etl.GetRefreshDays())
		}
		if etl.GetLockKey() != "" {
			cfg.lockKey = etl.GetLockKey()
		}
		if etl.GetLockTtl() != nil && etl.GetLockTtl().AsDuration() > 0 {
			cfg.lockTTL = etl.GetLockTtl().AsDuration()
		}
		if etl.GetLockRefreshInterval() != nil && etl.GetLockRefreshInterval().AsDuration() > 0 {
			cfg.lockRefresh = etl.GetLockRefreshInterval().AsDuration()
		}
	}

	if cfg.lockRefresh >= cfg.lockTTL {
		cfg.lockRefresh = cfg.lockTTL / 2
		if cfg.lockRefresh <= 0 {
			cfg.lockRefresh = time.Second
		}
	}

	return cfg
}

func parseConfigBool(value string, defaultValue bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return defaultValue
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// nextRunTime 返回指定时区 hour:minute 的下一次触发时刻
func nextRunTime(timezone string, hour, minute int) time.Time {
	loc, _ := time.LoadLocation(timezone)
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
