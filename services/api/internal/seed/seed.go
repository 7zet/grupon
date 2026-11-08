package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grupon/grupon/services/api/internal/domain/ledger"
	"github.com/grupon/grupon/services/api/internal/domain/merchant"
	"github.com/grupon/grupon/services/api/internal/store"
	sqlc "github.com/grupon/grupon/services/api/internal/store/sqlc"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type localized map[string]string

type seedData struct {
	Cities []struct {
		Slug   string    `json:"slug"`
		Name   localized `json:"name"`
		Lat    float64   `json:"lat"`
		Lng    float64   `json:"lng"`
		Active *bool     `json:"active"`
	} `json:"cities"`
	Categories []struct {
		Slug          string    `json:"slug"`
		Name          localized `json:"name"`
		Icon          string    `json:"icon"`
		Subcategories []string  `json:"subcategories"`
		Active        *bool     `json:"active"`
	} `json:"categories"`
	MerchantNames []string          `json:"merchantNames"`
	ReviewTexts   map[string]string `json:"reviewTexts"`
	Catalog       catalogData       `json:"-"`
}

// catalogData is the curated demo content in seed/catalog.json: realistic businesses and deal templates.
type catalogData struct {
	MerchantDescriptions map[string]localized `json:"merchantDescriptions"`
	Merchants            []struct {
		Name      string   `json:"name"`
		Category  string   `json:"category"`
		Addresses []string `json:"addresses"`
		Offers    []int    `json:"offers"`
	} `json:"merchants"`
	Deals map[string][]dealTemplate `json:"deals"`
}

type dealTemplate struct {
	Title       localized   `json:"title"`
	Short       localized   `json:"short"`
	Description localized   `json:"description"`
	Highlights  []localized `json:"highlights"`
	Included    []localized `json:"included"`
	Options     []struct {
		Name     localized `json:"name"`
		Original int64     `json:"original"`
		Percent  int64     `json:"percent"`
	} `json:"options"`
	Subcategory int   `json:"subcategory"`
	Images      []int `json:"images"`
}

type imageSeed struct {
	URL string `json:"url"`
}

type dealSeed struct {
	ID         string
	OptionID   string
	MerchantID string
	LocationID string
	Price      int64
	Title      []byte
}

// Run applies migrations and loads the deterministic development dataset in one transaction.
// It is rerunnable: every seeded table is truncated before the load.
func Run(ctx context.Context, databaseURL string) error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	if err := migrate(ctx, databaseURL, filepath.Join(root, "services", "api", "migrations")); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	data, images, err := loadSeed(root)
	if err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := populate(ctx, tx, data, images); err != nil {
		return fmt.Errorf("populate: %w", err)
	}
	return tx.Commit(ctx)
}

func migrate(ctx context.Context, databaseURL, migrations string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, db, migrations)
}

func repositoryRoot() (string, error) {
	if root := os.Getenv("SEED_ROOT"); root != "" {
		return root, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repository root not found")
		}
		dir = parent
	}
}

func loadSeed(root string) (seedData, []imageSeed, error) {
	var data seedData
	var images []imageSeed
	raw, err := os.ReadFile(filepath.Join(root, "seed", "data.json"))
	if err != nil {
		return data, nil, err
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, nil, err
	}
	raw, err = os.ReadFile(filepath.Join(root, "seed", "images.json"))
	if err != nil {
		return data, nil, err
	}
	if err := json.Unmarshal(raw, &images); err != nil {
		return data, nil, err
	}
	raw, err = os.ReadFile(filepath.Join(root, "seed", "catalog.json"))
	if err != nil {
		return data, nil, err
	}
	if err := json.Unmarshal(raw, &data.Catalog); err != nil {
		return data, nil, err
	}
	return data, images, nil
}

func populate(ctx context.Context, tx pgx.Tx, data seedData, images []imageSeed) error {
	if len(data.Cities) != 5 || len(data.Categories) != 11 || len(images) == 0 {
		return errors.New("seed source counts are invalid")
	}
	// Accounts people registered themselves are kept on every reseed: they are copied aside, the catalog and
	// transactions are reset, and the accounts are written back. Demo accounts are replaced by the seed.
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE keep_users ON COMMIT DROP AS SELECT * FROM users WHERE email NOT LIKE '%@local.test' AND email NOT LIKE '%@test.local'`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE keep_roles ON COMMIT DROP AS SELECT * FROM role_assignments WHERE merchant_id IS NULL AND user_id IN (SELECT id FROM keep_users)`); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `TRUNCATE users, merchants, cities, categories, homepage_sections, journal_lines, journal_entries, ledger_accounts, redemptions, payout_runs, refunds, disputes, merchant_payout_holds, fraud_signals, outbox, reconciliation_findings CASCADE`)
	if err != nil {
		return err
	}
	// Write the kept accounts back. Their orders were reset with the transactions, the logins are unchanged.
	if _, err := tx.Exec(ctx, `INSERT INTO users SELECT * FROM keep_users`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO role_assignments SELECT * FROM keep_roles`); err != nil {
		return err
	}
	cityIDs := make([]string, len(data.Cities))
	for i, city := range data.Cities {
		name, _ := json.Marshal(city.Name)
		if err := tx.QueryRow(ctx, `INSERT INTO cities(slug,name,lat,lng,sort_order,is_active) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, city.Slug, name, city.Lat, city.Lng, i, active(city.Active)).Scan(&cityIDs[i]); err != nil {
			return err
		}
	}
	categoryIDs := make([]string, len(data.Categories))
	subcategoryIDs := make([][]string, len(data.Categories))
	for i, category := range data.Categories {
		name, _ := json.Marshal(category.Name)
		if err := tx.QueryRow(ctx, `INSERT INTO categories(slug,name,icon,sort_order,searchable,deal_creation_enabled,publish_enabled,purchasable,redeemable) VALUES($1,$2,$3,$4,$5,$5,$5,$5,true) RETURNING id`, category.Slug, name, category.Icon, i, active(category.Active)).Scan(&categoryIDs[i]); err != nil {
			return err
		}
		for j, slug := range category.Subcategories {
			title := strings.ReplaceAll(slug, "-", " ")
			translated, _ := json.Marshal(localized{"uz": title, "ru": title, "en": title})
			var id string
			if err := tx.QueryRow(ctx, `INSERT INTO subcategories(category_id,slug,name,sort_order) VALUES($1,$2,$3,$4) RETURNING id`, categoryIDs[i], category.Slug+"-"+slug, translated, j).Scan(&id); err != nil {
				return err
			}
			subcategoryIDs[i] = append(subcategoryIDs[i], id)
		}
	}
	userIDs := make([]string, 24)
	emails := []string{"user@local.test", "owner@local.test", "cashier@local.test", "admin@local.test", "owner-b@local.test", "owner-c@local.test", "owner-d@local.test", "admin-b@local.test", "cashier-b@local.test", "buyer-b@local.test", "manager@local.test", "moderator@local.test", "support@local.test", "finance-a@local.test", "finance-b@local.test"}
	demoNames := []string{
		"Aziz — xaridor", "Dilnoza — Lotus Spa egasi", "Bekzod — Lotus Spa kassiri (Chilonzor)",
		"Saida — superadmin", "Jasur — Urban Grill egasi", "Malika — Family Cafe egasi",
		"Sardor — Beauty House egasi", "Kamron — admin", "Nodira — Urban Grill kassiri",
		"Zarina — xaridor", "Sherzod — Lotus Spa menejeri", "Madina — moderator",
		"Umida — qo'llab-quvvatlash", "Farhod — moliya tayyorlovchi", "Shahnoza — moliya tasdiqlovchi",
	}
	for i := range userIDs {
		email := fmt.Sprintf("customer%02d@local.test", i)
		name := fmt.Sprintf("Seed Customer %02d", i)
		if i < len(emails) {
			email, name = emails[i], demoNames[i]
		}
		if err := tx.QueryRow(ctx, `INSERT INTO users(email,name,locale,currency,email_verified_at,phone) VALUES($1,$2,$3,'UZS',now(),$4) RETURNING id`, email, name, []string{"uz", "ru", "en"}[i%3], seedPhone(i)).Scan(&userIDs[i]); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO role_assignments(user_id,role) VALUES($1,'user')`, userIDs[i]); err != nil {
			return err
		}
	}
	if len(data.Catalog.Merchants) != 30 {
		return errors.New("seed catalog must hold 30 merchants")
	}
	merchantIDs := make([]string, 30)
	locationIDs := make([][]string, 30)
	hours := `{"mon":["09:00","20:00"],"tue":["09:00","20:00"],"wed":["09:00","20:00"],"thu":["09:00","20:00"],"fri":["09:00","20:00"],"sat":["10:00","18:00"],"sun":null}`
	for i := range merchantIDs {
		entry := data.Catalog.Merchants[i]
		name := entry.Name
		description, _ := json.Marshal(data.Catalog.MerchantDescriptions[entry.Category])
		slug := fmt.Sprintf("seed-merchant-%02d", i+1)
		if err := tx.QueryRow(ctx, `INSERT INTO merchants(slug,legal_name,display_name,description,phone,website,commission_bps,review_status) VALUES($1,$2,$2,$3,$4,$5,1500,'approved') RETURNING id`, slug, name, description, fmt.Sprintf("+99890100%04d", i), "https://local.test/business/"+slug).Scan(&merchantIDs[i]); err != nil {
			return err
		}
		demoProof := merchant.DemoDocument()
		if _, err := tx.Exec(ctx, `INSERT INTO merchant_documents(merchant_id,kind,storage_key,content_type,size_bytes,uploaded_by) VALUES($1,'price_proof',$2,'application/pdf',$3,$4)`, merchantIDs[i], merchant.DemoDocumentKey(slug), len(demoProof), userIDs[1]); err != nil {
			return err
		}
		for j, address := range entry.Addresses {
			var id string
			city := 0 // Toshkent: the only open city in the pilot (cities are listed in seed/data.json)
			if err := tx.QueryRow(ctx, `INSERT INTO merchant_locations(merchant_id,city_id,address,lat,lng,opening_hours,phone) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, merchantIDs[i], cityIDs[city], address+", Toshkent", data.Cities[city].Lat+float64(i%7)/500+float64(j)/1000, data.Cities[city].Lng+float64(i%5)/500+float64(j)/1000, hours, fmt.Sprintf("+99890100%04d", i)).Scan(&id); err != nil {
				return err
			}
			locationIDs[i] = append(locationIDs[i], id)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO role_assignments(user_id,role,merchant_id) VALUES($1,'merchant_owner',$3),($2,'merchant_cashier',$3),($4,'merchant_manager',$3)`, userIDs[1], userIDs[2], merchantIDs[0], userIDs[10]); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO role_assignments(user_id,role) VALUES($1,'superadmin')`, userIDs[3]); err != nil {
		return err
	}
	// Parallel e2e specs each sign in as their own owner, admin and cashier. Owner B has merchant 2; owner C and D
	// have merchants 3 and 4. Cashier B works for merchant 2.
	if _, err := tx.Exec(ctx, `INSERT INTO role_assignments(user_id,role,merchant_id) VALUES($1,'merchant_owner',$2),($3,'merchant_owner',$4),($5,'merchant_owner',$6),($7,'merchant_cashier',$2)`, userIDs[4], merchantIDs[1], userIDs[5], merchantIDs[2], userIDs[6], merchantIDs[3], userIDs[8]); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO role_assignments(user_id,role) VALUES($1,'superadmin')`, userIDs[7]); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO role_assignments(user_id,role) VALUES($1,'moderator'),($2,'support'),($3,'finance'),($4,'finance')`, userIDs[11], userIDs[12], userIDs[13], userIDs[14]); err != nil {
		return err
	}
	// Development-only deterministic factors let the seeded platform admins exercise the mandatory
	// 2FA path in browser tests. Production never runs this seed command.
	if _, err := tx.Exec(ctx, `INSERT INTO user_two_factor(user_id,secret,enabled_at) VALUES($1,$3,now()),($2,$3,now())`, userIDs[3], userIDs[7], "JBSWY3DPEHPK3PXP"); err != nil {
		return err
	}
	statuses := []string{"active", "active", "active", "active", "active", "active", "scheduled", "paused", "ended", "draft"}
	// One deal per (business, offer) pair, so no business repeats a deal and every deal matches its business.
	// Categories take turns so that every page of the catalog mixes them.
	dealCategories := []string{"beauty-and-spas", "food-and-drink", "things-to-do", "health-and-fitness"}
	categoryIndex := map[string]int{}
	for i, category := range data.Categories {
		categoryIndex[category.Slug] = i
	}
	type planned struct{ merchant, template int }
	queues := map[string][]planned{}
	for round := 0; round < 8; round++ {
		for i, entry := range data.Catalog.Merchants {
			if round < len(entry.Offers) {
				queues[entry.Category] = append(queues[entry.Category], planned{i, entry.Offers[round]})
			}
		}
	}
	type dealPlan struct {
		category string
		planned
	}
	var plan []dealPlan
	for remaining := true; remaining; {
		remaining = false
		for _, slug := range dealCategories {
			if len(queues[slug]) > 0 {
				plan = append(plan, dealPlan{slug, queues[slug][0]})
				queues[slug] = queues[slug][1:]
				remaining = true
			}
		}
	}
	deals := make([]dealSeed, len(plan))
	terms, _ := json.Marshal(localized{"uz": "Kupon 30 kun amal qiladi. Tashrifdan oldin telefon orqali yoziling. Boshqa chegirmalar bilan qo'shilmaydi.", "ru": "Купон действует 30 дней. Перед визитом запишитесь по телефону. Не суммируется с другими скидками.", "en": "The voucher is valid for 30 days. Book by phone before your visit. It cannot be combined with other offers."})
	instructions, _ := json.Marshal(localized{"uz": "Kassirga QR-kodni yoki kupon kodini ko'rsating.", "ru": "Покажите кассиру QR-код или код купона.", "en": "Show the QR code or voucher code to the cashier."})
	faq, _ := json.Marshal([]map[string]localized{
		{"q": {"uz": "Kuponni qanday ishlataman?", "ru": "Как использовать купон?", "en": "How do I use the voucher?"}, "a": {"uz": "Joyda QR-kodni ko'rsating.", "ru": "Покажите QR-код на месте.", "en": "Show the QR code on site."}},
		{"q": {"uz": "Pulni qaytarish mumkinmi?", "ru": "Можно ли вернуть деньги?", "en": "Can I get a refund?"}, "a": {"uz": "Ishlatilmagan kupon 14 kun ichida qaytariladi.", "ru": "Неиспользованный купон можно вернуть в течение 14 дней.", "en": "An unused voucher is refundable within 14 days."}},
	})
	for i := range deals {
		slug := plan[i].category
		templates := data.Catalog.Deals[slug]
		if plan[i].template >= len(templates) {
			return fmt.Errorf("seed catalog has no deal template %d for %s", plan[i].template, slug)
		}
		merchantIndex := plan[i].merchant
		template := templates[plan[i].template]
		category := categoryIndex[slug]
		title, _ := json.Marshal(template.Title)
		short, _ := json.Marshal(template.Short)
		detail, _ := json.Marshal(template.Description)
		highlights, _ := json.Marshal(template.Highlights)
		included, _ := json.Marshal(template.Included)
		status := statuses[i%len(statuses)]
		if i%20 == 9 {
			status = "in_review" // a few deals wait in the moderation queue for the demo
		}
		start := time.Now().UTC().Add(-24 * time.Hour)
		end := start.Add(time.Duration(10+i%20) * 24 * time.Hour)
		if status == "scheduled" {
			start = time.Now().UTC().Add(48 * time.Hour)
			end = start.Add(10 * 24 * time.Hour)
		}
		if status == "ended" {
			start = time.Now().UTC().Add(-20 * 24 * time.Hour)
			end = time.Now().UTC().Add(-24 * time.Hour)
		}
		if err := tx.QueryRow(ctx, `INSERT INTO deals(slug,merchant_id,category_id,subcategory_id,title,short_description,description,highlights,whats_included,terms,faq,redemption_instructions,booking_required,status,starts_at,ends_at,voucher_valid_days,max_per_user,sold_count) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,30,5,$17) RETURNING id`, fmt.Sprintf("seed-deal-%03d", i+1), merchantIDs[merchantIndex], categoryIDs[category], subcategoryIDs[category][template.Subcategory%len(subcategoryIDs[category])], title, short, detail, highlights, included, terms, faq, instructions, i%4 == 0, status, start, end, i*3).Scan(&deals[i].ID); err != nil {
			return err
		}
		deals[i].MerchantID, deals[i].LocationID, deals[i].Title = merchantIDs[merchantIndex], locationIDs[merchantIndex][0], title
		for _, locationID := range locationIDs[merchantIndex] {
			if _, err := tx.Exec(ctx, `INSERT INTO deal_locations(deal_id,merchant_location_id) VALUES($1,$2)`, deals[i].ID, locationID); err != nil {
				return err
			}
		}
		for j, option := range template.Options {
			original := option.Original * tiyinPerSom
			// Deal prices are rounded to 500 so'm so they read like real prices.
			price := (original*(100-option.Percent)/100 + 25000) / 50000 * 50000
			optionTitle, _ := json.Marshal(option.Name)
			var optionID string
			if err := tx.QueryRow(ctx, `INSERT INTO deal_options(deal_id,title,original_price,deal_price,stock_total,stock_sold,sort_order) VALUES($1,$2,$3,$4,500,$5,$6) RETURNING id`, deals[i].ID, optionTitle, original, price, i*3, j).Scan(&optionID); err != nil {
				return err
			}
			if j == 0 {
				deals[i].OptionID, deals[i].Price = optionID, price
			}
		}
		// Photos come only from the template's own category-matched list; each photo appears once per deal.
		photos := min(len(template.Images), 3)
		for j := 0; j < photos; j++ {
			image := images[template.Images[(i+j)%len(template.Images)]%len(images)]
			if _, err := tx.Exec(ctx, `INSERT INTO deal_media(deal_id,url,alt,sort_order,width,height,blurhash) VALUES($1,$2,$3,$4,1200,900,'L6PZfSi_.AyE_3t7t7R**0o#DgR4')`, deals[i].ID, image.URL+"?auto=format&fit=crop&w=1200&q=80", title, j); err != nil {
				return err
			}
		}
	}
	for i := 0; i < 400; i++ {
		deal := deals[i%len(deals)]
		userID := userIDs[4+i%(len(userIDs)-4)]
		var orderID, itemID, voucherID string
		if err := tx.QueryRow(ctx, `INSERT INTO orders(number,user_id,status,subtotal,total,idempotency_key,paid_at,created_at) VALUES('GR-2026-' || lpad(nextval('order_number_seq')::text, 6, '0'),$1,'paid',$2,$2,$3,now()-make_interval(days => $4::int),now()-make_interval(days => $4::int)) RETURNING id`, userID, deal.Price, fmt.Sprintf("seed-order-%06d", i+1), 1+(i*7)%27).Scan(&orderID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO order_items(order_id,deal_id,deal_option_id,title_snapshot,unit_price_snapshot,original_price_snapshot,terms_snapshot,quantity) SELECT $1,$2,$3,$4,$5,o.original_price,d.terms,1 FROM deal_options o JOIN deals d ON d.id=o.deal_id WHERE o.id=$3 RETURNING id`, orderID, deal.ID, deal.OptionID, deal.Title, deal.Price).Scan(&itemID); err != nil {
			return err
		}
		code := fmt.Sprintf("%04X-%04X", i+1, 65535-i)
		if err := tx.QueryRow(ctx, `INSERT INTO vouchers(order_item_id,user_id,deal_id,deal_option_id,merchant_id,code,qr_token,status,valid_from,valid_until,redeemed_at,redeemed_by_user_id,redeemed_location_id) VALUES($1,$2,$3,$4,$5,$6,$7,'redeemed',now()-interval '10 days',now()+interval '20 days',now()-interval '1 day',$8,$9) RETURNING id`, itemID, userID, deal.ID, deal.OptionID, deal.MerchantID, code, "seed-signed-token-"+code, userIDs[2], deal.LocationID).Scan(&voucherID); err != nil {
			return err
		}
		// Every seeded sale is also in the ledger: payment received, then the one redemption.
		if _, err := tx.Exec(ctx, `UPDATE vouchers SET redemptions_used = 1 WHERE id = $1`, voucherID); err != nil {
			return err
		}
		// The demo cashier scanned the Lotus Spa vouchers, so their history is not empty.
		staffID := userIDs[1]
		if deal.MerchantID == merchantIDs[0] {
			staffID = userIDs[2]
		}
		var redemptionID string
		if err := tx.QueryRow(ctx, `INSERT INTO redemptions(voucher_id,merchant_id,merchant_location_id,staff_user_id,sequence,amount,commission) VALUES($1,$2,$3,$4,1,$5,$6) RETURNING id`, voucherID, deal.MerchantID, deal.LocationID, staffID, deal.Price, ledger.Commission(deal.Price, 1500)).Scan(&redemptionID); err != nil {
			return err
		}
		queries := sqlc.New(tx)
		if _, err := store.PostEntry(ctx, queries, ledger.PaymentReceived(itemID, deal.MerchantID, deal.Price)); err != nil {
			return err
		}
		if _, err := store.PostEntry(ctx, queries, ledger.Redeemed(redemptionID, deal.MerchantID, deal.Price, ledger.Commission(deal.Price, 1500))); err != nil {
			return err
		}
		locale := []string{"uz", "ru", "en"}[i%3]
		var reviewID string
		if err := tx.QueryRow(ctx, `INSERT INTO reviews(deal_id,merchant_id,user_id,voucher_id,rating,text,locale,status) VALUES($1,$2,$3,$4,$5,$6,$7,'published') RETURNING id`, deal.ID, deal.MerchantID, userID, voucherID, 3+i%3, reviewVariants[locale][(i/3)%len(reviewVariants[locale])], locale).Scan(&reviewID); err != nil {
			return err
		}
		if i%5 == 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO review_replies(review_id,merchant_user_id,text) VALUES($1,$2,$3)`, reviewID, userIDs[1], reviewReplies[locale]); err != nil {
				return err
			}
		}
	}
	// Each homepage section lists its own eight active deals, so no deal repeats between sections.
	var active []string
	for i := range deals {
		if statuses[i%len(statuses)] == "active" {
			active = append(active, deals[i].ID)
		}
	}
	sections := []struct {
		key   string
		title localized
		from  int
	}{
		{"featured", localized{"uz": "Tanlangan takliflar", "ru": "Рекомендуем", "en": "Featured deals"}, 0},
		{"popular", localized{"uz": "Yaqin atrofda mashhur", "ru": "Популярное рядом", "en": "Popular near you"}, 8},
	}
	for order, section := range sections {
		title, _ := json.Marshal(section.title)
		if _, err := tx.Exec(ctx, `INSERT INTO homepage_sections(key,title,type,deal_ids,sort_order) VALUES($1,$2,'deals',$3,$4)`, section.key, title, active[section.from:section.from+8], order+1); err != nil {
			return err
		}
	}
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// reviewVariants are short buyer reviews per language; seeded reviews cycle through them.
var reviewVariants = map[string][]string{
	"uz": {
		"Xizmat juda yoqdi, xodimlar e'tiborli va hammasi kelishilganidek bo'ldi.",
		"Narxiga arziydi. Kuponni ko'rsatdim, hech qanday muammo bo'lmadi.",
		"Joy toza va qulay, yana kelaman.",
		"Yozilish oson bo'ldi, kutib o'tirmadim.",
		"Tavsifdagidek, lekin band vaqt tanlash biroz qiyin bo'ldi.",
		"Do'stlarimga ham tavsiya qildim.",
	},
	"ru": {
		"Очень понравилось: персонал внимательный, всё прошло как договаривались.",
		"Стоит своих денег. Показал купон, никаких проблем.",
		"Чисто и уютно, обязательно приду ещё.",
		"Записаться было легко, ждать не пришлось.",
		"Всё как в описании, но выбрать удобное время было непросто.",
		"Порекомендовал друзьям.",
	},
	"en": {
		"Really enjoyed it. Friendly staff and everything went as agreed.",
		"Worth the price. I showed the voucher and there were no problems.",
		"Clean and comfortable, I will come back.",
		"Booking was easy and I did not have to wait.",
		"As described, though finding a convenient time slot took a while.",
		"I recommended it to my friends.",
	},
}

// reviewReplies are short merchant replies in the language of the review.
var reviewReplies = map[string]string{
	"uz": "Fikringiz uchun rahmat, sizni yana kutamiz!",
	"ru": "Спасибо за отзыв, будем рады видеть вас снова!",
	"en": "Thank you for the kind words, we hope to see you again!",
}

// tiyinPerSom converts so'm amounts in seed data to minor units. Money is stored in tiyin (ADR-0003).
const tiyinPerSom = 100

// active reads an optional seed flag; a missing flag means the item is open.
func active(flag *bool) bool {
	return flag == nil || *flag
}

// seedPhone is the development sign-in phone of the seeded user at index i. The first four are the
// named test accounts; the rest get numbers from a reserved development range.
func seedPhone(i int) string {
	named := []string{"+998900000001", "+998900000002", "+998900000003", "+998900000004", "+998900000005", "+998900000006", "+998900000007", "+998900000008", "+998900000009", "+998910000009", "+998910000010", "+998910000011", "+998910000012", "+998910000013", "+998910000014"}
	if i < len(named) {
		return named[i]
	}
	return fmt.Sprintf("+99891%07d", i)
}
