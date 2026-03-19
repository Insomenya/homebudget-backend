package main

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"homebudget/internal/database"
)

func main() {
	oldPath := "shared_expenses.db"
	newPath := "homebudget.db"
	if len(os.Args) > 1 {
		oldPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		newPath = os.Args[2]
	}

	// ── Новая БД ────────────────────────────────────────────
	newDB, err := database.Open(newPath)
	if err != nil {
		log.Fatalf("new db: %v", err)
	}
	defer newDB.Close()

	if err := database.Migrate(newDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Защита от повторного импорта
	var existing int
	newDB.QueryRow("SELECT COUNT(*) FROM transactions WHERE shared_group_id IS NOT NULL").Scan(&existing)
	if existing > 0 {
		log.Printf("⚠️  %s already has %d shared transactions — skipping", newPath, existing)
		return
	}

	// ── Старая БД ───────────────────────────────────────────
	oldDB, err := sql.Open("sqlite", oldPath)
	if err != nil {
		log.Fatalf("old db: %v", err)
	}
	defer oldDB.Close()

	var hasTbl int
	oldDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='expenses'").Scan(&hasTbl)
	if hasTbl == 0 {
		log.Fatal("❌ Old DB has no 'expenses' table")
	}

	// ── Маппинг участников ──────────────────────────────────
	var familyID, ksyushaID int64
	newDB.QueryRow("SELECT id FROM members WHERE name='Семья'").Scan(&familyID)
	newDB.QueryRow("SELECT id FROM members WHERE name='Ксюша'").Scan(&ksyushaID)
	if familyID == 0 || ksyushaID == 0 {
		log.Fatal("❌ Members 'Семья'/'Ксюша' not found — run server first to seed")
	}

	// ── Маппинг категорий ───────────────────────────────────
	catMap := make(map[string]int64)
	catRows, _ := newDB.Query("SELECT id, name FROM categories")
	for catRows.Next() {
		var id int64
		var name string
		catRows.Scan(&id, &name)
		catMap[name] = id
	}
	catRows.Close()

	defaultCat := catMap["Прочие расходы"]
	if defaultCat == 0 {
		log.Fatal("❌ Category 'Прочие расходы' not found")
	}

	var firstAccID int64
	newDB.QueryRow("SELECT id FROM accounts WHERE member_id=? ORDER BY id LIMIT 1", familyID).Scan(&firstAccID)

	keywords := map[string]string{
		"продукт": "Продукты", "еда": "Продукты", "магазин": "Продукты",
		"коммунал": "Коммуналка", "жкх": "Коммуналка", "квартплат": "Коммуналка",
		"интернет": "Связь и интернет", "связь": "Связь и интернет", "телефон": "Связь и интернет",
		"такси": "Транспорт", "метро": "Транспорт", "автобус": "Транспорт", "бензин": "Транспорт",
		"аптек": "Здоровье", "врач": "Здоровье", "лекарств": "Здоровье",
		"кино": "Развлечения", "ресторан": "Развлечения", "кафе": "Развлечения",
		"одежд": "Одежда",
	}

	// ── Импорт ──────────────────────────────────────────────
	rows, err := oldDB.Query("SELECT date, amount, description, payer FROM expenses ORDER BY date")
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	var imported int

	for rows.Next() {
		var date string
		var amount float64
		var desc, payer string
		if err := rows.Scan(&date, &amount, &desc, &payer); err != nil {
			log.Printf("  skip: %v", err)
			continue
		}

		var memberID int64
		var accountID *int64
		if payer == "we" {
			memberID = familyID
			if firstAccID > 0 {
				accountID = &firstAccID
			}
		} else {
			memberID = ksyushaID
		}

		catID := defaultCat
		low := strings.ToLower(desc)
		for kw, catName := range keywords {
			if strings.Contains(low, kw) {
				if id, ok := catMap[catName]; ok {
					catID = id
					break
				}
			}
		}

		_, err := newDB.Exec(
			`INSERT INTO transactions
			 (date,amount,description,type,account_id,to_account_id,
			  category_id,shared_group_id,paid_by_member_id,created_at,updated_at)
			 VALUES (?,?,?,'expense',?,NULL,?,1,?,?,?)`,
			date, amount, desc, accountID, catID, memberID, now, now)
		if err != nil {
			log.Printf("  insert: %v", err)
			continue
		}
		imported++
	}

	log.Printf("✅ Imported %d transactions: %s → %s", imported, oldPath, newPath)
}