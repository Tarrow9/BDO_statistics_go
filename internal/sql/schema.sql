-- internal/sql/schema.sql
/*
 *
 */

SET lock_timeout = '5s';
SET statement_timeout = '30s';
SET TIME ZONE 'Asia/Seoul';


CREATE TABLE IF NOT EXISTS items (
  item_id           TEXT PRIMARY KEY,
  item_attrs        jsonb,
  name              TEXT        NOT NULL,
  stock_count       int,
  buy_bid_price     int,
  sell_bid_price    int,
  last_trade_price  int,
  total_trade_count int,
  total_buy_bid     int,
  total_sell_bid    int,
  PRIMARY KEY item_id
);

CREATE TABLE public.item_ts (
  item_id       int         NOT NULL,
  time          timestamptz NOT NULL,
  name          text,
  trading_vol   int,
  trading_price int,
  PRIMARY KEY (item_id, time)
) PARTITION BY RANGE (time);

---------------------
--월 파티션 보장 함수
CREATE OR REPLACE FUNCTION public.ensure_month_partition(base_table regclass, month_start date)
RETURNS void
LANGUAGE plpgsql
AS $fn$
DECLARE
  sch   text := split_part(base_table::text, '.', 1);
  base  text := split_part(base_table::text, '.', 2);

  start_ts timestamptz := make_timestamptz(EXTRACT(YEAR FROM month_start)::int,
                                           EXTRACT(MONTH FROM month_start)::int,
                                           1, 0, 0, 0, 'Asia/Seoul');
  end_ts   timestamptz := (start_ts + interval '1 month');

  child_name text := base || '_' || to_char(start_ts, 'YYYY_MM');  -- ex) item_ts_2025_08
  child_qual text := format('%I.%I', sch, child_name);             -- ex) public.item_ts_2025_08
BEGIN
  IF to_regclass(child_qual) IS NULL THEN
    EXECUTE format(
      'CREATE TABLE %I.%I PARTITION OF %I.%I FOR VALUES FROM (%L) TO (%L)',
      sch, child_name,   -- 새 파티션 이름
      sch, base,         -- 부모 테이블
      start_ts, end_ts
    );
  END IF;
END
$fn$;

-- 오래된 파티션 드롭 함수
CREATE OR REPLACE FUNCTION public.drop_partitions_older_than_by_name(base_table regclass, keep_interval interval)
RETURNS void
LANGUAGE plpgsql
AS $fn$
DECLARE
  r record;
  sch   text := split_part(base_table::text, '.', 1);
  base  text := split_part(base_table::text, '.', 2);
  cutoff timestamptz := date_trunc('day', now() AT TIME ZONE 'Asia/Seoul') - keep_interval;
  part_month date;
BEGIN
  FOR r IN
    SELECT c.relname
    FROM pg_class c
    JOIN pg_inherits i ON i.inhrelid = c.oid
    WHERE i.inhparent = base_table
      AND c.relispartition
      -- (선택) 이름 규칙 검증: base_YYYY_MM 형태만 대상으로
      AND c.relname LIKE base || '\_%' ESCAPE '\'
  LOOP
    -- relname 끝 7글자 'YYYY_MM' → 월 시작일
    part_month := to_date(right(r.relname, 7), 'YYYY_MM');

    -- 상한(= part_month + 1개월)이 컷오프 이하이면 드롭
    IF (part_month + INTERVAL '1 month') <= cutoff THEN
      EXECUTE format('DROP TABLE IF EXISTS %I.%I', sch, r.relname);
    END IF;
  END LOOP;
END
$fn$;

-- 운영 DO 블록
DO $$
DECLARE
  this_month date := date_trunc('month', (now() AT TIME ZONE 'Asia/Seoul'))::date;
BEGIN
  -- 전달/현재달/다음달 파티션 확보 (KST 기준)
  PERFORM public.ensure_month_partition('public.item_ts', this_month - interval '1 month');
  PERFORM public.ensure_month_partition('public.item_ts', this_month);
  PERFORM public.ensure_month_partition('public.item_ts', this_month + interval '1 month');

  -- 1개월 보관: KST 기준으로 컷오프 이전 파티션 드롭
  PERFORM public.drop_partitions_older_than_by_name('public.item_ts', interval '1 month');
END$$;