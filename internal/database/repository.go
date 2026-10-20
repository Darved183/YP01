package database

import (
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{DB: db}
}

///
// Product
///

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

func (r *Repository) CreateProduct(product *Product) error {
	_, err := r.DB.NamedExec(`
		INSERT INTO products (
			name, type, gender, size, color,
			description, brand, season, price
		) VALUES (
			:name, :type, :gender, :size, :color,
			:description, :brand, :season, :price
		)
	`, product)

	return err
}

func (r *Repository) GetProducts(f ProductFilter) ([]Product, error) {
	var products []Product

	query := `
		SELECT id, name, type, gender, size, color,
		       description, brand, season, price
		FROM products
		WHERE 1 = 1
	`

	args := make([]interface{}, 0)

	if f.Type != "" {
		query += " AND type = ?"
		args = append(args, f.Type)
	}

	if f.Gender != "" {
		query += " AND gender = ?"
		args = append(args, f.Gender)
	}

	if f.Size != "" {
		query += " AND size = ?"
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

	if strings.EqualFold(f.Sort, "price") {
		order := "ASC"

		if strings.EqualFold(f.Order, "desc") {
			order = "DESC"
		}

		query += " ORDER BY price " + order
	} else {
		query += " ORDER BY id ASC"
	}

	offset := (f.Page - 1) * f.Limit

	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, offset)

	err := r.DB.Select(&products, query, args...)
	if err != nil {
		return nil, err
	}

	return products, nil
}

func (r *Repository) GetIdProduct(id int) (*Product, error) {
	var product Product

	err := r.DB.Get(
		&product,
		"SELECT * FROM products WHERE id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *Repository) PutProduct(product *Product) error {
	_, err := r.DB.NamedExec(`
		UPDATE products SET
			name = :name,
			type = :type,
			gender = :gender,
			size = :size,
			color = :color,
			description = :description,
			brand = :brand,
			season = :season,
			price = :price
		WHERE id = :id
	`, product)

	return err
}

func (r *Repository) DeleteProduct(id int) error {
	_, err := r.DB.Exec(
		"DELETE FROM products WHERE id = ?",
		id,
	)

	return err
}

///
// Photo
///

func (r *Repository) CreatePhoto(photo *Photo) error {
	_, err := r.DB.NamedExec(`
		INSERT INTO photos (product_id, url)
		VALUES (:product_id, :url)
	`, photo)

	return err
}

func (r *Repository) GetPhoto(id int) (*Photo, error) {
	var photo Photo

	err := r.DB.Get(
		&photo,
		"SELECT * FROM photos WHERE id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}

	return &photo, nil
}

func (r *Repository) DeletePhoto(id int) error {
	_, err := r.DB.Exec(
		"DELETE FROM photos WHERE id = ?",
		id,
	)

	return err
}

///
// Role
///

func (r *Repository) CreateRole(role *Role) error {
	_, err := r.DB.NamedExec(`
		INSERT INTO roles (name, description)
		VALUES (:name, :description)
	`, role)

	return err
}

func (r *Repository) GetRoles() ([]Role, error) {
	var roles []Role

	err := r.DB.Select(
		&roles,
		"SELECT * FROM roles ORDER BY id ASC",
	)
	if err != nil {
		return nil, err
	}

	return roles, nil
}

func (r *Repository) PutRole(role *Role) error {
	_, err := r.DB.NamedExec(`
		UPDATE roles SET
			name = :name,
			description = :description
		WHERE id = :id
	`, role)

	return err
}

func (r *Repository) DeleteRole(id int) error {
	_, err := r.DB.Exec(
		"DELETE FROM roles WHERE id = ?",
		id,
	)

	return err
}

///
// User
///

type UserFilter struct {
	Page  int
	Limit int
}

func (r *Repository) CreateUser(user *User) error {
	_, err := r.DB.NamedExec(`
		INSERT INTO users (
			username, email, password, role_id
		) VALUES (
			:username, :email, :password, :role_id
		)
	`, user)

	return err
}

func (r *Repository) GetUsers(f UserFilter) ([]User, error) {
	var users []User

	if f.Page < 1 {
		f.Page = 1
	}

	if f.Limit < 1 {
		f.Limit = 10
	}

	offset := (f.Page - 1) * f.Limit

	err := r.DB.Select(
		&users,
		"SELECT * FROM users ORDER BY id ASC LIMIT ? OFFSET ?",
		f.Limit,
		offset,
	)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Repository) GetUsersByRole(roleID int) ([]User, error) {
	var users []User

	err := r.DB.Select(
		&users,
		"SELECT * FROM users WHERE role_id = ? ORDER BY id ASC",
		roleID,
	)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Repository) PutUser(user *User) error {
	_, err := r.DB.NamedExec(`
		UPDATE users SET
			username = :username,
			email = :email,
			password = :password,
			role_id = :role_id
		WHERE id = :id
	`, user)

	return err
}

func (r *Repository) DeleteUser(id int) error {
	_, err := r.DB.Exec(
		"DELETE FROM users WHERE id = ?",
		id,
	)

	return err
}

///
// Cart
///

func (r *Repository) CreateCart(cart *Cart) error {
	_, err := r.DB.NamedExec(`
		INSERT INTO cart (user_id, product_id, quantity)
		VALUES (:user_id, :product_id, :quantity)
	`, cart)

	return err
}

func (r *Repository) GetCartByUser(userID int) ([]Cart, error) {
	var carts []Cart

	err := r.DB.Select(
		&carts,
		"SELECT * FROM cart WHERE user_id = ? ORDER BY id ASC",
		userID,
	)
	if err != nil {
		return nil, err
	}

	return carts, nil
}

func (r *Repository) PutCart(cart *Cart) error {
	_, err := r.DB.NamedExec(`
		UPDATE cart SET
			user_id = :user_id,
			product_id = :product_id,
			quantity = :quantity
		WHERE id = :id
	`, cart)

	return err
}

func (r *Repository) DeleteCart(id int) error {
	_, err := r.DB.Exec(
		"DELETE FROM cart WHERE id = ?",
		id,
	)

	return err
}

///
// Order
///

type OrderFilter struct {
	Page      int
	Limit     int
	Status    string
	StartDate string
	EndDate   string
}

func (r *Repository) CreateOrder(order *Order) error {
	_, err := r.DB.NamedExec(`
		INSERT INTO orders (user_id, status, total, created_at)
		VALUES (:user_id, :status, :total, :created_at)
	`, order)

	return err
}

func (r *Repository) GetOrders(f OrderFilter) ([]Order, error) {
	var orders []Order

	query := "SELECT * FROM orders WHERE 1 = 1"
	args := make([]interface{}, 0)

	if f.Status != "" {
		query += " AND status = ?"
		args = append(args, f.Status)
	}

	if f.StartDate != "" {
		query += " AND created_at >= ?"
		args = append(args, f.StartDate)
	}

	if f.EndDate != "" {
		query += " AND created_at <= ?"
		args = append(args, f.EndDate)
	}

	query += " ORDER BY created_at DESC"

	if f.Page < 1 {
		f.Page = 1
	}

	if f.Limit < 1 {
		f.Limit = 10
	}

	offset := (f.Page - 1) * f.Limit

	query += " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, offset)

	err := r.DB.Select(&orders, query, args...)
	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *Repository) GetOrder(id int) (*Order, error) {
	var order Order

	err := r.DB.Get(
		&order,
		"SELECT * FROM orders WHERE id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *Repository) PutOrder(order *Order) error {
	_, err := r.DB.NamedExec(`
		UPDATE orders SET
			user_id = :user_id,
			status = :status,
			total = :total,
			created_at = :created_at
		WHERE id = :id
	`, order)

	return err
}

///
// Order Item
///

func (r *Repository) CreateOrderItem(item *OrderItem) error {
	_, err := r.DB.NamedExec(`
		INSERT INTO order_items (
			order_id, product_id, quantity, price
		) VALUES (
			:order_id, :product_id, :quantity, :price
		)
	`, item)

	return err
}

func (r *Repository) GetOrderItems(orderID int) ([]OrderItem, error) {
	var items []OrderItem

	err := r.DB.Select(
		&items,
		"SELECT * FROM order_items WHERE order_id = ? ORDER BY id ASC",
		orderID,
	)
	if err != nil {
		return nil, err
	}

	return items, nil
}