ALTER TABLE
    evaluations DROP CONSTRAINT evaluations_evaluation_stage_check,
ADD
    CONSTRAINT evaluations_evaluation_stage_check CHECK (
        evaluation_stage :: text = ANY (
            ARRAY ['waiting'::character varying, 'received'::character varying, 'compiling'::character varying, 'testing'::character varying, 'finished'::character varying, 'internal_error'::character varying, 'compile_error'::character varying] :: text []
        )
    );