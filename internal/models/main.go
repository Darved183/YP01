package Models

type Product struct {
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Type        string  `json:"type" db:"type"`
	Gender      string  `json:"gender" db:"gender"`
	Size        string  `json:"size" db:"size"`
	Color       string  `json:"color" db:"color"`
	Description string  `json:"description" db:"description"`
	Brand       string  `json:"brand" db:"brand"`
	Season      string  `json:"season" db:"season"`
	Price       float64 `json:"price" db:"price"`
}

type Photo struct {
	ID        string `json:"id" db:"id"`
	Filename  string `json:"filename" db:"filename"`
	ProductID string `json:"product_id" db:"product_id"`
}

type Role struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type User struct {
	ID               string    `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Email            string    `json:"email" db:"email"`
	Phone            string    `json:"phone" db:"phone"`
	Role             string    `json:"role" db:"role"`
	RegistrationDate time.Time `json:"registration_date" db:"registration_date"`
}

type Cart struct {
	ID        string `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	ProductID string `json:"product_id" db:"product_id"`
	Count     int    `json:"count" db:"count"`
}

type Order struct {
	ID       string    `json:"id" db:"id"`
	UserID   string    `json:"user_id" db:"user_id"`
	DateTime string    `json:"date_time" db:"date_time"`
	Status   string    `json:"status" db:"status"`
}

type OrderItem struct {
	ID        string  `json:"id" db:"id"`
	OrderID   string  `json:"order_id" db:"order_id"`
	ProductID string  `json:"product_id" db:"product_id"`
	Price     float64 `json:"price" db:"price"`
}