\connect movements

CREATE TABLE movements (
    id           text PRIMARY KEY,
    warehouse_id text NOT NULL,
    code         text NOT NULL,
    label        text NOT NULL
);

INSERT INTO movements (id, warehouse_id, code, label)
SELECT 'MV-' || lpad(n::text, 6, '0'),
       'WH-1',
       'C-' || lpad((n % 1000)::text, 3, '0'),
       'Mouvement ' || n
           || CASE WHEN n % 7 = 0 THEN ' "urgent"' ELSE '' END
           || CASE WHEN n % 11 = 0 THEN ' quai C:\nord' ELSE '' END
FROM generate_series(1, 50000) AS n;
