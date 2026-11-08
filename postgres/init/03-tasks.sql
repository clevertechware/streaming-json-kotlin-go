\connect tasks

CREATE TABLE task_plan_movements (
    task_plan_id text NOT NULL,
    movement_id  text NOT NULL,
    code         text NOT NULL,
    label        text NOT NULL,
    PRIMARY KEY (task_plan_id, movement_id)
);
