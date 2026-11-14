package pages

import (
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"YP01/internal/database"

	"github.com/gorilla/mux"
)

// Страницы магазина. JSON по заданию остаётся в handlers.
// Здесь браузер получает обычный html: данные из базы подставляются в шаблон.

type Pages struct {
	Repo  *database.Repository
	pages map[string]*template.Template
}

// Шаблоны читаю один раз при запуске.
// layout.html — общая оболочка, второй файл — сама страница.
func New(repo *database.Repository) (*Pages, error) {
	s := &Pages{Repo: repo, pages: map[string]*template.Template{}}
	names := []string{
		"catalog", "product", "photo", "login", "register", "cart",
		"orders", "order", "products", "product_form", "users", "user_form", "roles", "message",
	}
	funcs := template.FuncMap{
		"color": colorStyle,
		"light": lightColor,
		"money": money,
	}
	for _, name := range names {
		t, err := template.New("layout").Funcs(funcs).ParseFiles(
			"frontend/templates/layout.html",
			"frontend/templates/"+name+".html",
		)
		if err != nil {
			return nil, err
		}
		s.pages[name] = t
	}
	return s, nil
}

func (s *Pages) Register(r *mux.Router) {
	r.HandleFunc("/", s.Catalog).Methods("GET")
	r.HandleFunc("/catalog", s.Catalog).Methods("GET")
	r.HandleFunc("/item/{id}", s.Product).Methods("GET")
	r.HandleFunc("/picture/{id}", s.Picture).Methods("GET")
	r.HandleFunc("/login", s.Login).Methods("GET")
	r.HandleFunc("/login", s.LoginPost).Methods("POST")
	r.HandleFunc("/register", s.RegisterPage).Methods("GET")
	r.HandleFunc("/register", s.RegisterPost).Methods("POST")
	r.HandleFunc("/logout", s.Logout).Methods("POST")
	r.HandleFunc("/cart", s.Cart).Methods("GET")
	r.HandleFunc("/cart/add", s.CartAdd).Methods("POST")
	r.HandleFunc("/cart/count", s.CartCount).Methods("POST")
	r.HandleFunc("/cart/remove", s.CartRemove).Methods("POST")
	r.HandleFunc("/checkout", s.Checkout).Methods("POST")
	r.HandleFunc("/orders", s.Orders).Methods("GET")
	r.HandleFunc("/orders/{id}", s.Order).Methods("GET")
	r.HandleFunc("/orders/{id}/status", s.OrderStatus).Methods("POST")
	r.HandleFunc("/orders/{id}/cancel", s.OrderCancel).Methods("POST")
	r.HandleFunc("/admin/products", s.AdminProducts).Methods("GET")
	r.HandleFunc("/admin/products/new", s.ProductForm).Methods("GET")
	r.HandleFunc("/admin/products/save", s.ProductSave).Methods("POST")
	r.HandleFunc("/admin/products/delete", s.ProductDelete).Methods("POST")
	r.HandleFunc("/admin/products/{id}", s.ProductForm).Methods("GET")
	r.HandleFunc("/admin/photos/add", s.PhotoAdd).Methods("POST")
	r.HandleFunc("/admin/photos/delete", s.PhotoDelete).Methods("POST")
	r.HandleFunc("/admin/users", s.AdminUsers).Methods("GET")
	r.HandleFunc("/admin/users/new", s.UserForm).Methods("GET")
	r.HandleFunc("/admin/users/save", s.UserSave).Methods("POST")
	r.HandleFunc("/admin/users/delete", s.UserDelete).Methods("POST")
	r.HandleFunc("/admin/users/{id}", s.UserForm).Methods("GET")
	r.HandleFunc("/admin/roles", s.AdminRoles).Methods("GET")
	r.HandleFunc("/admin/roles/save", s.RoleSave).Methods("POST")
	r.HandleFunc("/admin/roles/delete", s.RoleDelete).Methods("POST")
}

// В layout есть {{template "content" .}}. Туда попадает кусок из файла страницы.
func (s *Pages) render(w http.ResponseWriter, name string, p page) {
	t := s.pages[name]
	if t == nil {
		http.Error(w, "page not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", p); err != nil {
		slog.Error("template", "page", name, "error", err)
	}
}

func (s *Pages) fail(w http.ResponseWriter, p page, title, text string, err error) {
	if err != nil {
		slog.Error(title, "error", err)
	}
	p.Title = title
	p.Text = text
	s.render(w, "message", p)
}

func (s *Pages) NotFoundPage(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.NotFound(w, r)
		return
	}
	p := s.base(w, r)
	p.Title = "Страница не найдена"
	p.Text = "Такой страницы нет."
	s.render(w, "message", p)
}
