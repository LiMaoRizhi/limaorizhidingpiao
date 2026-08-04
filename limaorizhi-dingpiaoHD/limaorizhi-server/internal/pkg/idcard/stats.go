package idcard

import (
	"sync/atomic"
	"time"
)

// VerifyStats 实名认证统计快照（进程级，重启清零）
type VerifyStats struct {
	CacheHits    int64     `json:"cache_hits"`     // 缓存命中次数（直接返回，未调用云市场 API）
	CacheMisses  int64     `json:"cache_misses"`   // 缓存未命中次数（需调用云市场 API）
	APICalls     int64     `json:"api_calls"`      // 实际调用云市场 API 的次数（含重试，每次 doVerify 计 1）
	APIErrors    int64     `json:"api_errors"`     // API 调用失败次数（网络错误/超时/状态码非200等）
	CacheWrites  int64     `json:"cache_writes"`   // 缓存写入次数（API 成功后写入）
	CacheDeletes int64     `json:"cache_deletes"`  // 缓存主动删除次数（管理端触发）
	HitRate      float64   `json:"hit_rate"`       // 缓存命中率 = cache_hits / (cache_hits + cache_misses)
	Since        time.Time `json:"since"`          // 统计起始时间（进程启动时间或最近一次 Reset 时间）
}

var (
	statsCacheHits    int64
	statsCacheMisses  int64
	statsAPICalls     int64
	statsAPIErrors    int64
	statsCacheWrites  int64
	statsCacheDeletes int64
	statsSince        = time.Now()
)

func incCacheHit()   { atomic.AddInt64(&statsCacheHits, 1) }
func incCacheMiss()  { atomic.AddInt64(&statsCacheMisses, 1) }
func incAPICall()    { atomic.AddInt64(&statsAPICalls, 1) }
func incAPIError()   { atomic.AddInt64(&statsAPIErrors, 1) }
func incCacheWrite() { atomic.AddInt64(&statsCacheWrites, 1) }

// IncCacheDelete 管理端主动删除缓存时调用
func IncCacheDelete() { atomic.AddInt64(&statsCacheDeletes, 1) }

func GetVerifyStats() VerifyStats {
	hits := atomic.LoadInt64(&statsCacheHits)
	misses := atomic.LoadInt64(&statsCacheMisses)
	total := hits + misses
	var rate float64
	if total > 0 {
		rate = float64(hits) / float64(total)
	}
	return VerifyStats{
		CacheHits:    hits,
		CacheMisses:  misses,
		APICalls:     atomic.LoadInt64(&statsAPICalls),
		APIErrors:    atomic.LoadInt64(&statsAPIErrors),
		CacheWrites:  atomic.LoadInt64(&statsCacheWrites),
		CacheDeletes: atomic.LoadInt64(&statsCacheDeletes),
		HitRate:      rate,
		Since:        statsSince,
	}
}

func ResetVerifyStats() {
	atomic.StoreInt64(&statsCacheHits, 0)
	atomic.StoreInt64(&statsCacheMisses, 0)
	atomic.StoreInt64(&statsAPICalls, 0)
	atomic.StoreInt64(&statsAPIErrors, 0)
	atomic.StoreInt64(&statsCacheWrites, 0)
	atomic.StoreInt64(&statsCacheDeletes, 0)
	statsSince = time.Now()
}
