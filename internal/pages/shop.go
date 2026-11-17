package pages

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"YP01/internal/database"
	"YP01/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func (s *Pages) Cart(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "client" {
		p.Title = "Нет доступа"
		p.Text = "Корзина доступна авторизованному клиенту."
		s.render(w, "message", p)
		return
	}
	p.Title = "Корзина"
	p.Hint = "Сюда попадают товары текущего клиента."
	s.fillCart(&p)
	s.render(w, "cart", p)
}

func (s *Pages) fillCart(p *page) {
	lines, err := s.Repo.GetCartByUser(p.User.ID)
	if err != nil {
		p.Error = "Не получилось прочитать корзину"
		return
	}
	var total float64
	for _, line := range lines {
		row := cartLine{ID: line.ID, ProductID: line.ProductID, Count: line.Count}
		item, err := s.Repo.GetProductByID(line.ProductID)
		if err != nil {
			row.Missing = true
			row.Name = "Товар удалён"
			p.Broken = true
		} else {
			row.Name = item.Name
			row.Article = item.Article
			row.Color = item.Color
			row.Brand = item.Brand
			row.Meta = item.Size + " · " + item.Color + " · " + money(item.Price)
			photos, photoErr := s.Repo.PhotosByProduct(item.ID)
			if photoErr == nil {
				for _, photo := range photos {
					if fileExists(photo.Filename) {
						row.HasFile = true
						row.PhotoID = photo.ID
						break
					}
				}
			}
			row.Price = money(item.Price)
			var others int
			_ = s.Repo.DB.Get(&others, `SELECT COALESCE(SUM(count),0) FROM cart WHERE product_id = ? AND user_id != ?`, item.ID, p.User.ID)
			if line.Count > item.Stock-others {
				p.Broken = true
				row.Meta = item.Size + " · " + item.Color + " · на складе " + strconv.Itoa(item.Stock)
			}
			sum := item.Price * float64(line.Count)
			row.Sum = money(sum)
			total += sum
		}
		p.Lines = append(p.Lines, row)
	}
	p.Total = money(total)
}

func (s *Pages) CartAdd(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "client" {
		p.Title = "Нет доступа"
		p.Text = "Корзина доступна только клиентам."
		s.render(w, "message", p)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	productID := r.FormValue("product_id")
	back := shopBack(r.FormValue("back"))
	if _, err := s.Repo.GetProductByID(productID); err != nil {
		setFlash(w, "Товар не найден")
		goTo(w, r, back)
		return
	}
	count, _ := strconv.Atoi(r.FormValue("count"))
	if count < 1 {
		count = 1
	}
	left := s.freeFor(productID, p.User.ID)
	if left < 1 {
		setFlash(w, "Этого товара нет в наличии")
		goTo(w, r, back)
		return
	}
	if count > left {
		setFlash(w, "На складе осталось "+strconv.Itoa(left))
		goTo(w, r, back)
		return
	}
	lines, err := s.Repo.GetCartByUser(p.User.ID)
	if err != nil {
		setFlash(w, "Не получилось положить товар в корзину")
		goTo(w, r, back)
		return
	}
	for _, line := range lines {
		if line.ProductID == productID {
			line.Count += count
			_ = s.Repo.PutCart(&line)
			goTo(w, r, back)
			return
		}
	}
	err = s.Repo.CreateCart(&models.Cart{UserID: p.User.ID, ProductID: productID, Count: count})
	if err != nil {
		setFlash(w, "Не получилось положить товар в корзину")
	}
	goTo(w, r, back)
}

func (s *Pages) CartCount(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "client" {
		goTo(w, r, "/login")
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	back := "/cart"
	if r.FormValue("back") != "" {
		back = shopBack(r.FormValue("back"))
	}
	line, err := s.ownLine(p.User.ID, r.FormValue("id"))
	if err != nil {
		setFlash(w, "Позиция не найдена")
		goTo(w, r, back)
		return
	}
	if r.FormValue("op") == "dec" {
		if line.Count <= 1 {
			_ = s.Repo.DeleteCart(line.ID)
		} else {
			line.Count--
			_ = s.Repo.PutCart(line)
		}
	} else {
		if s.freeFor(line.ProductID, p.User.ID) < 1 {
			setFlash(w, "Больше нет на складе")
		} else {
			line.Count++
			_ = s.Repo.PutCart(line)
		}
	}
	goTo(w, r, back)
}

func (s *Pages) CartRemove(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "client" {
		goTo(w, r, "/login")
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	line, err := s.ownLine(p.User.ID, r.FormValue("id"))
	if err != nil {
		setFlash(w, "Позиция не найдена")
		goTo(w, r, "/cart")
		return
	}
	_ = s.Repo.DeleteCart(line.ID)
	goTo(w, r, "/cart")
}

func (s *Pages) ownLine(userID, id string) (*models.Cart, error) {
	line, err := s.Repo.GetCart(id)
	if err != nil {
		return nil, err
	}
	if line.UserID != userID {
		return nil, sql.ErrNoRows
	}
	return line, nil
}

func (s *Pages) Checkout(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "client" {
		p.Title = "Нет доступа"
		p.Text = "Оформлять заказ может только клиент."
		s.render(w, "message", p)
		return
	}
	lines, err := s.Repo.GetCartByUser(p.User.ID)
	if err != nil || len(lines) == 0 {
		setFlash(w, "Корзина пуста")
		goTo(w, r, "/cart")
		return
	}
	type piece struct {
		id, name string
		price    float64
		count    int
	}
	var pieces []piece
	for _, line := range lines {
		item, err := s.Repo.GetProductByID(line.ProductID)
		if err != nil {
			setFlash(w, "В корзине есть удалённый товар. Уберите его и оформите заказ снова.")
			goTo(w, r, "/cart")
			return
		}
		n := line.Count
		if n < 1 {
			n = 1
		}
		if n > item.Stock {
			setFlash(w, "«"+item.Name+"» на складе только "+strconv.Itoa(item.Stock))
			goTo(w, r, "/cart")
			return
		}
		pieces = append(pieces, piece{id: item.ID, name: item.Name, price: item.Price, count: n})
	}
	tx, err := s.Repo.DB.Beginx()
	if err != nil {
		setFlash(w, "Не получилось оформить заказ")
		goTo(w, r, "/cart")
		return
	}
	defer tx.Rollback()
	for _, piece := range pieces {
		res, err := tx.Exec(`UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`, piece.count, piece.id, piece.count)
		if err != nil {
			setFlash(w, "Не получилось оформить заказ")
			goTo(w, r, "/cart")
			return
		}
		got, _ := res.RowsAffected()
		if got != 1 {
			setFlash(w, "«"+piece.name+"» уже разобрали")
			goTo(w, r, "/cart")
			return
		}
	}
	orderID := uuid.NewString()
	var orderNo int
	if err = tx.Get(&orderNo, `SELECT COALESCE(MAX(number), 0) FROM orders`); err != nil {
		setFlash(w, "Не получилось оформить заказ")
		goTo(w, r, "/cart")
		return
	}
	if orderNo < 1000 {
		orderNo = 1000
	}
	orderNo++
	_, err = tx.Exec(
		`INSERT INTO orders (id, number, user_id, date_time, status, comment) VALUES (?, ?, ?, ?, ?, ?)`,
		orderID, orderNo, p.User.ID, nowStamp(), "Принят", "",
	)
	if err != nil {
		setFlash(w, "Не получилось оформить заказ")
		goTo(w, r, "/cart")
		return
	}
	for _, piece := range pieces {
		for i := 0; i < piece.count; i++ {
			_, err = tx.Exec(
				`INSERT INTO order_items (id, order_id, product_id, price) VALUES (?, ?, ?, ?)`,
				uuid.NewString(), orderID, piece.id, piece.price,
			)
			if err != nil {
				setFlash(w, "Не получилось записать состав заказа")
				goTo(w, r, "/cart")
				return
			}
		}
	}
	if _, err = tx.Exec(`DELETE FROM cart WHERE user_id = ?`, p.User.ID); err != nil {
		setFlash(w, "Не получилось оформить заказ")
		goTo(w, r, "/cart")
		return
	}
	if err = tx.Commit(); err != nil {
		setFlash(w, "Не получилось оформить заказ")
		goTo(w, r, "/cart")
		return
	}
	setFlash(w, "Заказ № "+strconv.Itoa(orderNo)+" оформлен")
	goTo(w, r, "/orders/"+strconv.Itoa(orderNo))
}

// Сколько ещё этот клиент может положить в корзину. Чужие корзины склад не занимают:
// число на карточке уменьшается только когда заказ уже оформлен.
func (s *Pages) freeFor(productID, userID string) int {
	item, err := s.Repo.GetProductByID(productID)
	if err != nil || item.Stock < 1 {
		return 0
	}
	var mine int
	if userID != "" {
		if err = s.Repo.DB.Get(&mine, `SELECT COALESCE(SUM(count),0) FROM cart WHERE product_id = ? AND user_id = ?`, productID, userID); err != nil {
			return 0
		}
	}
	left := item.Stock - mine
	if left < 0 {
		return 0
	}
	return left
}

func (s *Pages) Orders(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key == "guest" || p.Key == "custom" {
		p.Title = "Нет доступа"
		p.Text = "Историю заказов видит клиент. Все заказы и смену статуса — менеджер и администратор."
		s.render(w, "message", p)
		return
	}
	p.Staff = p.Key == "manager" || p.Key == "admin"
	if p.Staff {
		p.Title = "Заказы"
		p.Hint = ""
	} else {
		p.Title = "Мои заказы"
		p.Hint = "Только ваши заказы. Статус меняет магазин."
	}
	q := r.URL.Query()
	p.Status = q.Get("status")
	p.From = q.Get("from")
	p.To = q.Get("to")
	p.Statuses = []string{"Принят", "В работе", "Собран", "Готов к выдаче", "Отменён"}
	p.Live = true
	if p.From != "" && p.To != "" && p.From > p.To {
		p.Error = "Дата «с» позже даты «по»."
		s.render(w, "orders", p)
		return
	}
	f := database.OrderFilter{
		Page: 1, Status: p.Status, StartDate: p.From, EndDate: endOfDay(p.To),
	}
	if !p.Staff {
		f.UserID = p.User.ID
	}
	total, err := s.Repo.CountOrders(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать заказы", err)
		return
	}
	f.Limit = total
	if f.Limit < 1 {
		f.Limit = 1
	}
	orders, err := s.Repo.GetOrders(f)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось прочитать заказы", err)
		return
	}
	for _, order := range orders {
		row := orderRow{
			ID: order.ID, Number: order.Number, Ref: orderRef(order.Number, order.ID),
			When: formatDate(order.DateTime), Status: order.Status, Sum: money(s.orderSum(order.ID)),
		}
		if p.Staff {
			row.Next = staffStatuses()
			if buyer, err := s.Repo.GetUser(order.UserID); err == nil {
				row.Buyer = buyerLine(buyer.Name, buyer.Number)
			} else {
				row.Buyer = "Пользователь удалён"
			}
		}
		p.Orders = append(p.Orders, row)
	}
	p.RangeText = rangeText(1, total, total)
	s.render(w, "orders", p)
}

func (s *Pages) orderSum(id string) float64 {
	items, err := s.Repo.GetOrderItems(id)
	if err != nil {
		return 0
	}
	var sum float64
	for _, item := range items {
		sum += item.Price
	}
	return sum
}

func (s *Pages) fillGroup(g *itemGroup, item models.OrderItem) {
	g.Name = item.Name
	g.Description = item.Description
	g.Article = item.Article
	g.Spec = item.Spec
	prod, err := s.Repo.GetProductByID(item.ProductID)
	if err != nil {
		if g.Name == "" {
			g.Name = "Товар удалён"
		}
		g.Missing = true
		return
	}
	if g.Name == "" {
		g.Name = prod.Name
	}
	if g.Description == "" {
		g.Description = prod.Description
	}
	if g.Article == "" {
		g.Article = prod.Article
	}
	if g.Spec == "" && (prod.Size != "" || prod.Color != "") {
		g.Spec = prod.Size + " · " + prod.Color
	}
	g.Color = prod.Color
	g.Brand = prod.Brand
	photos, err := s.Repo.PhotosByProduct(prod.ID)
	if err != nil {
		return
	}
	for _, photo := range photos {
		if fileExists(photo.Filename) {
			g.HasFile = true
			g.PhotoID = photo.ID
			return
		}
	}
}

func (s *Pages) Order(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	order, err := s.Repo.FindOrder(mux.Vars(r)["id"])
	if err != nil {
		p.Title = "Заказ не найден"
		p.Text = "Такой страницы нет."
		s.render(w, "message", p)
		return
	}
	if p.Key == "guest" || p.Key == "custom" {
		p.Title = "Нет доступа"
		p.Text = "Заказ доступен участникам магазина."
		s.render(w, "message", p)
		return
	}
	p.Staff = p.Key == "manager" || p.Key == "admin"
	if !p.Staff && (p.User == nil || p.User.ID != order.UserID) {
		p.Title = "Нет доступа"
		p.Text = "Это заказ другого клиента."
		s.render(w, "message", p)
		return
	}
	p.Title = "Заказ"
	p.Order = order
	p.OrderRef = orderRef(order.Number, order.ID)
	p.Comment = order.Comment
	p.CanCancel = !p.Staff && order.Status != "Отменён" && order.Status != "Готов к выдаче"
	if p.CanCancel {
		p.Reasons = cancelReasons()
	}
	p.When = formatDate(order.DateTime)
	if p.Staff {
		p.Hint = "Здесь можно сменить статус."
		p.Statuses = staffStatuses()
	}
	if buyer, err := s.Repo.GetUser(order.UserID); err == nil {
		p.Buyer = buyerLine(buyer.Name, buyer.Number)
	} else {
		p.Buyer = "Пользователь удалён"
	}
	items, err := s.Repo.GetOrderItems(order.ID)
	if err != nil {
		s.fail(w, p, "Ошибка", "Не получилось открыть заказ", err)
		return
	}
	var sum float64
	for _, item := range items {
		sum += item.Price
		found := false
		for i := range p.Groups {
			if p.Groups[i].ProductID == item.ProductID && p.Groups[i].Price == money(item.Price) {
				p.Groups[i].Count++
				found = true
				break
			}
		}
		if found {
			continue
		}
		g := itemGroup{ProductID: item.ProductID, Price: money(item.Price), Count: 1}
		s.fillGroup(&g, item)
		p.Groups = append(p.Groups, g)
	}
	for i := range p.Groups {

		price := 0.0
		itemsSame := 0
		for _, item := range items {
			if item.ProductID == p.Groups[i].ProductID && money(item.Price) == p.Groups[i].Price {
				price = item.Price
				itemsSame++
			}
		}
		p.Groups[i].Sum = money(price * float64(itemsSame))
	}
	p.Sum = money(sum)
	p.Live = true
	s.render(w, "order", p)
}

func (s *Pages) OrderStatus(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "manager" && p.Key != "admin" {
		p.Title = "Нет доступа"
		p.Text = "Статус меняет менеджер или администратор."
		s.render(w, "message", p)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id := mux.Vars(r)["id"]
	order, err := s.Repo.FindOrder(id)
	if err != nil {
		setFlash(w, "Заказ не найден")
		goTo(w, r, "/orders")
		return
	}
	ref := orderRef(order.Number, order.ID)
	status := r.FormValue("status")
	if !statusOk(status) {
		setFlash(w, "Неизвестный статус")
		goTo(w, r, "/orders/"+ref)
		return
	}
	prev := order.Status
	if status == prev {
		goTo(w, r, "/orders/"+ref)
		return
	}
	if status == "Отменён" {
		if err = s.Repo.ReturnOrderStock(order.ID); err != nil {
			setFlash(w, "Не получилось вернуть товар на склад")
			goTo(w, r, "/orders/"+ref)
			return
		}
	}
	if prev == "Отменён" {
		ok, err := s.Repo.TakeOrderStock(order.ID)
		if err != nil || !ok {
			setFlash(w, "На складе уже не хватает товара, статус не изменён")
			goTo(w, r, "/orders/"+ref)
			return
		}
	}
	order.Status = status
	if err = s.Repo.PutOrder(order); err != nil {
		setFlash(w, "Не получилось сменить статус")
	} else {
		setFlash(w, "Статус: "+status)
	}
	back := r.FormValue("back")
	if !strings.HasPrefix(back, "/orders") {
		back = "/orders/" + ref
	}
	goTo(w, r, back)
}

func (s *Pages) OrderCancel(w http.ResponseWriter, r *http.Request) {
	p := s.base(w, r)
	if p.Key != "client" || p.User == nil {
		p.Title = "Нет доступа"
		p.Text = "Отменить заказ может только клиент, который его оформил."
		s.render(w, "message", p)
		return
	}
	id := mux.Vars(r)["id"]
	order, err := s.Repo.FindOrder(id)
	if err != nil || order.UserID != p.User.ID {
		setFlash(w, "Заказ не найден")
		goTo(w, r, "/orders")
		return
	}
	ref := orderRef(order.Number, order.ID)
	if order.Status == "Отменён" || order.Status == "Готов к выдаче" {
		setFlash(w, "Этот заказ уже нельзя отменить")
		goTo(w, r, "/orders/"+ref)
		return
	}
	if err = r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	comment, ok := cancelComment(r.FormValue("reason"), r.FormValue("comment"))
	if !ok {
		setFlash(w, comment)
		goTo(w, r, "/orders/"+ref)
		return
	}
	if err = s.Repo.ReturnOrderStock(order.ID); err != nil {
		setFlash(w, "Не получилось вернуть товар на склад")
		goTo(w, r, "/orders/"+ref)
		return
	}
	order.Status = "Отменён"
	order.Comment = comment
	if err = s.Repo.PutOrder(order); err != nil {
		setFlash(w, "Не получилось отменить заказ")
		goTo(w, r, "/orders/"+ref)
		return
	}
	setFlash(w, "Заказ отменён")
	goTo(w, r, "/orders/"+ref)
}

func buyerLine(name string, number int) string {
	if number > 0 {
		return name + " (" + strconv.Itoa(number) + ")"
	}
	return name
}

func orderRef(number int, id string) string {
	if number > 0 {
		return strconv.Itoa(number)
	}
	return id
}

func cancelReasons() []string {
	return []string{
		"Передумал покупать",
		"Ошибся с размером или цветом",
		"Заказал по ошибке",
		"Слишком долго ждать",
		"Другое",
	}
}

// Готовую причину пишем как есть. «Другое» требует свой текст.
func cancelComment(reason, own string) (string, bool) {
	reason = strings.TrimSpace(reason)
	if reason == "Другое" {
		own = strings.TrimSpace(own)
		if own == "" {
			return "Напишите свою причину", false
		}
		runes := []rune(own)
		if len(runes) > 500 {
			own = string(runes[:500])
		}
		return own, true
	}
	for _, item := range cancelReasons() {
		if item == reason {
			return reason, true
		}
	}
	return "Выберите причину отмены", false
}

func staffStatuses() []string {
	return []string{"Принят", "В работе", "Собран", "Готов к выдаче", "Отменён"}
}

func statusOk(s string) bool {
	list := []string{"Принят", "В работе", "Собран", "Готов к выдаче", "Отменён"}
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
