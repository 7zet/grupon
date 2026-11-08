package catalog

import "strings"

// searchGroups lists the same idea in Uzbek (Latin and Cyrillic), Russian and English. Deal titles are
// stored per language, so a buyer who types "массаж" must also reach the Uzbek title "Massaj" and the
// English "Massage". A simple table is enough for the main subcategories.
var searchGroups = [][]string{
	{"masaj", "massaj", "массаж", "масаж", "massage"},
	{"yoga", "йога", "yoga"},
	{"basseyn", "basseynlar", "бассейны", "pools", "бассейн", "pool", "swimming", "suzish", "плавание", "сузиш"},
	{"sartarosh", "soch olish", "стрижка", "haircut", "sartaroshxonalar", "барбершопы", "barbershops", "барбер", "парикмахер", "barber", "barbershop", "сартарош"},
	{"salon", "salonlari", "салоны", "salons", "салон", "красот", "go'zallik", "gozallik", "beauty", "гўзаллик"},
	{"spa", "спа", "hammom", "хаммам", "баня", "sauna", "сауна"},
	{"kvest", "kvestlar", "квесты", "escape rooms", "квест", "quest", "escape"},
	{"bouling", "боулинг", "bowling", "боулинг"},
	{"karting", "картинг", "kart"},
	{"fitnes", "zallari", "спортзалы", "gyms", "фитнес", "gym", "sport zal", "спортзал", "тренажер", "fitness"},
	{"restoran", "restoranlar", "рестораны", "restaurants", "ресторан", "restaurant", "ресторан"},
	{"kafe", "kafelar", "cafes", "кафе", "cafe", "qahva", "кофе", "coffee", "қаҳва"},
	{"manikyur", "маникюр", "nail", "manicure", "маникур"},
	{"plov", "плов", "pilaf", "палов"},
	{"sushi", "суши"},
	{"pitsa", "пицца", "pizza", "питса"},
	{"akvapark", "аквапарк", "aqua park", "water park"},
	{"planetariy", "планетарий", "planetarium"},
	{"ekskursiya", "экскурсия", "excursion", "tour", "тур", "walk"},
}

// ExpandSearch returns the query with the equivalent words of every matching group appended as
// alternatives ("a or b or c"), in the syntax of websearch_to_tsquery. A query that matches no group is
// returned unchanged.
func ExpandSearch(query string) string {
	lowered := strings.ToLower(strings.TrimSpace(query))
	if lowered == "" {
		return query
	}
	seen := map[string]bool{lowered: true}
	out := []string{query}
	for _, group := range searchGroups {
		matched := false
		for _, term := range group {
			if len([]rune(term)) >= 3 && strings.Contains(lowered, term) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		for _, term := range group {
			if !seen[term] {
				seen[term] = true
				out = append(out, term)
			}
		}
	}
	return strings.Join(out, " or ")
}
