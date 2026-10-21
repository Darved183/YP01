package database

import (
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
	_, err := r.DB.NamedExec(`
		INSERT INTO products (id, name, type, gender, size, color, description, brand, season, price)
		VALUES (:id, :name, :type, :gender, :size, :color, :description, :brand, :season, :price)
	`, p)
	return err
}

func (r *Repository) GetProducts(f ProductFilter) ([]models.Product, error) {
	var products []models.Product
	query := `SELECT id, name, type, gender, size, color, description, brand, season, price FROM products WHERE 1=1`
	args := []interface{}{}

	if f.Type != ""   { query += " AND type = ?";   args = append(args, f.Type) }
	if f.Gender != "" { query += " AND gender = ?"; args = append(args, f.Gender) }
	if f.Size != ""   { query += " AND size = ?";   args = append(args, f.Size) }
	if f.Color != ""  { query += " AND color = ?";  args = append(args, f.Color) }
	if f.Brand != ""  { query += " AND brand = ?";  args = append(args, f.Brand) }
	if f.Season != "" { query += " AND season = ?"; args = append(args, f.Season) }

	if strings.EqualFold(f.Sort, "price") {
		order := "ASC"
		if strings.EqualFold(f.Order, "desc") {
			order = "DESC"
		}
		query += " ORDER BY price " + order
	} else {
		query += " ORDER BY id ASC"
	}

	if f.Page < 1 { f.Page = 1 }
	if f.Limit < 1 { f.Limit = 10 }
	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, (f.Page-1)*f.Limit)

	err := r.DB.Select(&products, query, args...)
	return products, err
}

func (r *Repository) GetProductByID(id string) (*models.Product, error) {
	var p models.Product
	if err := r.DB.Get(&p, `SELECT * FROM products WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) PutProduct(p *models.Product) error {
	_, err := r.DB.NamedExec(`
		UPDATE products SET
			name = :name, type = :type, gender = :gender, size = :size,
			color = :color, description = :description, brand = :brand,
			season = :season, price = :price
		WHERE id = :id
	`, p)
	return err
}

func (r *Repository) DeleteProduct(id string) error {
	_, err := r.DB.Exec(`DELETE FROM products WHERE id = ?`, id)
	return err
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

// Role

func (r *Repository) CreateRole(role *models.Role) error {
	if role.ID == "" {
		role.ID = uuid.NewString()
	}
	_, err := r.DB.NamedExec(`INSERT INTO roles (id, name) VALUES (:id, :name)`, role)
	return err
}

func (r *Repository) GetRoles() ([]models.Role, error) {
	var roles []models.Role
	err := r.DB.Select(&roles, `SELECT * FROM roles ORDER BY id ASC`)
	return roles, err
}

func (r *Repository) PutRole(role *models.Role) error {
	_, err := r.DB.NamedExec(`UPDATE roles SET name = :name WHERE id = :id`, role)
	return err
}

func (r *Repository) DeleteRole(id string) error {
	_, err := r.DB.Exec(`DELETE FROM roles WHERE id = ?`, id)
	return err
}

// User

type UserFilter struct {
	Page  int
	Limit int
}

func (r *Repository) CreateUser(u *models.User) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	_, err := r.DB.NamedExec(`
		INSERT INTO users (id, name, email, phone, role_id, registration_date)
		VALUES (:id, :name, :email, :phone, :role_id, :registration_date)
	`, u)
	return err
}

func (r *Repository) GetUsers(f UserFilter) ([]models.User, error) {
	if f.Page < 1 { f.Page = 1 }
	if f.Limit < 1 { f.Limit = 10 }
	var users []models.User
	err := r.DB.Select(&users,
		`SELECT * FROM users ORDER BY registration_date ASC LIMIT ? OFFSET ?`,
		f.Limit, (f.Page-1)*f.Limit,
	)
	return users, err
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

// Cart

func (r *Repository) CreateCart(c *models.Cart) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	_, err := r.DB.NamedExec(`
		INSERT INTO cart (id, user_id, product_id, count) VALUES (:id, :user_id, :product_id, :count)
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

// Order

type OrderFilter struct {
	Page      int
	Limit     int
	Status    string
	StartDate string
	EndDate   string
}

func (r *Repository) CreateOrder(o *models.Order) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	_, err := r.DB.NamedExec(`
		INSERT INTO orders (id, user_id, date_time, status) VALUES (:id, :user_id, :date_time, :status)
	`, o)
	return err
}

func (r *Repository) GetOrders(f OrderFilter) ([]models.Order, error) {
	var orders []models.Order
	query := `SELECT * FROM orders WHERE 1=1`
	args := []interface{}{}

	if f.Status != ""    { query += " AND status = ?";    args = append(args, f.Status) }
	if f.StartDate != "" { query += " AND date_time >= ?"; args = append(args, f.StartDate) }
	if f.EndDate != ""   { query += " AND date_time <= ?"; args = append(args, f.EndDate) }

	query += " ORDER BY date_time DESC"

	if f.Page < 1 { f.Page = 1 }
	if f.Limit < 1 { f.Limit = 10 }
	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, (f.Page-1)*f.Limit)

	err := r.DB.Select(&orders, query, args...)
	return orders, err
}

func (r *Repository) GetOrder(id string) (*models.Order, error) {
	var o models.Order
	if err := r.DB.Get(&o, `SELECT * FROM orders WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) PutOrder(o *models.Order) error {
	_, err := r.DB.NamedExec(`
		UPDATE orders SET user_id = :user_id, date_time = :date_time, status = :status WHERE id = :id
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