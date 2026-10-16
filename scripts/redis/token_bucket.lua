-- KEYS[1]=bucket  ARGV: rate(tokens/s), burst, now_ms, cost
local d = redis.call('HMGET', KEYS[1], 't', 'ts')
local t = tonumber(d[1]) or tonumber(ARGV[2]); local ts = tonumber(d[2]) or tonumber(ARGV[3])
t = math.min(tonumber(ARGV[2]), t + (tonumber(ARGV[3]) - ts) / 1000 * tonumber(ARGV[1]))
local ok = 0
if t >= tonumber(ARGV[4]) then t = t - tonumber(ARGV[4]); ok = 1 end
redis.call('HSET', KEYS[1], 't', t, 'ts', ARGV[3]); redis.call('PEXPIRE', KEYS[1], 60000)
return {ok, math.floor(t)}
