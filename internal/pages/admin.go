package pages

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"YP01/internal/database"
	"YP01/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Админка тоже обычные формы. Сохранение идёт в ту же базу, отдельного api тут нет.
func (s *Pages) AdminProducts(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Каталог редактирует администратор."
		s.render(w, "message", p)
		return
	}
	p.Title = "Товары"
	p.Hint = "Добавление, правка и удаление каталога."
	p.ActionHref = "/admin/products/new"
	p.ActionLabel = "Новый товар"
	f := database.ProductFilter{Page: pageNum(r), Limit: 6}
	total, err := s.Repo.CountProducts(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать товары", err)
		return
	}
	products, err := s.Repo.GetProducts(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать товары", err)
		return
	}
	for _, item := range products {
		p.Cards = append(p.Cards, s.cardOf(item))
	}
	p.Page = f.Page
	p.Pages = pageCount(total, f.Limit)
	p.RangeText = rangeText(f.Page, f.Limit, total)
	p.PagePath = "/admin/products"
	p.PrevPage = f.Page - 1
	p.NextPage = f.Page + 1
	s.render(w, "products", p)
}

func (s *Pages) ProductForm(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Карточку товара меняет администратор."
		s.render(w, "message", p)
		return
	}
	id := mux.Vars(r)["id"]
	p.Product = &models.Product{}
	if id != "" {
		item, err := s.Repo.GetProductByID(id)
		if err != nil {
			p.Title = "Товар не найден"
			p.Text = "Такой страницы нет."
			s.render(w, "message", p)
			return
		}
		p.Product = item
		p.Editing = true
		p.Title = "Правка товара"
	} else {
		p.Title = "Новый товар"
		p.Hint = "Фотографий можно приложить сразу несколько."
	}
	if err := s.fillProductForm(&p); err != nil {
		s.fail(w, p, "Ошибка", "Не получилось открыть форму", err)
		return
	}
	s.render(w, "product_form", p)
}

func (s *Pages) fillProductForm(p *page) error {
	labels := []struct{ key, label string }{
		{"type", "Тип"}, {"color", "Цвет"}, {"brand", "Бренд"},
	}
	for _, item := range labels {
		values, err := s.Repo.ProductField(item.key)
		if err != nil {
			return err
		}
		p.Fields = append(p.Fields, filterField{Key: item.key, Label: item.label, Options: sortOptions(item.key, values)})
	}
	if p.Product != nil && p.Product.ID != "" {
		photos, err := s.Repo.PhotosByProduct(p.Product.ID)
		if err != nil {
			return err
		}
		for _, photo := range photos {
			if !fileExists(photo.Filename) {
				continue
			}
			p.Photos = append(p.Photos, photoCard{ID: photo.ID, Filename: photo.Filename, HasFile: true})
		}
	}
	return nil
}

func (s *Pages) ProductSave(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Карточку товара меняет администратор."
		s.render(w, "message", p)
		return
	}
	if err := parseAnyForm(r); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	prod, msg := readProduct(r)
	p.Product = &prod
	p.Editing = prod.ID != ""
	if msg != "" {
		p.Title = "Товар"
		p.Error = msg
		_ = s.fillProductForm(&p)
		s.render(w, "product_form", p)
		return
	}
	var err error
	if prod.ID == "" {
		err = s.Repo.CreateProduct(&prod)
	} else {
		if _, err = s.Repo.GetProductByID(prod.ID); err == nil {
			err = s.Repo.PutProduct(&prod)
		}
	}
	if err != nil {
		p.Title = "Товар"
		p.Error = "Не получилось сохранить товар"
		_ = s.fillProductForm(&p)
		s.render(w, "product_form", p)
		return
	}
	saved, failed := s.savePostedPhotos(r, prod.ID)
	if failed > 0 && saved == 0 {
		setFlash(w, "Товар сохранён, фото не записалось")
	} else if failed > 0 {
		setFlash(w, "Товар сохранён, часть фото не записалась")
	} else if p.Editing {
		setFlash(w, "Товар сохранён")
	} else {
		setFlash(w, "Товар добавлен")
	}
	goTo(w, r, "/admin/products/"+prod.ID)
}

func readProduct(r *http.Request) (models.Product, string) {
	p := models.Product{
		ID:          r.FormValue("id"),
		Name:        strings.TrimSpace(r.FormValue("name")),
		Type:        strings.TrimSpace(r.FormValue("type")),
		Gender:      strings.TrimSpace(r.FormValue("gender")),
		Size:        strings.TrimSpace(r.FormValue("size")),
		Color:       strings.TrimSpace(r.FormValue("color")),
		Brand:       strings.TrimSpace(r.FormValue("brand")),
		Season:      strings.TrimSpace(r.FormValue("season")),
		Description: strings.TrimSpace(r.FormValue("description")),
	}
	if p.Name == "" || p.Type == "" || p.Gender == "" || p.Size == "" || p.Color == "" || p.Brand == "" || p.Season == "" {
		return p, "Заполните все поля товара"
	}
	switch p.Gender {
	case "женский", "мужской", "унисекс":
	default:
		return p, "Пол: женский, мужской или унисекс"
	}
	switch p.Size {
	case "XS", "S", "M", "L", "XL":
	default:
		return p, "Размер: XS, S, M, L или XL"
	}
	switch p.Season {
	case "лето", "зима", "демисезон", "всесезон":
	default:
		return p, "Сезон: лето, зима, демисезон или всесезон"
	}
	price, ok := parsePrice(r.FormValue("price"))
	if !ok || price <= 0 {
		return p, "Укажите цену больше нуля"
	}
	p.Price = price
	stock, err := strconv.Atoi(strings.TrimSpace(r.FormValue("stock")))
	if err != nil || stock < 0 || stock > 1000000 {
		return p, "Укажите, сколько штук на складе"
	}
	p.Stock = stock
	runes := []rune(p.Description)
	if len(runes) > 1000 {
		p.Description = string(runes[:1000])
	}
	runes = []rune(p.Name)
	if len(runes) > 120 {
		p.Name = string(runes[:120])
	}
	return p, ""
}

func (s *Pages) ProductDelete(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Каталог редактирует администратор."
		s.render(w, "message", p)
		return
	}
	if err := parseAnyForm(r); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id := r.FormValue("id")
	photos, _ := s.Repo.PhotosByProduct(id)
	n, err := s.Repo.DeleteProduct(id)
	if err != nil {
		if strings.Contains(err.Error(), "constraint") {
			setFlash(w, "Нельзя удалить: товар есть в заказе")
		} else {
			setFlash(w, "Не получилось удалить товар")
		}
		goTo(w, r, "/admin/products/"+id)
		return
	}
	if n == 0 {
		setFlash(w, "Товар не найден")
		goTo(w, r, "/admin/products")
		return
	}
	for _, photo := range photos {
		_ = os.Remove(filepath.Join("uploads", filepath.Base(photo.Filename)))
	}
	setFlash(w, "Товар удалён")
	goTo(w, r, "/admin/products")
}

func (s *Pages) PhotoAdd(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Фотографии добавляет администратор."
		s.render(w, "message", p)
		return
	}
	if err := parseAnyForm(r); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	productID := r.FormValue("product_id")
	if _, err := s.Repo.GetProductByID(productID); err != nil {
		setFlash(w, "Товар не найден")
		goTo(w, r, "/admin/products")
		return
	}
	saved, failed := s.savePostedPhotos(r, productID)
	if saved == 0 {
		setFlash(w, "Выберите файлы")
	} else if failed > 0 {
		setFlash(w, "Добавлено не всё")
	} else if saved == 1 {
		setFlash(w, "Фотография добавлена")
	} else {
		setFlash(w, "Фотографии добавлены")
	}
	goTo(w, r, "/admin/products/"+productID)
}

func (s *Pages) PhotoDelete(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Фотографии удаляет администратор."
		s.render(w, "message", p)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	photo, err := s.Repo.GetPhoto(r.FormValue("id"))
	if err != nil {
		setFlash(w, "Фотография не найдена")
		goTo(w, r, "/catalog")
		return
	}
	if err = s.Repo.DeletePhoto(photo.ID); err != nil {
		setFlash(w, "Не получилось удалить фотографию")
	} else {
		_ = os.Remove(filepath.Join("uploads", filepath.Base(photo.Filename)))
		setFlash(w, "Фотография удалена")
	}
	back := r.FormValue("back")
	if back == "" {
		back = "/admin/products/" + photo.ProductID
	}
	goTo(w, r, back)
}

func (s *Pages) savePostedPhotos(r *http.Request, productID string) (int, int) {
	if r.MultipartForm == nil {
		return 0, 0
	}
	saved, failed := 0, 0
	for _, header := range r.MultipartForm.File["file"] {
		if header.Size == 0 {
			continue
		}
		file, err := header.Open()
		if err != nil {
			failed++
			continue
		}
		name, err := savePhotoFile(file, header.Filename)
		file.Close()
		if err != nil {
			failed++
			continue
		}
		if err = s.Repo.CreatePhoto(&models.Photo{Filename: name, ProductID: productID}); err != nil {
			_ = os.Remove(filepath.Join("uploads", name))
			failed++
			continue
		}
		saved++
	}
	return saved, failed
}

func savePhotoFile(file io.Reader, original string) (string, error) {
	name := uuid.NewString() + filepath.Ext(original)
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", err
	}
	path := filepath.Join("uploads", name)
	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err = io.Copy(dst, file); err != nil {
		os.Remove(path)
		return "", err
	}
	return name, nil
}

func parseAnyForm(r *http.Request) error {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		return r.ParseMultipartForm(20 << 20)
	}
	return r.ParseForm()
}

func (s *Pages) AdminUsers(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Пользователей ведёт администратор."
		s.render(w, "message", p)
		return
	}
	p.Title = "Пользователи"
	p.ActionHref = "/admin/users/new"
	p.ActionLabel = "Новый пользователь"
	p.RoleID = r.URL.Query().Get("role")
	if p.RoleID == "" {
		p.Hint = "Полный список с страницами. Фильтр сверху оставляет одну роль."
	} else {
		p.Hint = "Показаны пользователи выбранной роли."
	}
	roles, err := s.Repo.GetRoles()
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать пользователей", err)
		return
	}
	for _, role := range roles {
		p.Roles = append(p.Roles, roleRow{ID: role.ID, Name: role.Name, Selected: role.ID == p.RoleID})
	}
	f := database.UserFilter{Page: pageNum(r), Limit: 2, RoleID: p.RoleID}
	total, err := s.Repo.CountUsers(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать пользователей", err)
		return
	}
	users, err := s.Repo.GetUsers(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать пользователей", err)
		return
	}
	for _, user := range users {
		row := userRow{
			ID: user.ID, Name: user.Name, Email: user.Email, Phone: user.Phone,
			Date: formatDate(user.RegistrationDate),
		}
		if role, err := s.Repo.GetRole(user.RoleID); err == nil {
			row.Role = role.Name
		} else {
			row.Role = "—"
		}
		p.Users = append(p.Users, row)
	}
	p.Keep = keepQuery{Role: p.RoleID}
	p.Page = f.Page
	p.Pages = pageCount(total, f.Limit)
	p.RangeText = rangeText(f.Page, f.Limit, total)
	p.PagePath = "/admin/users"
	p.PrevPage = f.Page - 1
	p.NextPage = f.Page + 1
	s.render(w, "users", p)
}

func (s *Pages) UserForm(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Учётные записи меняет администратор."
		s.render(w, "message", p)
		return
	}
	id := mux.Vars(r)["id"]
	user := &models.User{}
	if id != "" {
		found, err := s.Repo.GetUser(id)
		if err != nil {
			p.Title = "Пользователь не найден"
			p.Text = "Такой страницы нет."
			s.render(w, "message", p)
			return
		}
		user = found
		p.Editing = true
		p.Title = "Правка пользователя"
		p.When = formatDate(user.RegistrationDate)
	} else {
		p.Title = "Новый пользователь"
		p.Hint = "Администратор назначает любую роль."
	}
	roles, err := s.Repo.GetRoles()
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось открыть форму", err)
		return
	}
	for _, role := range roles {
		if skipRole(role.Name, role.ID, user.RoleID) {
			continue
		}
		p.Roles = append(p.Roles, roleRow{ID: role.ID, Name: role.Name, Selected: role.ID == user.RoleID})
	}
	p.Email = user.Email
	p.Buyer = user.Phone
	p.Text = user.Name
	p.RoleID = user.RoleID
	p.Next = user.ID
	s.render(w, "user_form", p)
}

func (s *Pages) showUserForm(w http.ResponseWriter, p page, id, name, email, phone, roleID, msg string) {
	p.Error = msg
	p.Text = name
	p.Email = email
	p.Buyer = phone
	p.RoleID = roleID
	p.Next = id
	if id != "" {
		p.Editing = true
		p.Title = "Правка пользователя"
		if user, err := s.Repo.GetUser(id); err == nil {
			p.When = formatDate(user.RegistrationDate)
		}
	} else {
		p.Title = "Новый пользователь"
		p.Hint = "Администратор назначает любую роль."
	}
	roles, err := s.Repo.GetRoles()
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось открыть форму", err)
		return
	}
	for _, role := range roles {
		if skipRole(role.Name, role.ID, roleID) {
			continue
		}
		p.Roles = append(p.Roles, roleRow{ID: role.ID, Name: role.Name, Selected: role.ID == roleID})
	}
	s.render(w, "user_form", p)
}

func (s *Pages) UserSave(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Учётные записи меняет администратор."
		s.render(w, "message", p)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id := r.FormValue("id")
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	phone := strings.TrimSpace(r.FormValue("phone"))
	roleID := r.FormValue("role_id")
	pass := r.FormValue("password")
	msg := ""
	if name == "" || email == "" || phone == "" || roleID == "" {
		msg = "Заполните данные пользователя и роль"
	} else if !validEmail(email) {
		msg = "Почта должна быть вида имя@почта.ru"
	} else if !validPhone(phone) {
		msg = "Телефон укажите в формате +7 900 000-00-00"
	} else if id == "" && len(pass) < 4 {
		msg = "Пароль должен быть не короче 4 символов"
	} else if pass != "" && len(pass) < 4 {
		msg = "Пароль должен быть не короче 4 символов"
	}
	if msg == "" {
		role, err := s.Repo.GetRole(roleID)
		kept := ""
		if id != "" {
			if current, err := s.Repo.GetUser(id); err == nil {
				kept = current.RoleID
			}
		}
		if err != nil {
			msg = "Роль не найдена"
		} else if skipRole(role.Name, role.ID, kept) {
			msg = "Роль «Гость» не назначают учётной записи"
		}
	}
	if msg == "" {
		other, err := s.Repo.GetUserByEmail(email)
		if err == nil && other.ID != id {
			msg = "Такая почта уже зарегистрирована"
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.fail(w, p, "Ошибка", "Не получилось сохранить пользователя", err)
			return
		}
	}
	if msg == "" && id != "" && p.User != nil && id == p.User.ID {
		current, _ := s.Repo.GetUser(id)
		if current != nil && current.RoleID != roleID {
			msg = "Нельзя сменить роль своей учётной записи"
		}
	}
	if msg != "" {
		s.showUserForm(w, p, id, name, email, phone, roleID, msg)
		return
	}
	if id == "" {
		user := models.User{
			Name: name, Email: email, Phone: phone, RoleID: roleID,
			RegistrationDate: nowStamp(), Password: pass,
		}
		if err := s.Repo.CreateUser(&user); err != nil {
			setFlash(w, "Не получилось создать пользователя")
			goTo(w, r, "/admin/users/new")
			return
		}
		setFlash(w, "Пользователь создан")
		goTo(w, r, "/admin/users")
		return
	}
	user, err := s.Repo.GetUser(id)
	if err != nil {
		setFlash(w, "Пользователь не найден")
		goTo(w, r, "/admin/users")
		return
	}
	user.Name = name
	user.Email = email
	user.Phone = phone
	user.RoleID = roleID
	if err = s.Repo.PutUser(user); err != nil {
		setFlash(w, "Не получилось сохранить пользователя")
		goTo(w, r, "/admin/users/"+id)
		return
	}
	if pass != "" {
		_ = s.Repo.SetPassword(id, pass)
	}
	setFlash(w, "Пользователь сохранён")
	goTo(w, r, "/admin/users")
}

func (s *Pages) UserDelete(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Пользователей ведёт администратор."
		s.render(w, "message", p)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id := r.FormValue("id")
	if p.User != nil && p.User.ID == id {
		setFlash(w, "Нельзя удалить свой аккаунт")
		goTo(w, r, "/admin/users")
		return
	}
	if err := s.Repo.DeleteUser(id); err != nil {
		if strings.Contains(err.Error(), "constraint") {
			setFlash(w, "Нельзя удалить: у пользователя есть заказы")
		} else {
			setFlash(w, "Не получилось удалить пользователя")
		}
		goTo(w, r, "/admin/users")
		return
	}
	setFlash(w, "Пользователь удалён")
	goTo(w, r, "/admin/users")
}

func (s *Pages) AdminRoles(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Роли настраивает администратор."
		s.render(w, "message", p)
		return
	}
	p.Title = "Роли"
	p.Hint = "Гость, клиент, менеджер и администратор уже есть. Свою роль можно добавить."
	roles, err := s.Repo.GetRoles()
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать роли", err)
		return
	}
	for _, role := range roles {
		p.Roles = append(p.Roles, roleRow{
			ID: role.ID, Name: role.Name, Note: roleNote(role.Name), System: systemRole(role.Name),
		})
	}
	s.render(w, "roles", p)
}

func (s *Pages) RoleSave(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Роли настраивает администратор."
		s.render(w, "message", p)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	id := r.FormValue("id")
	if name == "" {
		setFlash(w, "Введите название роли")
		goTo(w, r, "/admin/roles")
		return
	}
	roles, err := s.Repo.GetRoles()
	if err != nil {
		setFlash(w, "Не получилось сохранить роль")
		goTo(w, r, "/admin/roles")
		return
	}
	for _, role := range roles {
		if strings.EqualFold(role.Name, name) && role.ID != id {
			setFlash(w, "Такая роль уже есть")
			goTo(w, r, "/admin/roles")
			return
		}
	}
	if id == "" {
		err = s.Repo.CreateRole(&models.Role{Name: name})
		if err == nil {
			setFlash(w, "Роль создана")
		}
	} else {
		role, gerr := s.Repo.GetRole(id)
		if gerr != nil {
			setFlash(w, "Роль не найдена")
			goTo(w, r, "/admin/roles")
			return
		}
		if systemRole(role.Name) {
			setFlash(w, "Название этой роли менять нельзя")
			goTo(w, r, "/admin/roles")
			return
		}
		role.Name = name
		err = s.Repo.PutRole(role)
		if err == nil {
			setFlash(w, "Роль сохранена")
		}
	}
	if err != nil {
		setFlash(w, "Не получилось сохранить роль")
	}
	goTo(w, r, "/admin/roles")
}

func (s *Pages) RoleDelete(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Роли настраивает администратор."
		s.render(w, "message", p)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	role, err := s.Repo.GetRole(r.FormValue("id"))
	if err != nil {
		setFlash(w, "Роль не найдена")
		goTo(w, r, "/admin/roles")
		return
	}
	if systemRole(role.Name) {
		setFlash(w, "Эту роль нельзя удалить")
		goTo(w, r, "/admin/roles")
		return
	}
	if err = s.Repo.DeleteRole(role.ID); err != nil {
		if strings.Contains(err.Error(), "constraint") {
			setFlash(w, "Сначала переведите пользователей на другую роль")
		} else {
			setFlash(w, "Не получилось удалить роль")
		}
	} else {
		setFlash(w, "Роль удалена")
	}
	goTo(w, r, "/admin/roles")
}
