package pages

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"YP01/internal/models"
)

func (s *Pages) Login(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	p.Title = "Вход"
	p.Next = safeNext(r.URL.Query().Get("next"))
	s.render(w, "login", p)
}

func (s *Pages) LoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	pass := r.FormValue("password")
	next := safeNext(r.FormValue("next"))
	user, err := s.Repo.GetUserByEmail(email)
	if err != nil || user.Password != pass || pass == "" {
		p := s.base(w, r)
		p.Title = "Вход"
		p.Error = "Неверная почта или пароль"
		p.Email = email
		p.Next = next
		s.render(w, "login", p)
		return
	}
	setUserCookie(w, user.ID)
	setFlash(w, "Вход выполнен")
	goTo(w, r, next)
}

func (s *Pages) RegisterPage(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	p.Title = "Регистрация"
	p.Hint = "После регистрации можно сразу собирать корзину."
	p.Next = safeNext(r.URL.Query().Get("next"))
	s.render(w, "register", p)
}

func (s *Pages) RegisterPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	phone := strings.TrimSpace(r.FormValue("phone"))
	pass := r.FormValue("password")
	next := safeNext(r.FormValue("next"))
	msg := ""
	if name == "" || email == "" || phone == "" {
		msg = "Заполните имя, почту и телефон"
	} else if !validEmail(email) {
		msg = "Некорректная почта"
	} else if len(phone) < 5 {
		msg = "Укажите телефон"
	} else if len(pass) < 4 {
		msg = "Пароль должен быть не короче 4 символов"
	}
	if msg == "" {
		if _, err := s.Repo.GetUserByEmail(email); err == nil {
			msg = "Такая почта уже зарегистрирована"
		} else if !errors.Is(err, sql.ErrNoRows) {
			p := s.base(w, r)
			s.fail(w, p, "Ошибка", "Не получилось создать аккаунт", err)
			return
		}
	}
	if msg != "" {
		p := s.base(w, r)
		p.Title = "Регистрация"
		p.Hint = "После регистрации можно сразу собирать корзину."
		p.Error = msg
		p.Email = email
		p.Next = next
		p.Text = name
		p.Buyer = phone
		s.render(w, "register", p)
		return
	}
	role, err := s.clientRole()
	if err != nil {
		p := s.base(w, r)
		s.fail(w, p, "Ошибка", "Роль клиента не найдена", err)
		return
	}
	user := models.User{
		Name: name, Email: email, Phone: phone, RoleID: role.ID,
		RegistrationDate: time.Now().Format("2006-01-02T15:04:05"),
		Password:         pass,
	}
	if err = s.Repo.CreateUser(&user); err != nil {
		p := s.base(w, r)
		s.fail(w, p, "Ошибка", "Не получилось создать аккаунт", err)
		return
	}
	setUserCookie(w, user.ID)
	setFlash(w, "Аккаунт создан")
	goTo(w, r, next)
}

func (s *Pages) Logout(w http.ResponseWriter, r *http.Request) {
	clearUserCookie(w)
	setFlash(w, "Вы вышли из аккаунта")
	goTo(w, r, "/catalog")
}

func (s *Pages) clientRole() (*models.Role, error) {
	roles, err := s.Repo.GetRoles()
	if err != nil {
		return nil, err
	}
	for i := range roles {
		if roleKey(roles[i].Name) == "client" {
			return &roles[i], nil
		}
	}
	return nil, sql.ErrNoRows
}
