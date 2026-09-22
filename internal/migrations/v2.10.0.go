package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_10_0 adds tasks: projects, configurable statuses, tasks with subtasks,
// comments and an activity history.
func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'task_priority') THEN
				CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'urgent');
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'task_status_category') THEN
				CREATE TYPE task_status_category AS ENUM ('todo', 'in_progress', 'done');
			END IF;
		END$$;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS task_projects (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			color TEXT NOT NULL DEFAULT '',
			archived_at TIMESTAMPTZ NULL,
			created_by BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
			CONSTRAINT constraint_task_projects_on_name CHECK (length(name) <= 140),
			CONSTRAINT constraint_task_projects_on_description CHECK (length(description) <= 2000),
			CONSTRAINT constraint_task_projects_on_color CHECK (length(color) <= 20)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS task_statuses (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL UNIQUE,
			category task_status_category NOT NULL DEFAULT 'todo',
			color TEXT NOT NULL DEFAULT '',
			position INT NOT NULL DEFAULT 0,
			is_default BOOLEAN NOT NULL DEFAULT false,
			CONSTRAINT constraint_task_statuses_on_name CHECK (length(name) <= 140),
			CONSTRAINT constraint_task_statuses_on_color CHECK (length(color) <= 20)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			"uuid" UUID DEFAULT gen_random_uuid() NOT NULL UNIQUE,
			project_id INT REFERENCES task_projects(id) ON DELETE SET NULL ON UPDATE CASCADE,
			parent_task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE ON UPDATE CASCADE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status_id INT REFERENCES task_statuses(id) ON DELETE RESTRICT ON UPDATE CASCADE NOT NULL,
			priority task_priority NOT NULL DEFAULT 'medium',
			assigned_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE SET NULL ON UPDATE CASCADE,
			due_date DATE NULL,
			completed_at TIMESTAMPTZ NULL,
			position DOUBLE PRECISION NOT NULL DEFAULT 0,
			created_by BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
			CONSTRAINT constraint_tasks_on_title CHECK (length(title) <= 500),
			CONSTRAINT constraint_tasks_on_description CHECK (length(description) <= 50000)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_tasks_on_assigned_user_id ON tasks (assigned_user_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_tasks_on_project_id ON tasks (project_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_tasks_on_status_id ON tasks (status_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_tasks_on_parent_task_id ON tasks (parent_task_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_tasks_on_conversation_id ON tasks (conversation_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_tasks_on_due_date ON tasks (due_date);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS task_comments (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			user_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
			content TEXT NOT NULL,
			CONSTRAINT constraint_task_comments_on_content CHECK (length(content) <= 10000)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_task_comments_on_task_id ON task_comments (task_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS task_activities (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
			activity_type TEXT NOT NULL,
			meta JSONB DEFAULT '{}'::jsonb NOT NULL
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_task_activities_on_task_id ON task_activities (task_id);
	`); err != nil {
		return err
	}

	// Seed the default statuses on the first run only.
	if _, err := db.Exec(`
		INSERT INTO task_statuses (name, category, position, is_default)
		SELECT * FROM (VALUES
			('To do', 'todo'::task_status_category, 1, true),
			('In progress', 'in_progress'::task_status_category, 2, false),
			('Done', 'done'::task_status_category, 3, false)
		) AS v(name, category, position, is_default)
		WHERE NOT EXISTS (SELECT 1 FROM task_statuses);
	`); err != nil {
		return err
	}

	// Agents get to read, write and delete tasks, admins also manage projects and statuses.
	for _, p := range []string{"tasks:read", "tasks:write", "tasks:delete"} {
		if _, err := db.Exec(`
			UPDATE roles
			SET permissions = array_append(permissions, $1)
			WHERE name IN ('Agent', 'Admin')
			AND NOT ($1 = ANY(permissions));
		`, p); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'tasks:manage')
		WHERE name = 'Admin'
		AND NOT ('tasks:manage' = ANY(permissions));
	`); err != nil {
		return err
	}

	return nil
}
