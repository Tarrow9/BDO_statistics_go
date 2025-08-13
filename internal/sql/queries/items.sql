



-- dashboard
-- 6hours
SELECT time, value1, value2
FROM item_ts
WHERE item_id = $1 AND time >= now() - interval '6 hours'
ORDER BY time;

-- 1day
SELECT
  date_trunc('minute', time)
    - make_interval(mins => (extract(minute from time)::int % 10)) AS bucket,
  avg(value1) AS avg_v1,
  avg(value2) AS avg_v2
FROM item_ts
WHERE item_id = $1
  AND time >= now() - interval '1 day'   -- 필요에 따라 7 days/30 days 등으로 변경
GROUP BY bucket
ORDER BY bucket;

-- 7day
SELECT
  date_trunc('hour', time) AS bucket,
  avg(value1) AS avg_v1,
  avg(value2) AS avg_v2
FROM item_ts
WHERE item_id = $1
  AND time >= now() - interval '7 days'
GROUP BY bucket
ORDER BY bucket;

-- 1month
SELECT
  date_trunc('hour', time)
    - make_interval(hours => (extract(hour from time)::int % 4)) AS bucket,
  avg(value1) AS avg_v1,
  avg(value2) AS avg_v2
FROM item_ts
WHERE item_id = $1
  AND time >= now() - interval '30 days'
GROUP BY bucket
ORDER BY bucket;