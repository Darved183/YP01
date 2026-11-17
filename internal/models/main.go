package models

type Product struct {
	ID          string  `json:"id" db:"id"`
	Article     string  `json:"article" db:"article"`
	Name        string  `json:"name" db:"name"`
	Type        string  `json:"type" db:"type"`
	Gender      string  `json:"gender" db:"gender"`
	Size        string  `json:"size" db:"size"`
	Color       string  `json:"color" db:"color"`
	Description string  `json:"description" db:"description"`
	Brand       string  `json:"brand" db:"brand"`
	Season      string  `json:"season" db:"season"`
	Price       float64 `json:"price" db:"price"`
	Stock       int     `json:"stock" db:"stock"`
}

type Photo struct {
	ID        string `json:"id" db:"id"`
	Filename  string `json:"filename" db:"filename"`
	ProductID string `json:"product_id" db:"product_id"`
}

type Role struct {
	ID     string `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	Rights string `json:"-" db:"rights"`
}

type User struct {
	ID               string `json:"id" db:"id"`
	Number           int    `json:"number" db:"number"`
	Name             string `json:"name" db:"name"`
	Email            string `json:"email" db:"email"`
	Phone            string `json:"phone" db:"phone"`
	RoleID           string `json:"role_id" db:"role_id"`
	RegistrationDate string `json:"registration_date" db:"registration_date"`
	Password         string `json:"-" db:"password"`
}

type ProductSize struct {
	ProductID string `json:"-" db:"product_id"`
	Size      string `json:"size" db:"size"`
	Stock     int    `json:"stock" db:"stock"`
}

type Cart struct {
	ID        string `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	ProductID string `json:"product_id" db:"product_id"`
	Size      string `json:"size" db:"size"`
	Count     int    `json:"count" db:"count"`
}

type Order struct {
	ID       string `json:"id" db:"id"`
	Number   int    `json:"number" db:"number"`
	UserID   string `json:"user_id" db:"user_id"`
	DateTime string `json:"date_time" db:"date_time"`
	Status   string `json:"status" db:"status"`
	Comment  string `json:"comment" db:"comment"`
	Receiver string `json:"receiver" db:"receiver"`
	Phone    string `json:"phone" db:"phone"`
	City     string `json:"city" db:"city"`
	Address  string `json:"address" db:"address"`
	Delivery string `json:"delivery" db:"delivery"`
	Payment  string `json:"payment" db:"payment"`
	Place    string `json:"place" db:"place"`
	Note     string `json:"note" db:"note"`
}

type OrderItem struct {
	ID          string  `json:"id" db:"id"`
	OrderID     string  `json:"order_id" db:"order_id"`
	ProductID   string  `json:"product_id" db:"product_id"`
	Price       float64 `json:"price" db:"price"`
	Name        string  `json:"name" db:"name"`
	Description string  `json:"description" db:"description"`
	Article     string  `json:"article" db:"article"`
	Spec        string  `json:"spec" db:"spec"`
	Size        string  `json:"size" db:"size"`
}
