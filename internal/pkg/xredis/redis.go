package xredis

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	READ_TIMEOUT = 30 * time.Second
)

var defaultRedisClient *RedisClient

func SetupRedisFromConfigStruct(redisConfig *RedisConfig) error {
	if len(redisConfig.Addr) == 0 {
		return errors.New("redis addr is empty")
	}
	c, err := setupRedis(redisConfig)
	if err != nil {
		return err
	}
	defaultRedisClient = c
	return nil
}

func Close() error {
	return defaultRedisClient.close()
}

func GetClient() redis.UniversalClient {
	return defaultRedisClient.getClient()
}

func Get(ctx context.Context, key string) (string, bool, error) {
	begin := time.Now()
	result, err := getString(ctx, key)
	if err != nil && err != redis.Nil {
		return "", false, err
	} else if err == redis.Nil {
		return "", false, nil
	}
	ObserveRedisLatency("get", time.Since(begin).Seconds())
	return result, true, nil
}

func GetInt(ctx context.Context, key string) (int, bool, error) {
	begin := time.Now()
	result, found, err := Get(ctx, key)
	if err != nil {
		return 0, false, err
	} else if !found {
		return 0, false, nil
	}
	ObserveRedisLatency("get", time.Since(begin).Seconds())
	vi, err := strconv.Atoi(result)
	if err != nil {
		return 0, true, err
	}

	return vi, true, nil
}

func GetInt64(ctx context.Context, key string) (int64, bool, error) {
	begin := time.Now()
	result, found, err := Get(ctx, key)
	if err != nil {
		return 0, false, err
	} else if !found {
		return 0, false, nil
	}
	ObserveRedisLatency("get", time.Since(begin).Seconds())
	vi, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, true, err
	}
	return vi, true, nil
}

func GetUint64(ctx context.Context, key string) (uint64, bool, error) {
	begin := time.Now()
	result, found, err := Get(ctx, key)
	if err != nil {
		return 0, false, err
	} else if !found {
		return 0, false, nil
	}
	ObserveRedisLatency("get", time.Since(begin).Seconds())
	vi, err := strconv.ParseUint(result, 10, 64)
	if err != nil {
		return 0, found, err
	}
	return vi, true, nil
}

func getString(ctx context.Context, key string) (string, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	ObserveRedisLatency("get", time.Since(begin).Seconds())
	return result, nil
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	begin := time.Now()
	err := defaultRedisClient.c.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return err
	}
	ObserveRedisLatency("set", time.Since(begin).Seconds())
	return nil
}

func Del(ctx context.Context, key string) error {
	begin := time.Now()
	err := defaultRedisClient.c.Del(ctx, key).Err()
	if err != nil {
		return err
	}
	ObserveRedisLatency("del", time.Since(begin).Seconds())
	return nil
}

func Incr(ctx context.Context, key string) (int64, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	ObserveRedisLatency("incr", time.Since(begin).Seconds())
	return result, nil
}

func Decr(ctx context.Context, key string) (int64, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.Decr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	ObserveRedisLatency("decr", time.Since(begin).Seconds())
	return result, nil
}

func IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, err
	}
	ObserveRedisLatency("incrBy", time.Since(begin).Seconds())
	return result, nil
}

func DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.DecrBy(ctx, key, value).Result()
	if err != nil {
		return 0, err
	}
	ObserveRedisLatency("decrBy", time.Since(begin).Seconds())
	return result, nil
}

func Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return defaultRedisClient.c.Expire(ctx, key, expiration).Result()
}

func TTL(ctx context.Context, key string) (time.Duration, error) {
	return defaultRedisClient.c.TTL(ctx, key).Result()
}

func Exists(ctx context.Context, key string) (bool, error) {
	ret, err := defaultRedisClient.c.Exists(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return false, err
	} else if err == redis.Nil {
		return false, nil
	}

	return ret == 1, nil
}

func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return false, err
	}
	ObserveRedisLatency("setNX", time.Since(begin).Seconds())
	return result, nil
}

func HGet(ctx context.Context, key, field string) (string, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.HGet(ctx, key, field).Result()
	if err != nil {
		return "", err
	}
	ObserveRedisLatency("hget", time.Since(begin).Seconds())
	return result, nil
}

func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	ObserveRedisLatency("hgetAll", time.Since(begin).Seconds())
	return result, nil
}

func HSet(ctx context.Context, key, field string, value interface{}) error {
	begin := time.Now()
	err := defaultRedisClient.c.HSet(ctx, key, field, value).Err()
	if err != nil {
		return err
	}
	ObserveRedisLatency("hset", time.Since(begin).Seconds())
	return nil
}

func HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.HMGet(ctx, key, fields...).Result()
	if err != nil {
		return nil, err
	}
	ObserveRedisLatency("hmget", time.Since(begin).Seconds())
	return result, nil
}

func HDel(ctx context.Context, key string, fields ...string) error {
	begin := time.Now()
	err := defaultRedisClient.c.HDel(ctx, key, fields...).Err()
	if err != nil {
		return err
	}
	ObserveRedisLatency("hdel", time.Since(begin).Seconds())
	return nil
}

func HIncrBy(ctx context.Context, key, field string, value int64) (int64, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.HIncrBy(ctx, key, field, value).Result()
	if err != nil {
		return 0, err
	}
	ObserveRedisLatency("hincrBy", time.Since(begin).Seconds())
	return result, nil
}

func LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	begin := time.Now()
	result, err := defaultRedisClient.c.LPush(ctx, key, values...).Result()
	if err != nil {
		return 0, err
	}
	ObserveRedisLatency("lpush", time.Since(begin).Seconds())
	return result, nil
}

func RunScript(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	result, err := defaultRedisClient.c.Eval(ctx, script, keys, args...).Result()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func ScriptLoad(ctx context.Context, script string) (string, error) {
	return defaultRedisClient.c.ScriptLoad(ctx, script).Result()
}

func ScriptFlush(ctx context.Context) error {
	return defaultRedisClient.c.ScriptFlush(ctx).Err()
}

func ScriptExists(ctx context.Context, sha string) (bool, error) {
	exists, err := defaultRedisClient.c.ScriptExists(ctx, sha).Result()
	if err != nil {
		return false, err
	}
	return exists[0], nil
}

func RunScriptBySha(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error) {
	result, err := defaultRedisClient.c.EvalSha(ctx, sha, keys, args...).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func Subscribe(ctx context.Context, key string) *redis.PubSub {
	return defaultRedisClient.c.Subscribe(ctx, key)
}

func Publish(ctx context.Context, key string, val string) (int64, error) {
	return defaultRedisClient.c.Publish(ctx, key, val).Result()
}

func XGroupCreate(ctx context.Context, stream string, group string, start string) error {
	return defaultRedisClient.c.XGroupCreate(ctx, stream, group, start).Err()
}

func XGroupCreateMkStream(ctx context.Context, stream string, group string, start string) error {
	return defaultRedisClient.c.XGroupCreateMkStream(ctx, stream, group, start).Err()
}

func XGroupSetID(ctx context.Context, stream string, group string, start string) error {
	return defaultRedisClient.c.XGroupSetID(ctx, stream, group, start).Err()
}

func XGroupDestroy(ctx context.Context, stream string, group string) error {
	return defaultRedisClient.c.XGroupDestroy(ctx, stream, group).Err()
}

func XGroupCreateConsumer(ctx context.Context, stream string, group string, consumer string) error {
	return defaultRedisClient.c.XGroupCreateConsumer(ctx, stream, group, consumer).Err()
}

func XAdd(ctx context.Context, stream string, values interface{}) (string, error) {
	return defaultRedisClient.c.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Approx: true,
		Values: values,
	}).Result()
}

func XReadGroup(ctx context.Context, group string, consumer string, stream string, count int64, block time.Duration) ([]redis.XMessage, error) {
	result, err := defaultRedisClient.c.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result[0].Messages, nil
	}
	return nil, nil
}

func XAck(ctx context.Context, stream string, group string, ids ...string) error {
	return defaultRedisClient.c.XAck(ctx, stream, group, ids...).Err()
}

func XDel(ctx context.Context, stream string, ids ...string) error {
	return defaultRedisClient.c.XDel(ctx, stream, ids...).Err()
}

func ObserveRedisLatency(command string, latency float64) {
}
