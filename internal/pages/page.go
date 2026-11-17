package pages

import (
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"YP01/internal/models"
)

type page struct {
	Title, Hint, Path, Flash, Error, Text string
	ActionHref, ActionLabel               string
	User                                  *models.User
	RoleName, Key, Next, Email            string
	CartCount                             int
	NavCatalog, NavOrders, NavCart        bool
	NavProducts, NavUsers                 bool

	Fields    []filterField
	Sort      string
	Cards     []card
	RangeText string
	Return    string

	Product   *models.Product
	Photos    []photoCard
	Available int
	InCart    int
	CartID    string
	Editing   bool
	HasFile   bool
	Photo     *models.Photo
	Lines     []cartLine
	Total     string
	Broken    bool
	Orders    []orderRow
	Staff     bool
	Status    string
	From, To  string
	Statuses  []string
	Groups    []itemGroup
	Sum       string
	Buyer     string
	When      string
	Order     *models.Order
	OrderRef  string
	Comment   string
	CanCancel bool
	Live      bool
	Users     []userRow
	Roles     []roleRow
	RoleID    string
}

type filterField struct {
	Key, Label, Current string
	Options             []string
}

type card struct {
	ID, Article, Name, Type, Gender, Size, Price, Brand, Color, PhotoID, CartID string
	HasFile                                                                     bool
	Stock, Left, InCart                                                         int
}

type photoCard struct {
	ID, Filename string
	HasFile      bool
}

type cartLine struct {
	ID, ProductID, Article, Name, Meta, Price, Sum, Color, Brand, PhotoID string
	Count                                                                 int
	Missing, HasFile                                                      bool
}

type orderRow struct {
	ID, When, Buyer, Sum, Status, Ref string
	Number                            int
	Next                              []string
}

type itemGroup struct {
	Name, ProductID, Price, Sum, Article, Description, Spec string
	PhotoID, Color, Brand                                   string
	Count                                                   int
	Missing, HasFile                                        bool
}

type userRow struct {
	ID, Name, Email, Phone, Role, Date string
}

type roleRow struct {
	ID, Name string
	Selected bool
}

func (s *Pages) base(w http.ResponseWriter, r *http.Request) page {
	u, role, key := s.current(r)
	p := page{
		Path: r.URL.Path,
		User: u,
		Key:  key,
	}
	if role != nil {
		p.RoleName = role.Name
	}
	if key == "client" && u != nil {
		lines, err := s.Repo.GetCartByUser(u.ID)
		if err == nil {
			for _, line := range lines {
				p.CartCount += line.Count
			}
		}
	}
	switch {
	case p.Path == "/" || p.Path == "/catalog":
		p.NavCatalog = true
	case p.Path == "/orders" || strings.HasPrefix(p.Path, "/orders/"):
		p.NavOrders = true
	case p.Path == "/cart":
		p.NavCart = true
	case strings.HasPrefix(p.Path, "/admin/products"):
		p.NavProducts = true
	case strings.HasPrefix(p.Path, "/admin/users"):
		p.NavUsers = true
	}
	p.Flash = takeFlash(w, r)
	return p
}

// После входа в куку пишется id пользователя. Нет куки — это гость.
func (s *Pages) current(r *http.Request) (*models.User, *models.Role, string) {
	c, err := r.Cookie("user_id")
	if err != nil || c.Value == "" {
		return nil, nil, "guest"
	}
	u, err := s.Repo.GetUser(c.Value)
	if err != nil {
		return nil, nil, "guest"
	}
	role, err := s.Repo.GetRole(u.RoleID)
	if err != nil {
		return u, nil, "custom"
	}
	return u, role, roleKey(role.Name)
}

func roleKey(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "client", "клиент":
		return "client"
	case "manager", "менеджер":
		return "manager"
	case "admin", "администратор":
		return "admin"
	default:
		return "custom"
	}
}

// Гость — это витрина без входа, а не учётка. В списке ролей она есть, в форму пользователя её не кладу.
func skipRole(name, id, current string) bool {
	return systemRole(name) && roleKey(name) == "custom" && id != current
}

func systemRole(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "guest", "client", "manager", "admin", "гость", "клиент", "менеджер", "администратор":
		return true
	default:
		return false
	}
}

func allowed(key string, keys ...string) bool {
	for _, item := range keys {
		if key == item {
			return true
		}
	}
	return false
}

func setUserCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user_id",
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearUserCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "user_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func setFlash(w http.ResponseWriter, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    url.QueryEscape(msg),
		Path:     "/",
		MaxAge:   30,
		HttpOnly: true,
	})
}

func takeFlash(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie("flash")
	if err != nil || c.Value == "" {
		return ""
	}
	http.SetCookie(w, &http.Cookie{Name: "flash", Value: "", Path: "/", MaxAge: -1})
	msg, err := url.QueryUnescape(c.Value)
	if err != nil {
		return c.Value
	}
	return msg
}

func goTo(w http.ResponseWriter, r *http.Request, path string) {
	http.Redirect(w, r, path, http.StatusSeeOther)
}

func safeNext(s string) string {
	if !strings.HasPrefix(s, "/") || strings.HasPrefix(s, "//") || strings.Contains(s, "\n") {
		return "/catalog"
	}
	return s
}

func rangeText(page, limit, total int) string {
	if total == 0 {
		return "Ничего не найдено"
	}
	from := (page-1)*limit + 1
	if from > total {
		return "На этой странице пусто"
	}
	to := page * limit
	if to > total {
		to = total
	}
	return fmt.Sprintf("Показано %d–%d из %d", from, to, total)
}

// Куда вернуть после «В корзину»: каталог или карточка товара, не чужой адрес.
func shopBack(path string) string {
	if strings.ContainsAny(path, "\r\n") || strings.HasPrefix(path, "//") || !strings.HasPrefix(path, "/") {
		return "/catalog"
	}
	u, err := url.Parse(path)
	if err != nil || u.Host != "" || u.Scheme != "" || strings.Contains(u.Path, "..") {
		return "/catalog"
	}
	if u.Path == "/" || u.Path == "/catalog" || strings.HasPrefix(u.Path, "/item/") {
		return path
	}
	return "/catalog"
}

func money(v float64) string {
	n := int(math.Round(v * 100))
	neg := ""
	if n < 0 {
		neg = "-"
		n = -n
	}
	rub := n / 100
	kop := n % 100
	s := strconv.Itoa(rub)
	var b strings.Builder
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(ch)
	}
	if kop == 0 {
		return neg + b.String() + " ₽"
	}
	return fmt.Sprintf("%s%s,%02d ₽", neg, b.String(), kop)
}

func formatDate(s string) string {
	if s == "" {
		return "—"
	}
	if len(s) >= 10 {
		parts := strings.Split(s[:10], "-")
		if len(parts) == 3 {
			out := parts[2] + "." + parts[1] + "." + parts[0]
			if len(s) >= 16 && (s[10] == 'T' || s[10] == ' ') {
				out += " " + s[11:16]
			}
			return out
		}
	}
	return s
}

func colorStyle(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	known := map[string]string{
		"чёрный": "#2a2a2a", "черный": "#2a2a2a", "белый": "#f4f1ea",
		"бежевый": "#d9c7ae", "синий": "#35597a", "серый": "#8e8a84",
		"хаки": "#6d7052", "коричневый": "#6b4a36", "зелёный": "#3e5c49",
		"зеленый": "#3e5c49", "красный": "#8d3b3b", "молочный": "#efe8dc",
	}
	if c, ok := known[key]; ok {
		return c
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return fmt.Sprintf("hsl(%d 32%% 42%%)", h.Sum32()%360)
}

func lightColor(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "белый", "бежевый", "молочный":
		return true
	default:
		return false
	}
}

func fileExists(name string) bool {
	if name == "" || strings.Contains(name, "..") {
		return false
	}
	st, err := os.Stat(filepath.Join("uploads", filepath.Base(name)))
	return err == nil && !st.IsDir()
}

func parsePrice(s string) (float64, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n < 0 || math.IsNaN(n) {
		return 0, false
	}
	return math.Round(n*100) / 100, true
}

// Почта вида имя@домен.зона. Одного знака @ мало: «а@б» и «а@б.в» не пропускаю.
func validEmail(s string) bool {
	if strings.Count(s, "@") != 1 || strings.Contains(s, "..") {
		return false
	}
	parts := strings.Split(s, "@")
	local, domain := parts[0], parts[1]
	if len(local) == 0 || len(domain) < 4 {
		return false
	}
	if local[0] == '.' || local[len(local)-1] == '.' {
		return false
	}
	for _, r := range local {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' || r == '+'
		if !ok {
			return false
		}
	}
	dot := strings.LastIndex(domain, ".")
	if dot < 1 || dot > len(domain)-3 {
		return false
	}
	for _, r := range domain {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-'
		if !ok {
			return false
		}
	}
	return true
}

// Российский номер: +7 или 8 и ещё 10 цифр. Пробелы, скобки и дефисы можно.
func validPhone(s string) bool {
	if s == "" || strings.Count(s, "+") > 1 {
		return false
	}
	if strings.Contains(s, "+") && !strings.HasPrefix(s, "+") {
		return false
	}
	digits := make([]byte, 0, 11)
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits = append(digits, byte(r))
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return false
		}
	}
	if len(digits) != 11 {
		return false
	}
	return digits[0] == '7' || digits[0] == '8'
}

func nowStamp() string {
	return time.Now().Format("2006-01-02T15:04:05")
}

func sortOptions(field string, values []string) []string {
	rank := map[string][]string{
		"size":   {"XS", "S", "M", "L", "XL", "XXL"},
		"season": {"лето", "демисезон", "зима", "всесезон"},
	}
	order, ok := rank[field]
	if !ok {
		return values
	}
	pos := map[string]int{}
	for i, name := range order {
		pos[name] = i
	}
	sorted := append([]string(nil), values...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			pi, oki := pos[sorted[i]]
			pj, okj := pos[sorted[j]]
			swap := false
			if oki && okj {
				swap = pi > pj
			} else if oki != okj {
				swap = !oki
			} else {
				swap = sorted[i] > sorted[j]
			}
			if swap {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

func endOfDay(s string) string {
	if len(s) == 10 {
		return s + "T23:59:59"
	}
	return s
}
