package pages

import (
	"net/http"

	"YP01/internal/database"
	"YP01/internal/models"

	"github.com/gorilla/mux"
)

func (s *Pages) Catalog(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	p.Title = "Каталог"
	p.Hint = "Одежда на каждый день."
	q := r.URL.Query()
	f := database.ProductFilter{
		Page:   1,
		Type:   q.Get("type"),
		Gender: q.Get("gender"),
		Size:   q.Get("size"),
		Color:  q.Get("color"),
		Brand:  q.Get("brand"),
		Season: q.Get("season"),
	}
	switch q.Get("sort") {
	case "price-desc":
		f.Sort, f.Order, p.Sort = "price", "desc", "price-desc"
	case "price-asc":
		f.Sort, f.Order, p.Sort = "price", "asc", "price-asc"
	case "price":
		f.Sort = "price"
		if q.Get("order") == "desc" {
			f.Order, p.Sort = "desc", "price-desc"
		} else {
			f.Order, p.Sort = "asc", "price-asc"
		}
	}
	labels := []struct{ key, label string }{
		{"type", "Тип"}, {"gender", "Пол"}, {"size", "Размер"},
		{"color", "Цвет"}, {"brand", "Бренд"}, {"season", "Сезон"},
	}
	for _, item := range labels {
		values, err := s.Repo.ProductField(item.key)
		if err != nil {
			s.fail(w, p, "Ошибка", "Не получилось прочитать каталог", err)
			return
		}
		cur := q.Get(item.key)
		p.Fields = append(p.Fields, filterField{
			Key: item.key, Label: item.label, Current: cur, Options: sortOptions(item.key, values),
		})
	}
	total, err := s.Repo.CountProducts(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать каталог", err)
		return
	}
	f.Limit = total
	if f.Limit < 1 {
		f.Limit = 1
	}
	products, err := s.Repo.GetProducts(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать каталог", err)
		return
	}
	inCart := map[string]models.Cart{}
	if p.Key == "client" && p.User != nil {
		lines, err := s.Repo.GetCartByUser(p.User.ID)
		if err == nil {
			for _, line := range lines {
				inCart[line.ProductID] = line
			}
		}
	}
	for _, item := range products {
		c := s.cardOf(item)
		userID := ""
		if p.User != nil {
			userID = p.User.ID
		}
		c.Left = s.freeFor(item.ID, userID)
		if line, ok := inCart[item.ID]; ok {
			c.InCart = line.Count
			c.CartID = line.ID
		}
		p.Cards = append(p.Cards, c)
	}
	p.RangeText = rangeText(1, total, total)
	p.Return = r.URL.RequestURI()
	if p.Return == "" || p.Return == "/" {
		p.Return = "/catalog"
	}
	s.render(w, "catalog", p)
}

func (s *Pages) cardOf(item models.Product) card {
	c := card{
		ID: item.ID, Article: item.Article, Name: item.Name, Type: item.Type, Gender: item.Gender,
		Size: item.Size, Price: money(item.Price), Brand: item.Brand, Color: item.Color,
		Stock: item.Stock, Left: item.Stock,
	}
	photos, err := s.Repo.PhotosByProduct(item.ID)
	if err != nil {
		return c
	}
	for _, photo := range photos {
		if fileExists(photo.Filename) {
			c.HasFile = true
			c.PhotoID = photo.ID
			break
		}
	}
	return c
}

func (s *Pages) Product(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	item, err := s.Repo.FindProduct(mux.Vars(r)["id"])
	if err != nil {
		p.Title = "Товар не найден"
		p.Text = "Такой страницы нет."
		s.render(w, "message", p)
		return
	}
	p.Title = item.Name
	p.Product = item
	userID := ""
	if p.User != nil {
		userID = p.User.ID
	}
	p.Available = s.freeFor(item.ID, userID)
	if p.Key == "client" && p.User != nil {
		lines, err := s.Repo.GetCartByUser(p.User.ID)
		if err == nil {
			for _, line := range lines {
				if line.ProductID == item.ID {
					p.InCart = line.Count
					p.CartID = line.ID
					break
				}
			}
		}
	}
	photos, err := s.Repo.PhotosByProduct(item.ID)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось открыть товар", err)
		return
	}
	for _, photo := range photos {
		if !fileExists(photo.Filename) {
			continue
		}
		p.Photos = append(p.Photos, photoCard{ID: photo.ID, Filename: photo.Filename, HasFile: true})
	}
	p.HasFile = len(p.Photos) > 0
	s.render(w, "product", p)
}

func (s *Pages) Picture(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	photo, err := s.Repo.GetPhoto(mux.Vars(r)["id"])
	if err != nil {
		p.Title = "Фотография не найдена"
		p.Text = "Такой страницы нет."
		s.render(w, "message", p)
		return
	}
	p.Title = "Фотография"
	p.Photo = photo
	p.HasFile = fileExists(photo.Filename)
	if item, err := s.Repo.GetProductByID(photo.ProductID); err == nil {
		p.Product = item
		p.Title = item.Name
	}
	s.render(w, "photo", p)
}
