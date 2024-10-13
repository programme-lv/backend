alter table runtime_data drop column test_id;

-- make columns stdout, stderr, exit_code, cpu_time_millis, wall_time_millis, memory_kibi_bytes not null
alter table runtime_data alter column stdout set not null;
alter table runtime_data alter column stderr set not null;
alter table runtime_data alter column exit_code set not null;
alter table runtime_data alter column cpu_time_millis set not null;
alter table runtime_data alter column wall_time_millis set not null;
alter table runtime_data alter column memory_kibi_bytes set not null;
