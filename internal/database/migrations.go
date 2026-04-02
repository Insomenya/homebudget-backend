package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

func Migrate(db *sql.DB) error {
	for i, s := range ddl {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("ddl #%d: %w", i, err)
		}
	}
	// Legacy column cleanup — add columns if missing (idempotent)
	addCol(db, "planned_transactions", "original_day", "INTEGER NOT NULL DEFAULT 0")
	addCol(db, "planned_transactions", "notify_days_before", "INTEGER NOT NULL DEFAULT 3")
	addCol(db, "planned_transactions", "overdue_days_limit", "INTEGER NOT NULL DEFAULT 30")
	addCol(db, "planned_transactions", "loan_id", "INTEGER REFERENCES loans(id) ON DELETE SET NULL")
	addCol(db, "transactions", "reminder_id", "INTEGER REFERENCES planned_reminders(id) ON DELETE SET NULL")
	addCol(db, "loans", "end_date", "TEXT NOT NULL DEFAULT ''")
	addCol(db, "loans", "default_account_id", "INTEGER REFERENCES accounts(id) ON DELETE SET NULL")
	addCol(db, "loans", "loan_account_id", "INTEGER REFERENCES accounts(id) ON DELETE SET NULL")
	addCol(db, "loans", "planned_id", "INTEGER REFERENCES planned_transactions(id) ON DELETE SET NULL")
	addCol(db, "loans", "accounting_start_date", "TEXT NOT NULL DEFAULT ''")
	addCol(db, "loans", "initial_accrued_interest", "REAL NOT NULL DEFAULT 0")
	addCol(db, "loans", "loan_category_id", "INTEGER REFERENCES categories(id) ON DELETE SET NULL")
	addCol(db, "loans", "remaining_debt", "REAL NOT NULL DEFAULT 0")
	// Init existing loans: remaining_debt = principal minus pre-accounting payments
	db.Exec("UPDATE loans SET remaining_debt = principal - already_paid WHERE remaining_debt = 0")

	// Drop old budget tables
	for _, t := range []string{"budget_cells", "budget_rows", "budget_columns"} {
		db.Exec("DROP TABLE IF EXISTS " + t)
	}

	// Remove legacy columns from transactions if they exist (SQLite can't DROP COLUMN easily, ignore)
	// is_pending and planned_id are legacy — we keep them but ignore on new code

	if err := seed(db); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	log.Println("✅ Migrated")
	return nil
}

var ddl = []string{
	`CREATE TABLE IF NOT EXISTS members (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL,
		icon        TEXT    NOT NULL DEFAULT '',
		is_archived INTEGER NOT NULL DEFAULT 0,
		created_at  TEXT    NOT NULL,
		updated_at  TEXT    NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS lookup_values (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		group_name TEXT NOT NULL,
		value      TEXT NOT NULL,
		label      TEXT NOT NULL,
		sort_order INTEGER NOT NULL DEFAULT 0,
		is_active  INTEGER NOT NULL DEFAULT 1,
		UNIQUE(group_name, value)
	)`,

	`CREATE TABLE IF NOT EXISTS accounts (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		name            TEXT    NOT NULL,
		type            TEXT    NOT NULL DEFAULT 'cash',
		currency        TEXT    NOT NULL DEFAULT 'RUB',
		initial_balance REAL    NOT NULL DEFAULT 0,
		member_id       INTEGER NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
		is_archived     INTEGER NOT NULL DEFAULT 0,
		is_hidden       INTEGER NOT NULL DEFAULT 0,
		created_at      TEXT    NOT NULL,
		updated_at      TEXT    NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS categories (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL,
		type        TEXT    NOT NULL DEFAULT 'expense',
		icon        TEXT    NOT NULL DEFAULT '',
		parent_id   INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		sort_order  INTEGER NOT NULL DEFAULT 0,
		is_archived INTEGER NOT NULL DEFAULT 0,
		created_at  TEXT    NOT NULL,
		updated_at  TEXT    NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS shared_groups (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL,
		icon        TEXT    NOT NULL DEFAULT '',
		is_archived INTEGER NOT NULL DEFAULT 0,
		created_at  TEXT    NOT NULL,
		updated_at  TEXT    NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS shared_group_members (
		id                INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id          INTEGER NOT NULL REFERENCES shared_groups(id) ON DELETE CASCADE,
		member_id         INTEGER NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
		share_numerator   INTEGER NOT NULL DEFAULT 1,
		share_denominator INTEGER NOT NULL DEFAULT 1,
		UNIQUE(group_id, member_id)
	)`,

	`CREATE TABLE IF NOT EXISTS loans (
		id                INTEGER PRIMARY KEY AUTOINCREMENT,
		name              TEXT    NOT NULL,
		principal         REAL    NOT NULL CHECK(principal > 0),
		annual_rate       REAL    NOT NULL CHECK(annual_rate >= 0),
		start_date        TEXT    NOT NULL,
		end_date          TEXT    NOT NULL,
		monthly_payment   REAL    NOT NULL,
		already_paid      REAL    NOT NULL DEFAULT 0,
		remaining_debt    REAL    NOT NULL DEFAULT 0,
		account_id        INTEGER REFERENCES accounts(id) ON DELETE SET NULL,
		default_account_id INTEGER REFERENCES accounts(id) ON DELETE SET NULL,
		loan_account_id   INTEGER REFERENCES accounts(id) ON DELETE SET NULL,
		category_id       INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		planned_id        INTEGER REFERENCES planned_transactions(id) ON DELETE SET NULL,
		accounting_start_date TEXT NOT NULL DEFAULT '',
		initial_accrued_interest REAL NOT NULL DEFAULT 0,
		is_active         INTEGER NOT NULL DEFAULT 1,
		created_at        TEXT    NOT NULL,
		updated_at        TEXT    NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS planned_transactions (
		id                 INTEGER PRIMARY KEY AUTOINCREMENT,
		name               TEXT    NOT NULL,
		amount             REAL    NOT NULL CHECK(amount > 0),
		type               TEXT    NOT NULL DEFAULT 'expense',
		category_id        INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		shared_group_id    INTEGER REFERENCES shared_groups(id) ON DELETE SET NULL,
		paid_by_member_id  INTEGER REFERENCES members(id) ON DELETE SET NULL,
		loan_id            INTEGER REFERENCES loans(id) ON DELETE SET NULL,
		recurrence         TEXT    NOT NULL DEFAULT 'monthly',
		start_date         TEXT    NOT NULL,
		end_date           TEXT,
		next_due           TEXT    NOT NULL,
		original_day       INTEGER NOT NULL DEFAULT 0,
		notify_days_before INTEGER NOT NULL DEFAULT 3,
		overdue_days_limit INTEGER NOT NULL DEFAULT 30,
		is_active          INTEGER NOT NULL DEFAULT 1,
		created_at         TEXT    NOT NULL,
		updated_at         TEXT    NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS planned_reminders (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		planned_id      INTEGER NOT NULL REFERENCES planned_transactions(id) ON DELETE CASCADE,
		due_date        TEXT    NOT NULL,
		amount          REAL    NOT NULL,
		transaction_id  INTEGER REFERENCES transactions(id) ON DELETE SET NULL,
		prev_next_due   TEXT    NOT NULL DEFAULT '',
		is_executed     INTEGER NOT NULL DEFAULT 0,
		created_at      TEXT    NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS transactions (
		id                INTEGER PRIMARY KEY AUTOINCREMENT,
		date              TEXT    NOT NULL,
		amount            REAL    NOT NULL CHECK(amount > 0),
		description       TEXT    NOT NULL DEFAULT '',
		type              TEXT    NOT NULL DEFAULT 'expense',
		account_id        INTEGER REFERENCES accounts(id) ON DELETE RESTRICT,
		to_account_id     INTEGER REFERENCES accounts(id) ON DELETE RESTRICT,
		category_id       INTEGER REFERENCES categories(id) ON DELETE RESTRICT,
		shared_group_id   INTEGER REFERENCES shared_groups(id) ON DELETE RESTRICT,
		paid_by_member_id INTEGER REFERENCES members(id) ON DELETE RESTRICT,
		loan_id           INTEGER REFERENCES loans(id) ON DELETE SET NULL,
		reminder_id       INTEGER REFERENCES planned_reminders(id) ON DELETE SET NULL,
		created_at        TEXT    NOT NULL,
		updated_at        TEXT    NOT NULL
	)`,

	// ── Indexes ─────────────────────────────────────
	`CREATE INDEX IF NOT EXISTS idx_tx_date       ON transactions(date)`,
	`CREATE INDEX IF NOT EXISTS idx_tx_type       ON transactions(type)`,
	`CREATE INDEX IF NOT EXISTS idx_tx_account    ON transactions(account_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tx_category   ON transactions(category_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tx_group      ON transactions(shared_group_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tx_paid_by    ON transactions(paid_by_member_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tx_loan       ON transactions(loan_id)`,
	`CREATE INDEX IF NOT EXISTS idx_tx_reminder   ON transactions(reminder_id)`,
	`CREATE INDEX IF NOT EXISTS idx_cat_parent    ON categories(parent_id)`,
	`CREATE INDEX IF NOT EXISTS idx_acc_member    ON accounts(member_id)`,
	`CREATE INDEX IF NOT EXISTS idx_sgm_group     ON shared_group_members(group_id)`,
	`CREATE INDEX IF NOT EXISTS idx_pt_next_due   ON planned_transactions(next_due)`,
	`CREATE INDEX IF NOT EXISTS idx_pt_active     ON planned_transactions(is_active)`,
	`CREATE INDEX IF NOT EXISTS idx_pt_loan       ON planned_transactions(loan_id)`,
	`CREATE INDEX IF NOT EXISTS idx_pr_planned    ON planned_reminders(planned_id)`,
	`CREATE INDEX IF NOT EXISTS idx_pr_executed   ON planned_reminders(is_executed)`,
	`CREATE INDEX IF NOT EXISTS idx_lookup_group  ON lookup_values(group_name, sort_order)`,
	`CREATE INDEX IF NOT EXISTS idx_loan_active   ON loans(is_active)`,
	`CREATE INDEX IF NOT EXISTS idx_acc_hidden    ON accounts(is_hidden)`,
}

func seed(db *sql.DB) error {
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM members").Scan(&n)
	if n > 0 {
		return nil
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	members := []struct{ name, icon string }{{"Семья", "👨‍👩"}, {"Ксюша", "👩"}}
	for _, m := range members {
		db.Exec(`INSERT INTO members (name,icon,is_archived,created_at,updated_at) VALUES (?,?,0,?,?)`, m.name, m.icon, now, now)
	}

	accounts := []struct{ name, typ string }{{"Наличные", "cash"}, {"Карта", "bank_card"}}
	for _, a := range accounts {
		db.Exec(`INSERT INTO accounts (name,type,currency,initial_balance,member_id,is_archived,is_hidden,created_at,updated_at) VALUES (?,?,'RUB',0,1,0,0,?,?)`, a.name, a.typ, now, now)
	}

	cats := []struct{ name, typ, icon string; order int }{
		{"Продукты", "expense", "🛒", 10}, {"Коммуналка", "expense", "🏠", 20},
		{"Связь и интернет", "expense", "🌐", 30}, {"Транспорт", "expense", "🚌", 40},
		{"Здоровье", "expense", "💊", 50}, {"Развлечения", "expense", "🎮", 60},
		{"Одежда", "expense", "👕", 70}, {"Ипотека", "expense", "🏦", 75},
		{"Прочие расходы", "expense", "📦", 100},
		{"Зарплата", "income", "💰", 10}, {"Подработка", "income", "💼", 20},
		{"Прочие доходы", "income", "📥", 100},
	}
	for _, c := range cats {
		db.Exec(`INSERT INTO categories (name,type,icon,parent_id,sort_order,is_archived,created_at,updated_at) VALUES (?,?,?,NULL,?,0,?,?)`, c.name, c.typ, c.icon, c.order, now, now)
	}

	db.Exec(`INSERT INTO shared_groups (name,icon,is_archived,created_at,updated_at) VALUES ('Квартира','🏡',0,?,?)`, now, now)
	db.Exec(`INSERT INTO shared_group_members (group_id,member_id,share_numerator,share_denominator) VALUES (1,1,2,3)`)
	db.Exec(`INSERT INTO shared_group_members (group_id,member_id,share_numerator,share_denominator) VALUES (1,2,1,3)`)

	lookups := []struct{ group, value, label string; order int }{
		{"account_type", "cash", "Наличные", 10},
		{"account_type", "bank_card", "Банковская карта", 20},
		{"account_type", "savings", "Накопления", 30},
		{"account_type", "deposit", "Вклад", 40},
		{"account_type", "credit", "Кредитный счёт", 50},
		{"account_type", "ewallet", "Электронный кошелёк", 60},
		{"currency", "RUB", "₽ Рубль", 10},
		{"currency", "USD", "$ Доллар", 20},
		{"currency", "EUR", "€ Евро", 30},
		{"transaction_type", "expense", "Расход", 10},
		{"transaction_type", "income", "Доход", 20},
		{"transaction_type", "transfer", "Перевод", 30},
		{"category_type", "expense", "Расход", 10},
		{"category_type", "income", "Доход", 20},
		{"recurrence", "once", "Разово", 10},
		{"recurrence", "weekly", "Еженедельно", 20},
		{"recurrence", "biweekly", "Раз в 2 недели", 30},
		{"recurrence", "monthly", "Ежемесячно", 40},
		{"recurrence", "quarterly", "Ежеквартально", 50},
		{"recurrence", "yearly", "Ежегодно", 60},
	}
	for _, l := range lookups {
		db.Exec(`INSERT OR IGNORE INTO lookup_values (group_name,value,label,sort_order) VALUES (?,?,?,?)`, l.group, l.value, l.label, l.order)
	}

	log.Println("🌱 Seeded")
	return nil
}

func addCol(db *sql.DB, table, col, def string) {
	q := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, def)
	db.Exec(q) // ignore error = column already exists
}