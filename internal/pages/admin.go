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
	f := database.ProductFilter{Page: 1}
	total, err := s.Repo.CountProducts(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать товары", err)
		return
	}
	f.Limit = total
	if f.Limit < 1 {
		f.Limit = 1
	}
	products, err := s.Repo.GetProducts(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать товары", err)
		return
	}
	for _, item := range products {
		p.Cards = append(p.Cards, s.cardOf(item))
	}
	p.RangeText = rangeText(1, total, total)
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
		{"type", "Тип"}, {"gender", "Пол"}, {"size", "Размер"},
		{"color", "Цвет"}, {"brand", "Бренд"}, {"season", "Сезон"},
	}
	for _, item := range labels {
		values, err := s.Repo.ProductField(item.key)
		if err != nil {
			return err
		}
		p.Fields = append(p.Fields, filterField{Key: item.key, Label: item.label, Options: sortOptions(item.key, values)})
	}
	if p.Product != nil {
		keepValue(p.Fields, "type", p.Product.Type)
		keepValue(p.Fields, "size", p.Product.Size)
		keepValue(p.Fields, "color", p.Product.Color)
	}
	if p.Product != nil && p.Product.ID != "" {
		photos, err := s.Repo.PhotosByProduct(p.Product.ID)
		if err != nil {
			return err
		}
		for _, photo := range photos {
			p.Photos = append(p.Photos, photoCard{ID: photo.ID, Filename: photo.Filename, HasFile: fileExists(photo.Filename)})
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
	if err := s.sameAsCatalog(&prod); err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать списки", err)
		return
	}
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
		Type:        freshOrList(r, "type"),
		Gender:      strings.TrimSpace(r.FormValue("gender")),
		Size:        freshOrList(r, "size"),
		Color:       freshOrList(r, "color"),
		Brand:       strings.TrimSpace(r.FormValue("brand")),
		Season:      strings.TrimSpace(r.FormValue("season")),
		Description: strings.TrimSpace(r.FormValue("description")),
	}
	if p.Name == "" || p.Gender == "" || p.Brand == "" || p.Season == "" {
		return p, "Заполните все поля товара"
	}
	if p.Type == "" || p.Size == "" || p.Color == "" {
		return p, "Выберите тип, размер и цвет или напишите новые"
	}
	if longText(p.Type, 40) || longText(p.Size, 40) || longText(p.Color, 40) {
		return p, "Тип, размер и цвет не длиннее 40 символов"
	}
	price, ok := parsePrice(r.FormValue("price"))
	if !ok {
		return p, "Укажите корректную цену"
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

// Новое поле важнее списка. Если такое слово уже есть в каталоге, берём его написание.
func freshOrList(r *http.Request, key string) string {
	fresh := strings.Join(strings.Fields(r.FormValue(key+"_new")), " ")
	if fresh != "" {
		return fresh
	}
	return strings.Join(strings.Fields(r.FormValue(key)), " ")
}

func longText(s string, max int) bool {
	return len([]rune(s)) > max
}

func keepValue(fields []filterField, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	for i := range fields {
		if fields[i].Key != key {
			continue
		}
		for _, opt := range fields[i].Options {
			if strings.EqualFold(opt, value) {
				return
			}
		}
		fields[i].Options = append(fields[i].Options, value)
	}
}

func (s *Pages) sameAsCatalog(p *models.Product) error {
	pairs := []struct {
		key  string
		dest *string
	}{
		{"type", &p.Type},
		{"size", &p.Size},
		{"color", &p.Color},
	}
	for _, item := range pairs {
		values, err := s.Repo.ProductField(item.key)
		if err != nil {
			return err
		}
		for _, known := range values {
			if strings.EqualFold(known, *item.dest) {
				*item.dest = known
				break
			}
		}
	}
	return nil
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
		p.Hint = "Все учётные записи"
	} else {
		p.Hint = "Показаны пользователи выбранной роли."
	}
	roles, err := s.Repo.GetRoles()
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать пользователей", err)
		return
	}
	for _, role := range roles {
		if skipRole(role.Name, role.ID, p.RoleID) {
			continue
		}
		p.Roles = append(p.Roles, roleRow{ID: role.ID, Name: role.Name, Selected: role.ID == p.RoleID})
	}
	f := database.UserFilter{Page: 1, RoleID: p.RoleID}
	total, err := s.Repo.CountUsers(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать пользователей", err)
		return
	}
	f.Limit = total
	if f.Limit < 1 {
		f.Limit = 1
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
	p.RangeText = rangeText(1, total, total)
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
		p.Hint = "Гость — это посетитель без регистрации, такую роль аккаунту не дают."
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
		chosen := role.ID == user.RoleID
		if user.RoleID == "" && roleKey(role.Name) == "client" {
			chosen = true
		}
		p.Roles = append(p.Roles, roleRow{ID: role.ID, Name: role.Name, Selected: chosen})
	}
	p.Email = user.Email
	p.Buyer = user.Phone
	p.Text = user.Name
	p.RoleID = user.RoleID
	p.Next = user.ID
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
	} else if !nameOK(name) {
		msg = "Имя должно быть от 2 до 80 символов"
	} else if !validEmail(email) {
		msg = "Некорректная почта"
	} else if id == "" && len(pass) < 4 {
		msg = "Пароль должен быть не короче 4 символов"
	} else if pass != "" && len(pass) < 4 {
		msg = "Пароль должен быть не короче 4 символов"
	}
	if msg == "" {
		role, err := s.Repo.GetRole(roleID)
		if err != nil {
			msg = "Роль не найдена"
		} else if skipRole(role.Name, role.ID, "") {
			msg = "Гостю роль не назначают: это посетитель без аккаунта"
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
		setFlash(w, msg)
		if id == "" {
			goTo(w, r, "/admin/users/new")
		} else {
			goTo(w, r, "/admin/users/"+id)
		}
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
