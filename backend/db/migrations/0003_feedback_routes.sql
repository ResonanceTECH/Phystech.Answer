-- Обратная связь по ответам и интерактивные маршруты (спека §16, §24).

CREATE TABLE IF NOT EXISTS answer_feedback (
    id         text PRIMARY KEY,
    answer_id  text NOT NULL REFERENCES answers(id) ON DELETE CASCADE,
    user_id    text REFERENCES users(id) ON DELETE SET NULL,
    rating     text NOT NULL, -- positive|negative
    comment    text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_feedback_answer ON answer_feedback(answer_id);

CREATE TABLE IF NOT EXISTS action_routes (
    id         text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    answer_id  text NOT NULL REFERENCES answers(id) ON DELETE CASCADE,
    title      text NOT NULL DEFAULT '',
    stale      boolean NOT NULL DEFAULT false, -- источник изменился
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_routes_user ON action_routes(user_id);

CREATE TABLE IF NOT EXISTS action_route_steps (
    id         text PRIMARY KEY,
    route_id   text NOT NULL REFERENCES action_routes(id) ON DELETE CASCADE,
    step_id    text REFERENCES answer_steps(id) ON DELETE SET NULL,
    step_order int NOT NULL DEFAULT 0,
    title      text NOT NULL DEFAULT '',
    done       boolean NOT NULL DEFAULT false,
    done_at    timestamptz
);
CREATE INDEX IF NOT EXISTS idx_route_steps ON action_route_steps(route_id);
