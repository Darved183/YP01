package database

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

func ensurePassword(db *sqlx.DB) error {
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'password'`); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := db.Exec(`ALTER TABLE users ADD COLUMN password TEXT NOT NULL DEFAULT ''`)
	return err
}

func SeedIfEmpty(db *sqlx.DB) error {
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM products`); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = insertDemo(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureStock(db *sqlx.DB) error {
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM pragma_table_info('products') WHERE name = 'stock'`); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if _, err := db.Exec(`ALTER TABLE products ADD COLUMN stock INTEGER NOT NULL DEFAULT 20`); err != nil {
		return err
	}
	stocks := []int{18, 12, 7, 15, 9, 6, 11, 4, 8, 14, 20, 5, 10, 3}
	for i, stock := range stocks {
		if _, err := db.Exec(`UPDATE products SET stock = ? WHERE id = ?`, stock, nid("c0000000-0000-4000-8000", i+1)); err != nil {
			return err
		}
	}

	_, err := db.Exec(`
		UPDATE products SET stock = stock - (
			SELECT COUNT(*) FROM order_items oi
			JOIN orders o ON o.id = oi.order_id
			WHERE oi.product_id = products.id AND o.status != 'Отменён'
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE products SET stock = 0 WHERE stock < 0`)
	return err
}

func ensureRoleNames(db *sqlx.DB) error {
	names := []struct{ id, name, old string }{
		{"00000000-0000-0000-0000-000000000001", "Гость", "guest"},
		{"00000000-0000-0000-0000-000000000002", "Клиент", "client"},
		{"00000000-0000-0000-0000-000000000003", "Менеджер", "manager"},
		{"00000000-0000-0000-0000-000000000004", "Администратор", "admin"},
	}
	for _, item := range names {
		if _, err := db.Exec(`UPDATE roles SET name = ? WHERE id = ? AND lower(name) = ?`, item.name, item.id, item.old); err != nil {
			return err
		}
	}
	return nil
}

func ensureSeasons(db *sqlx.DB) error {
	// Весна и осень — это и есть демисезон, отдельно их не держим.
	_, err := db.Exec(`UPDATE products SET season = 'демисезон' WHERE season IN ('весна', 'осень')`)
	return err
}

// Галочки прав лежат в roles.rights строкой. На вход в магазин они не влияют.
func ensureRoleRights(db *sqlx.DB) error {
	ok, err := hasColumn(db, "roles", "rights")
	if err != nil || ok {
		return err
	}
	_, err = db.Exec(`ALTER TABLE roles ADD COLUMN rights TEXT NOT NULL DEFAULT ''`)
	return err
}

func hasColumn(db *sqlx.DB, table, column string) (bool, error) {
	var n int
	err := db.Get(&n, `SELECT COUNT(*) FROM pragma_table_info('`+table+`') WHERE name = ?`, column)
	return n > 0, err
}

// Артикул короткий: его видит покупатель. Длинный id остаётся в базе и на сайт не выводится.
func ensureArticles(db *sqlx.DB) error {
	ok, err := hasColumn(db, "products", "article")
	if err != nil {
		return err
	}
	if !ok {
		if _, err = db.Exec(`ALTER TABLE products ADD COLUMN article TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	var ids []string
	if err = db.Select(&ids, `SELECT id FROM products WHERE article = '' ORDER BY id`); err != nil {
		return err
	}
	var max int
	if err = db.Get(&max, `SELECT COALESCE(MAX(CAST(article AS INTEGER)), 1000) FROM products WHERE article != ''`); err != nil {
		return err
	}
	for _, id := range ids {
		max++
		if _, err = db.Exec(`UPDATE products SET article = ? WHERE id = ?`, strconv.Itoa(max), id); err != nil {
			return err
		}
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_products_article ON products(article)`)
	return err
}

// Номер заказа для человека. В списках и в адресе страницы он, а не id.
func ensureOrderNumbers(db *sqlx.DB) error {
	ok, err := hasColumn(db, "orders", "number")
	if err != nil {
		return err
	}
	if !ok {
		if _, err = db.Exec(`ALTER TABLE orders ADD COLUMN number INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	var ids []string
	if err = db.Select(&ids, `SELECT id FROM orders WHERE number = 0 ORDER BY date_time, id`); err != nil {
		return err
	}
	var max int
	if err = db.Get(&max, `SELECT COALESCE(MAX(number), 0) FROM orders`); err != nil {
		return err
	}
	if max < 1000 {
		max = 1000
	}
	for _, id := range ids {
		max++
		if _, err = db.Exec(`UPDATE orders SET number = ? WHERE id = ?`, max, id); err != nil {
			return err
		}
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_number ON orders(number)`)
	return err
}

// Короткий номер клиента. В заказах его видно вместо длинного id.
func ensureUserNumbers(db *sqlx.DB) error {
	ok, err := hasColumn(db, "users", "number")
	if err != nil {
		return err
	}
	if !ok {
		if _, err = db.Exec(`ALTER TABLE users ADD COLUMN number INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	var ids []string
	if err = db.Select(&ids, `SELECT id FROM users WHERE number = 0 ORDER BY registration_date, id`); err != nil {
		return err
	}
	var max int
	if err = db.Get(&max, `SELECT COALESCE(MAX(number), 0) FROM users`); err != nil {
		return err
	}
	if max < 1000 {
		max = 1000
	}
	for _, id := range ids {
		max++
		if _, err = db.Exec(`UPDATE users SET number = ? WHERE id = ?`, max, id); err != nil {
			return err
		}
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_number ON users(number)`)
	return err
}

func ensureOrderText(db *sqlx.DB) error {
	cols := []struct{ name, ddl string }{
		{"name", `ALTER TABLE order_items ADD COLUMN name TEXT NOT NULL DEFAULT ''`},
		{"description", `ALTER TABLE order_items ADD COLUMN description TEXT NOT NULL DEFAULT ''`},
		{"article", `ALTER TABLE order_items ADD COLUMN article TEXT NOT NULL DEFAULT ''`},
		{"spec", `ALTER TABLE order_items ADD COLUMN spec TEXT NOT NULL DEFAULT ''`},
	}
	for _, col := range cols {
		ok, err := hasColumn(db, "order_items", col.name)
		if err != nil {
			return err
		}
		if !ok {
			if _, err = db.Exec(col.ddl); err != nil {
				return err
			}
		}
	}
	_, err := db.Exec(`
		UPDATE order_items SET
			name = (SELECT name FROM products WHERE products.id = order_items.product_id),
			description = (SELECT description FROM products WHERE products.id = order_items.product_id),
			article = (SELECT article FROM products WHERE products.id = order_items.product_id),
			spec = (SELECT size || ' · ' || color FROM products WHERE products.id = order_items.product_id)
		WHERE name = ''
		  AND EXISTS (SELECT 1 FROM products WHERE products.id = order_items.product_id)
	`)
	return err
}

func sizeRank(name string) int {
	order := []string{"XS", "S", "M", "L", "XL", "XXL"}
	for i, item := range order {
		if item == name {
			return i
		}
	}
	return len(order) + 1
}

func sortSizeNames(names []string) []string {
	sorted := append([]string(nil), names...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sizeRank(sorted[j]) < sizeRank(sorted[i]) || (sizeRank(sorted[j]) == sizeRank(sorted[i]) && sorted[j] < sorted[i]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

// У одного товара несколько размеров, у каждого свой остаток.
// Старый размер сохраняет своё количество, остальные стандартные размеры появляются с двумя штуками.
func ensureProductSizes(db *sqlx.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS product_sizes (
		product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
		size TEXT NOT NULL,
		stock INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (product_id, size)
	)`)
	if err != nil {
		return err
	}
	type row struct {
		ID    string `db:"id"`
		Size  string `db:"size"`
		Stock int    `db:"stock"`
	}
	var products []row
	if err = db.Select(&products, `
		SELECT id, size, stock FROM products
		WHERE NOT EXISTS (SELECT 1 FROM product_sizes ps WHERE ps.product_id = products.id)
	`); err != nil {
		return err
	}
	standard := []string{"XS", "S", "M", "L", "XL"}
	for _, item := range products {
		orig := strings.TrimSpace(item.Size)
		if strings.Contains(orig, ",") {
			orig = strings.TrimSpace(strings.Split(orig, ",")[0])
		}
		if orig == "" {
			orig = "M"
		}
		if _, err = db.Exec(`INSERT INTO product_sizes (product_id, size, stock) VALUES (?, ?, ?)`, item.ID, orig, item.Stock); err != nil {
			return err
		}
		for _, size := range standard {
			if strings.EqualFold(size, orig) {
				continue
			}
			if _, err = db.Exec(`INSERT INTO product_sizes (product_id, size, stock) VALUES (?, ?, ?)`, item.ID, size, 2); err != nil {
				return err
			}
		}
	}

	ok, err := hasColumn(db, "cart", "size")
	if err != nil {
		return err
	}
	if !ok {
		if _, err = db.Exec(`ALTER TABLE cart ADD COLUMN size TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if _, err = db.Exec(`
		UPDATE cart SET size = (SELECT size FROM products WHERE products.id = cart.product_id)
		WHERE size = ''
		  AND EXISTS (
			SELECT 1 FROM products
			WHERE products.id = cart.product_id AND INSTR(products.size, ',') = 0 AND products.size != ''
		  )
	`); err != nil {
		return err
	}

	ok, err = hasColumn(db, "order_items", "size")
	if err != nil {
		return err
	}
	if !ok {
		if _, err = db.Exec(`ALTER TABLE order_items ADD COLUMN size TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if _, err = db.Exec(`
		UPDATE order_items SET size = (SELECT size FROM products WHERE products.id = order_items.product_id)
		WHERE size = ''
		  AND EXISTS (
			SELECT 1 FROM products
			WHERE products.id = order_items.product_id AND INSTR(products.size, ',') = 0 AND products.size != ''
		  )
	`); err != nil {
		return err
	}

	var ids []string
	if err = db.Select(&ids, `SELECT id FROM products`); err != nil {
		return err
	}
	for _, id := range ids {
		var sizes []string
		if err = db.Select(&sizes, `SELECT size FROM product_sizes WHERE product_id = ?`, id); err != nil {
			return err
		}
		if len(sizes) == 0 {
			continue
		}
		var sum int
		if err = db.Get(&sum, `SELECT COALESCE(SUM(stock), 0) FROM product_sizes WHERE product_id = ?`, id); err != nil {
			return err
		}
		if _, err = db.Exec(`UPDATE products SET size = ?, stock = ? WHERE id = ?`, strings.Join(sortSizeNames(sizes), ", "), sum, id); err != nil {
			return err
		}
	}
	return nil
}

func ensureComment(db *sqlx.DB) error {
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM pragma_table_info('orders') WHERE name = 'comment'`); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := db.Exec(`ALTER TABLE orders ADD COLUMN comment TEXT NOT NULL DEFAULT ''`)
	return err
}

func nid(prefix string, n int) string {
	return fmt.Sprintf("%s-%012d", prefix, n)
}

func insertDemo(tx *sqlx.Tx) error {
	const (
		clientRole  = "00000000-0000-0000-0000-000000000002"
		managerRole = "00000000-0000-0000-0000-000000000003"
		adminRole   = "00000000-0000-0000-0000-000000000004"
	)

	users := []struct {
		n, name, email, phone, role, date, pass string
	}{
		{"1", "Анна Соколова", "client@forma.test", "+7 900 111-22-33", clientRole, "2026-09-01T10:00:00", "client"},
		{"2", "Мария Орлова", "maria@forma.test", "+7 900 222-33-44", clientRole, "2026-10-20T15:30:00", "maria"},
		{"3", "Игорь Лебедев", "manager@forma.test", "+7 900 333-44-55", managerRole, "2026-08-15T09:00:00", "manager"},
		{"4", "Ольга Крылова", "admin@forma.test", "+7 900 444-55-66", adminRole, "2026-08-01T09:00:00", "admin"},
	}
	for i, u := range users {
		_, err := tx.Exec(`
			INSERT INTO users (id, name, email, phone, role_id, registration_date, password)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, nid("b0000000-0000-4000-8000", i+1), u.name, u.email, u.phone, u.role, u.date, u.pass)
		if err != nil {
			return err
		}
	}

	type row struct {
		name, typ, gender, size, color, desc, brand, season string
		price                                               float64
		stock                                               int
	}
	products := []row{
		{"Базовая футболка", "Футболка", "унисекс", "M", "белый", "Плотный хлопок, прямой крой, спокойная посадка.", "FORMA", "лето", 2490, 18},
		{"Футболка оверсайз", "Футболка", "мужской", "L", "чёрный", "Свободный силуэт и тяжёлый трикотаж.", "FORMA", "демисезон", 2990, 12},
		{"Льняная рубашка", "Рубашка", "мужской", "M", "бежевый", "Натуральный лён, воротник на пуговицах, летний вес ткани.", "Север", "лето", 6490, 7},
		{"Рубашка в полоску", "Рубашка", "женский", "S", "синий", "Хлопок в тонкую полоску, приталенный крой.", "Север", "демисезон", 5890, 15},
		{"Брюки чинос", "Брюки", "мужской", "L", "хаки", "Мягкий твил, пояс на пуговице, заужены книзу.", "Linea", "демисезон", 7290, 9},
		{"Широкие брюки", "Брюки", "женский", "M", "чёрный", "Высокая посадка и широкая штанина.", "Linea", "демисезон", 7990, 6},
		{"Платье миди", "Платье", "женский", "S", "бежевый", "Лёгкая ткань, длина миди, тонкие бретели.", "Atelier N", "лето", 9990, 11},
		{"Шерстяное платье", "Платье", "женский", "M", "серый", "Плотная шерсть, длинный рукав, прямой силуэт.", "Atelier N", "зима", 12990, 4},
		{"Бомбер", "Куртка", "унисекс", "M", "чёрный", "Короткий бомбер на молнии, гладкая плащёвка.", "Nord", "демисезон", 14990, 8},
		{"Джинсовая куртка", "Куртка", "мужской", "XL", "синий", "Классическая джинсовая куртка, фурнитура без блеска.", "Nord", "демисезон", 11990, 14},
		{"Свитер из мериноса", "Свитер", "унисекс", "M", "серый", "Тонкий меринос, резинка по краю.", "Wool&Co", "зима", 8990, 20},
		{"Свитер крупной вязки", "Свитер", "мужской", "L", "хаки", "Объёмная вязка и высокий ворот.", "Wool&Co", "демисезон", 7590, 5},
		{"Юбка плиссе", "Юбка", "женский", "XS", "белый", "Плиссированная юбка длины миди.", "FORMA", "демисезон", 5490, 10},
		{"Шерстяное пальто", "Пальто", "женский", "S", "чёрный", "Двубортное пальто, длина ниже колена.", "Atelier N", "зима", 24990, 3},
	}
	for i, p := range products {
		id := nid("c0000000-0000-4000-8000", i+1)
		_, err := tx.Exec(`
			INSERT INTO products (id, article, name, type, gender, size, color, description, brand, season, price, stock)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, strconv.Itoa(1001+i), p.name, p.typ, p.gender, p.size, p.color, p.desc, p.brand, p.season, p.price, p.stock)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO photos (id, filename, product_id) VALUES (?, ?, ?)
		`, nid("d0000000-0000-4000-8000", i+1), fmt.Sprintf("product-%02d.jpg", i+1), id)
		if err != nil {
			return err
		}
	}
	_, err := tx.Exec(`
		INSERT INTO photos (id, filename, product_id) VALUES (?, ?, ?)
	`, nid("d0000000-0000-4000-8000", 15), "product-14-detail.jpg", nid("c0000000-0000-4000-8000", 14))
	if err != nil {
		return err
	}

	anna := nid("b0000000-0000-4000-8000", 1)
	maria := nid("b0000000-0000-4000-8000", 2)
	_, err = tx.Exec(`INSERT INTO cart (id, user_id, product_id, count) VALUES (?, ?, ?, ?)`,
		nid("e0000000-0000-4000-8000", 1), anna, nid("c0000000-0000-4000-8000", 13), 1)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO cart (id, user_id, product_id, count) VALUES (?, ?, ?, ?)`,
		nid("e0000000-0000-4000-8000", 2), anna, nid("c0000000-0000-4000-8000", 9), 2)
	if err != nil {
		return err
	}

	orders := []struct {
		id, user, date, status string
		number                 int
	}{
		{nid("f0000000-0000-4000-8000", 1), anna, "2026-09-15T11:20:00", "Отменён", 1001},
		{nid("f0000000-0000-4000-8000", 2), anna, "2026-10-28T16:40:00", "В работе", 1002},
		{nid("f0000000-0000-4000-8000", 3), maria, "2026-11-05T13:05:00", "Готов к выдаче", 1003},
	}
	for _, o := range orders {
		_, err = tx.Exec(`INSERT INTO orders (id, number, user_id, date_time, status) VALUES (?, ?, ?, ?, ?)`, o.id, o.number, o.user, o.date, o.status)
		if err != nil {
			return err
		}
	}
	items := []struct {
		n     int
		order int
		prod  int
	}{
		{1, 1, 1},
		{2, 2, 3},
		{3, 2, 5},
		{4, 3, 14},
	}
	for _, it := range items {
		var price float64
		pid := nid("c0000000-0000-4000-8000", it.prod)
		if err = tx.Get(&price, `SELECT price FROM products WHERE id = ?`, pid); err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO order_items (id, order_id, product_id, price) VALUES (?, ?, ?, ?)`,
			nid("11000000-0000-4000-8000", it.n), nid("f0000000-0000-4000-8000", it.order), pid, price)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(`
		UPDATE products SET stock = stock - (
			SELECT COUNT(*) FROM order_items oi
			JOIN orders o ON o.id = oi.order_id
			WHERE oi.product_id = products.id AND o.status != 'Отменён'
		)
	`)
	return err
}
