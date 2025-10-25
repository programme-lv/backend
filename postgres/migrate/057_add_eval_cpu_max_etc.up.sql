-- Add columns to track resource usage during evaluation
ALTER TABLE evaluations
    ADD COLUMN cpu_max_ms INTEGER,  -- Maximum CPU time used in milliseconds
    ADD COLUMN mem_max_kib INTEGER, -- Maximum memory used in kibibytes
    ADD COLUMN exceeded_cpu BOOLEAN,-- Whether CPU limit was exceeded
    ADD COLUMN exceeded_mem BOOLEAN; -- Whether memory limit was exceeded
