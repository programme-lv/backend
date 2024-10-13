ALTER TABLE runtime_data ALTER COLUMN exit_code TYPE bigint USING exit_code::bigint;
ALTER TABLE runtime_data ALTER COLUMN cpu_time_millis TYPE bigint USING cpu_time_millis::bigint;
ALTER TABLE runtime_data ALTER COLUMN wall_time_millis TYPE bigint USING wall_time_millis::bigint;
ALTER TABLE runtime_data ALTER COLUMN memory_kibi_bytes TYPE bigint USING memory_kibi_bytes::bigint;
