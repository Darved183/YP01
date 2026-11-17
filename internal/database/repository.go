package database

import (
	"fmt"
	"strconv"
	"strings"

	"YP01/internal/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{DB: db}
}

// Product

type ProductFilter struct {
	Page   int
	Limit  int
	Type   string
	Gender string
	Size   string
	Color  string
	Brand  string
	Season string
	Sort   string
	Order  string
}

func (r *Repository) CreateProduct(p *models.Product) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Article == "" {
		article, err := r.nextArticle()
		if err != nil {
			return err
		}
		p.Article = article
	}
	_, err := r.DB.NamedExec(`
		INSERT INTO products (id, article, name, type, gender, size, color, description, brand, season, price, stock)
		VALUES (:id, :article, :name, :type, :gender, :size, :color, :description, :brand, :season, :price, :stock)
	`, p)
	return err
}

func (r *Repository) nextArticle() (string, error) {
	var n int
	err := r.DB.Get(&n, `SELECT COALESCE(MAX(CAST(article AS INTEGER)), 1000) FROM products WHERE article != ''`)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(n + 1), nil
}

func productWhere(f ProductFilter) (string, []interface{}) {
	query := ` WHERE 1=1`
	args := []interface{}{}

	if f.Type != "" {
		query += " AND type = ?"
		args = append(args, f.Type)
	}
	if f.Gender != "" {
		query += " AND gender = ?"
		args = append(args, f.Gender)
	}
	if f.Size != "" {
		query += " AND EXISTS (SELECT 1 FROM product_sizes ps WHERE ps.product_id = products.id AND ps.size = ?)"
		args = append(args, f.Size)
	}
	if f.Color != "" {
		query += " AND color = ?"
		args = append(args, f.Color)
	}
	if f.Brand != "" {
		query += " AND brand = ?"
		args = append(args, f.Brand)
	}
	if f.Season != "" {
		query += " AND season = ?"
		args = append(args, f.Season)
	}
	return query, args
}

func (r *Repository) GetProducts(f ProductFilter) ([]models.Product, error) {
	var products []models.Product
	where, args := productWhere(f)
	query := `SELECT id, article, name, type, gender, size, color, description, brand, season, price, stock FROM products` + where

	if strings.EqualFold(f.Sort, "price") {
		order := "ASC"
		if strings.EqualFold(f.Order, "desc") {
			order = "DESC"
		}
		query += " ORDER BY price " + order
	} else {
		query += " ORDER BY id ASC"
	}

	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 10
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, (f.Page-1)*f.Limit)

	err := r.DB.Select(&products, query, args...)
	return products, err
}

func (r *Repository) CountProducts(f ProductFilter) (int, error) {
	where, args := productWhere(f)
	var n int
	err := r.DB.Get(&n, `SELECT COUNT(*) FROM products`+where, args...)
	return n, err
}

func (r *Repository) ProductField(field string) ([]string, error) {
	switch field {
	case "type", "gender", "size", "color", "brand", "season":
	default:
		return nil, fmt.Errorf("bad field")
	}
	if field == "size" {
		var out []string
		err := r.DB.Select(&out, `SELECT DISTINCT size FROM product_sizes WHERE size != '' ORDER BY size`)
		return out, err
	}
	var out []string
	err := r.DB.Select(&out, `SELECT DISTINCT `+field+` FROM products WHERE `+field+` != '' ORDER BY `+field)
	return out, err
}

func (r *Repository) GetProductByID(id string) (*models.Product, error) {
	var p models.Product
	if err := r.DB.Get(&p, `SELECT * FROM products WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &p, nil
}

// Сначала артикул, потом id: старые ссылки с id тоже открываются.
func (r *Repository) FindProduct(key string) (*models.Product, error) {
	var p models.Product
	err := r.DB.Get(&p, `SELECT * FROM products WHERE article = ?`, key)
	if err == nil {
		return &p, nil
	}
	return r.GetProductByID(key)
}

func (r *Repository) PutProduct(p *models.Product) error {
	_, err := r.DB.NamedExec(`
		UPDATE products SET
			name = :name, type = :type, gender = :gender, size = :size,
			color = :color, description = :description, brand = :brand,
			season = :season, price = :price, stock = :stock
		WHERE id = :id
	`, p)
	return err
}

func (r *Repository) DeleteProduct(id string) (int64, error) {
	res, err := r.DB.Exec(`DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Photo

func (r *Repository) CreatePhoto(p *models.Photo) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	_, err := r.DB.NamedExec(`
		INSERT INTO photos (id, filename, product_id) VALUES (:id, :filename, :product_id)
	`, p)
	return err
}

func (r *Repository) GetPhoto(id string) (*models.Photo, error) {
	var p models.Photo
	if err := r.DB.Get(&p, `SELECT * FROM photos WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) DeletePhoto(id string) error {
	_, err := r.DB.Exec(`DELETE FROM photos WHERE id = ?`, id)
	return err
}

func (r *Repository) PhotosByProduct(productID string) ([]models.Photo, error) {
	var list []models.Photo
	err := r.DB.Select(&list, `SELECT * FROM photos WHERE product_id = ? ORDER BY id`, productID)
	return list, err
}

// Role

func (r *Repository) CreateRole(role *models.Role) error {
	if role.ID == "" {
		role.ID = uuid.NewString()
	}
	_, err := r.DB.NamedExec(`INSERT INTO roles (id, name, rights) VALUES (:id, :name, :rights)`, role)
	return err
}

func (r *Repository) GetRoles() ([]models.Role, error) {
	var roles []models.Role
	err := r.DB.Select(&roles, `SELECT * FROM roles ORDER BY id ASC`)
	return roles, err
}

func (r *Repository) PutRole(role *models.Role) error {
	_, err := r.DB.NamedExec(`UPDATE roles SET name = :name, rights = :rights WHERE id = :id`, role)
	return err
}

func (r *Repository) DeleteRole(id string) error {
	_, err := r.DB.Exec(`DELETE FROM roles WHERE id = ?`, id)
	return err
}

func (r *Repository) GetRole(id string) (*models.Role, error) {
	var role models.Role
	if err := r.DB.Get(&role, `SELECT * FROM roles WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &role, nil
}

// User

type UserFilter struct {
	Page   int
	Limit  int
	RoleID string
}

func (r *Repository) CreateUser(u *models.User) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if u.Number < 1 {
		var max int
		if err = tx.Get(&max, `SELECT COALESCE(MAX(number), 0) FROM users`); err != nil {
			return err
		}
		if max < 1000 {
			max = 1000
		}
		u.Number = max + 1
	}
	_, err = tx.NamedExec(`
		INSERT INTO users (id, number, name, email, phone, role_id, registration_date, password)
		VALUES (:id, :number, :name, :email, :phone, :role_id, :registration_date, :password)
	`, u)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) GetUsers(f UserFilter) ([]models.User, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 10
	}
	var users []models.User
	query := `SELECT * FROM users WHERE 1=1`
	args := []interface{}{}
	if f.RoleID != "" {
		query += ` AND role_id = ?`
		args = append(args, f.RoleID)
	}
	query += ` ORDER BY registration_date ASC LIMIT ? OFFSET ?`
	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	err := r.DB.Select(&users, query, args...)
	return users, err
}

func (r *Repository) CountUsers(f UserFilter) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE 1=1`
	args := []interface{}{}
	if f.RoleID != "" {
		query += ` AND role_id = ?`
		args = append(args, f.RoleID)
	}
	var n int
	err := r.DB.Get(&n, query, args...)
	return n, err
}

func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	var u models.User
	if err := r.DB.Get(&u, `SELECT * FROM users WHERE lower(email) = lower(?)`, email); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) SetPassword(id, password string) error {
	_, err := r.DB.Exec(`UPDATE users SET password = ? WHERE id = ?`, password, id)
	return err
}

func (r *Repository) GetUsersByRole(roleID string) ([]models.User, error) {
	var users []models.User
	err := r.DB.Select(&users, `SELECT * FROM users WHERE role_id = ? ORDER BY registration_date ASC`, roleID)
	return users, err
}

func (r *Repository) PutUser(u *models.User) error {
	_, err := r.DB.NamedExec(`
		UPDATE users SET
			name = :name, email = :email, phone = :phone,
			role_id = :role_id, registration_date = :registration_date
		WHERE id = :id
	`, u)
	return err
}

func (r *Repository) DeleteUser(id string) error {
	_, err := r.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *Repository) GetUser(id string) (*models.User, error) {
	var u models.User
	if err := r.DB.Get(&u, `SELECT * FROM users WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &u, nil
}

// Cart

func (r *Repository) CreateCart(c *models.Cart) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	_, err := r.DB.NamedExec(`
		INSERT INTO cart (id, user_id, product_id, size, count) VALUES (:id, :user_id, :product_id, :size, :count)
	`, c)
	return err
}

func (r *Repository) GetCartByUser(userID string) ([]models.Cart, error) {
	var carts []models.Cart
	err := r.DB.Select(&carts, `SELECT * FROM cart WHERE user_id = ? ORDER BY id ASC`, userID)
	return carts, err
}

func (r *Repository) PutCart(c *models.Cart) error {
	_, err := r.DB.NamedExec(`
		UPDATE cart SET user_id = :user_id, product_id = :product_id, count = :count WHERE id = :id
	`, c)
	return err
}

func (r *Repository) DeleteCart(id string) error {
	_, err := r.DB.Exec(`DELETE FROM cart WHERE id = ?`, id)
	return err
}

func (r *Repository) GetCart(id string) (*models.Cart, error) {
	var c models.Cart
	if err := r.DB.Get(&c, `SELECT * FROM cart WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &c, nil
}

// Order

type OrderFilter struct {
	Page      int
	Limit     int
	Status    string
	StartDate string
	EndDate   string
	UserID    string
}

func (r *Repository) CreateOrder(o *models.Order) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	if o.Number == 0 {
		n, err := r.nextOrderNumber()
		if err != nil {
			return err
		}
		o.Number = n
	}
	_, err := r.DB.NamedExec(`
		INSERT INTO orders (id, number, user_id, date_time, status, comment)
		VALUES (:id, :number, :user_id, :date_time, :status, :comment)
	`, o)
	return err
}

func (r *Repository) nextOrderNumber() (int, error) {
	var n int
	err := r.DB.Get(&n, `SELECT COALESCE(MAX(number), 0) FROM orders`)
	if n < 1000 {
		n = 1000
	}
	return n + 1, err
}

func (r *Repository) GetOrders(f OrderFilter) ([]models.Order, error) {
	var orders []models.Order
	where, args := orderWhere(f)
	query := `SELECT * FROM orders` + where + ` ORDER BY date_time DESC`

	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 10
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, (f.Page-1)*f.Limit)

	err := r.DB.Select(&orders, query, args...)
	return orders, err
}

func orderWhere(f OrderFilter) (string, []interface{}) {
	query := ` WHERE 1=1`
	args := []interface{}{}
	if f.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, f.UserID)
	}
	if f.Status != "" {
		query += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.StartDate != "" {
		query += " AND date_time >= ?"
		args = append(args, f.StartDate)
	}
	if f.EndDate != "" {
		query += " AND date_time <= ?"
		args = append(args, f.EndDate)
	}
	return query, args
}

func (r *Repository) CountOrders(f OrderFilter) (int, error) {
	where, args := orderWhere(f)
	var n int
	err := r.DB.Get(&n, `SELECT COUNT(*) FROM orders`+where, args...)
	return n, err
}

func (r *Repository) ClearCart(userID string) error {
	_, err := r.DB.Exec(`DELETE FROM cart WHERE user_id = ?`, userID)
	return err
}

func (r *Repository) GetOrder(id string) (*models.Order, error) {
	var o models.Order
	if err := r.DB.Get(&o, `SELECT * FROM orders WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &o, nil
}

// Номер заказа в адресе страницы. Старая ссылка с id тоже открывается.
func (r *Repository) FindOrder(key string) (*models.Order, error) {
	if n, err := strconv.Atoi(key); err == nil && n > 0 {
		var o models.Order
		if err = r.DB.Get(&o, `SELECT * FROM orders WHERE number = ?`, n); err == nil {
			return &o, nil
		}
	}
	return r.GetOrder(key)
}

func (r *Repository) PutOrder(o *models.Order) error {
	_, err := r.DB.NamedExec(`
		UPDATE orders SET user_id = :user_id, date_time = :date_time, status = :status, comment = :comment WHERE id = :id
	`, o)
	return err
}

// OrderItem

func (r *Repository) CreateOrderItem(i *models.OrderItem) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	_, err := r.DB.NamedExec(`
		INSERT INTO order_items (id, order_id, product_id, price)
		VALUES (:id, :order_id, :product_id, :price)
	`, i)
	return err
}

func (r *Repository) GetOrderItems(orderID string) ([]models.OrderItem, error) {
	var items []models.OrderItem
	err := r.DB.Select(&items, `SELECT * FROM order_items WHERE order_id = ? ORDER BY id ASC`, orderID)
	return items, err
}

type itemNeed struct {
	ProductID string
	Size      string
	Count     int
}

func (r *Repository) orderItemCounts(orderID string) ([]itemNeed, error) {
	items, err := r.GetOrderItems(orderID)
	if err != nil {
		return nil, err
	}
	var counts []itemNeed
	for _, item := range items {
		found := false
		for i := range counts {
			if counts[i].ProductID == item.ProductID && counts[i].Size == item.Size {
				counts[i].Count++
				found = true
				break
			}
		}
		if !found {
			counts = append(counts, itemNeed{ProductID: item.ProductID, Size: item.Size, Count: 1})
		}
	}
	return counts, nil
}

func (r *Repository) ReturnOrderStock(orderID string) error {
	counts, err := r.orderItemCounts(orderID)
	if err != nil {
		return err
	}
	for _, row := range counts {
		if row.Size == "" {
			if _, err = r.DB.Exec(`UPDATE products SET stock = stock + ? WHERE id = ?`, row.Count, row.ProductID); err != nil {
				return err
			}
			continue
		}
		if _, err = r.DB.Exec(`UPDATE product_sizes SET stock = stock + ? WHERE product_id = ? AND size = ?`, row.Count, row.ProductID, row.Size); err != nil {
			return err
		}
		if _, err = r.DB.Exec(`UPDATE products SET stock = (SELECT COALESCE(SUM(stock), 0) FROM product_sizes WHERE product_id = ?) WHERE id = ?`, row.ProductID, row.ProductID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) TakeOrderStock(orderID string) (bool, error) {
	counts, err := r.orderItemCounts(orderID)
	if err != nil {
		return false, err
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	for _, row := range counts {
		if row.Size == "" {
			res, err := tx.Exec(`UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`, row.Count, row.ProductID, row.Count)
			if err != nil {
				return false, err
			}
			got, _ := res.RowsAffected()
			if got != 1 {
				return false, nil
			}
			continue
		}
		res, err := tx.Exec(`UPDATE product_sizes SET stock = stock - ? WHERE product_id = ? AND size = ? AND stock >= ?`, row.Count, row.ProductID, row.Size, row.Count)
		if err != nil {
			return false, err
		}
		got, _ := res.RowsAffected()
		if got != 1 {
			return false, nil
		}
		if _, err = tx.Exec(`UPDATE products SET stock = (SELECT COALESCE(SUM(stock), 0) FROM product_sizes WHERE product_id = ?) WHERE id = ?`, row.ProductID, row.ProductID); err != nil {
			return false, err
		}
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) ProductSizes(productID string) ([]models.ProductSize, error) {
	var rows []models.ProductSize
	err := r.DB.Select(&rows, `SELECT product_id, size, stock FROM product_sizes WHERE product_id = ?`, productID)
	return rows, err
}

func (r *Repository) SizeStock(productID, size string) (int, error) {
	var n int
	err := r.DB.Get(&n, `SELECT stock FROM product_sizes WHERE product_id = ? AND size = ?`, productID, size)
	return n, err
}

func (r *Repository) ReplaceSizes(productID string, rows []models.ProductSize) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM product_sizes WHERE product_id = ?`, productID); err != nil {
		return err
	}
	sum := 0
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if _, err = tx.Exec(`INSERT INTO product_sizes (product_id, size, stock) VALUES (?, ?, ?)`, productID, row.Size, row.Stock); err != nil {
			return err
		}
		sum += row.Stock
		names = append(names, row.Size)
	}
	if _, err = tx.Exec(`UPDATE products SET size = ?, stock = ? WHERE id = ?`, strings.Join(names, ", "), sum, productID); err != nil {
		return err
	}
	return tx.Commit()
}
