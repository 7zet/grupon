package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grupon/grupon/services/api/internal/domain/ledger"
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
		if err := tx.QueryRow(ctx, `INSERT INTO categories(slug,name,icon,sort_order,is_active) VALUES($1,$2,$3,$4,$5) RETURNING id`, category.Slug, name, category.Icon, i, active(category.Active)).Scan(&categoryIDs[i]); err != nil {
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
	emails := []string{"user@local.test", "owner@local.test", "cashier@local.test", "admin@local.test", "owner-b@local.test", "owner-c@local.test", "owner-d@local.test", "admin-b@local.test", "cashier-b@local.test"}
	for i := range userIDs {
		email := fmt.Sprintf("customer%02d@local.test", i)
		name := fmt.Sprintf("Seed Customer %02d", i)
		if i < len(emails) {
			email, name = emails[i], []string{"Test User", "Merchant Owner", "Merchant Cashier", "Superadmin", "Merchant Owner B", "Merchant Owner C", "Merchant Owner D", "Superadmin B", "Merchant Cashier B"}[i]
		}
		if err := tx.QueryRow(ctx, `INSERT INTO users(email,name,locale,currency,email_verified_at,phone) VALUES($1,$2,$3,'UZS',now(),$4) RETURNING id`, email, name, []string{"uz", "ru", "en"}[i%3], seedPhone(i)).Scan(&userIDs[i]); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO role_assignments(user_id,role) VALUES($1,'user')`, userIDs[i]); err != nil {
			return err
		}
	}
	merchantIDs := make([]string, 30)
	locationIDs := make([][]string, 30)
	description, _ := json.Marshal(localized{"uz": "Mahalliy ishonchli biznes", "ru": "Надёжный местный бизнес", "en": "A trusted local business"})
	hours := `{"mon":["09:00","20:00"],"tue":["09:00","20:00"],"wed":["09:00","20:00"],"thu":["09:00","20:00"],"fri":["09:00","20:00"],"sat":["10:00","18:00"],"sun":null}`
	for i := range merchantIDs {
		name := fmt.Sprintf("%s %d", data.MerchantNames[i%len(data.MerchantNames)], i+1)
		slug := fmt.Sprintf("seed-merchant-%02d", i+1)
		if err := tx.QueryRow(ctx, `INSERT INTO merchants(slug,legal_name,display_name,description,phone,website,commission_bps,review_status) VALUES($1,$2,$2,$3,$4,$5,1500,'approved') RETURNING id`, slug, name, description, fmt.Sprintf("+99890100%04d", i), "https://local.test/business/"+slug).Scan(&merchantIDs[i]); err != nil {
			return err
		}
		for j := 0; j < 1+i%3; j++ {
			var id string
			city := 0 // Toshkent: the only open city in the pilot (cities are listed in seed/data.json)
			if err := tx.QueryRow(ctx, `INSERT INTO merchant_locations(merchant_id,city_id,address,lat,lng,opening_hours,phone) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, merchantIDs[i], cityIDs[city], fmt.Sprintf("Seed ko'chasi %d, %d", i+1, j+1), data.Cities[city].Lat+float64(j)/1000, data.Cities[city].Lng+float64(j)/1000, hours, fmt.Sprintf("+99890100%04d", i)).Scan(&id); err != nil {
				return err
			}
			locationIDs[i] = append(locationIDs[i], id)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO role_assignments(user_id,role,merchant_id) VALUES($1,'merchant_owner',$3),($2,'merchant_cashier',$3)`, userIDs[1], userIDs[2], merchantIDs[0]); err != nil {
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
	deals := make([]dealSeed, 100)
	statuses := []string{"active", "active", "active", "active", "active", "active", "scheduled", "paused", "ended", "draft"}
	for i := range deals {
		merchant := i % len(merchantIDs)
		category := i % len(categoryIDs)
		titleMap := localized{"uz": fmt.Sprintf("%s uchun maxsus taklif %d", data.Categories[category].Name["uz"], i+1), "ru": fmt.Sprintf("Специальное предложение %d", i+1), "en": fmt.Sprintf("Special local offer %d", i+1)}
		title, _ := json.Marshal(titleMap)
		short, _ := json.Marshal(localized{"uz": "Cheklangan muddatli foydali taklif", "ru": "Выгодное предложение на ограниченное время", "en": "A valuable limited-time local offer"})
		detail, _ := json.Marshal(localized{"uz": "Xizmat tafsilotlari va foydalanish shartlari to'liq ko'rsatilgan.", "ru": "Полное описание услуги и условий использования.", "en": "Complete service details and redemption terms."})
		highlights := `[{"uz":"Sifatli xizmat","ru":"Качественный сервис","en":"Quality service"},{"uz":"Qulay vaqt","ru":"Удобное время","en":"Convenient timing"},{"uz":"Tekshirilgan hamkor","ru":"Проверенный партнёр","en":"Verified partner"}]`
		faq := `[{"q":{"uz":"Qanday foydalanaman?","ru":"Как использовать?","en":"How do I use it?"},"a":{"uz":"Kuponni ko'rsating.","ru":"Покажите купон.","en":"Show the voucher."}}]`
		start := time.Now().UTC().Add(-24 * time.Hour)
		end := start.Add(time.Duration(10+i%20) * 24 * time.Hour)
		if statuses[i%len(statuses)] == "scheduled" {
			start = time.Now().UTC().Add(48 * time.Hour)
			end = start.Add(10 * 24 * time.Hour)
		}
		if statuses[i%len(statuses)] == "ended" {
			start = time.Now().UTC().Add(-20 * 24 * time.Hour)
			end = time.Now().UTC().Add(-24 * time.Hour)
		}
		if err := tx.QueryRow(ctx, `INSERT INTO deals(slug,merchant_id,category_id,subcategory_id,title,short_description,description,highlights,whats_included,terms,faq,redemption_instructions,booking_required,status,starts_at,ends_at,voucher_valid_days,max_per_user,sold_count) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8,$7,$9,$7,$10,$11,$12,$13,30,5,$14) RETURNING id`, fmt.Sprintf("seed-deal-%03d", i+1), merchantIDs[merchant], categoryIDs[category], subcategoryIDs[category][i%len(subcategoryIDs[category])], title, short, detail, highlights, faq, i%4 == 0, statuses[i%len(statuses)], start, end, i*3).Scan(&deals[i].ID); err != nil {
			return err
		}
		deals[i].MerchantID, deals[i].LocationID, deals[i].Title = merchantIDs[merchant], locationIDs[merchant][0], title
		for _, locationID := range locationIDs[merchant] {
			if _, err := tx.Exec(ctx, `INSERT INTO deal_locations(deal_id,merchant_location_id) VALUES($1,$2)`, deals[i].ID, locationID); err != nil {
				return err
			}
		}
		for j := 0; j < 1+i%3; j++ {
			original := int64(100000+(i%12)*25000+j*10000) * tiyinPerSom
			price := original * int64(55+i%30) / 100
			var optionID string
			if err := tx.QueryRow(ctx, `INSERT INTO deal_options(deal_id,title,original_price,deal_price,stock_total,stock_sold,sort_order) VALUES($1,$2,$3,$4,500,$5,$6) RETURNING id`, deals[i].ID, title, original, price, i*3, j).Scan(&optionID); err != nil {
				return err
			}
			if j == 0 {
				deals[i].OptionID, deals[i].Price = optionID, price
			}
		}
		for j := 0; j < 3; j++ {
			if _, err := tx.Exec(ctx, `INSERT INTO deal_media(deal_id,url,alt,sort_order,width,height,blurhash) VALUES($1,$2,$3,$4,1200,900,'L6PZfSi_.AyE_3t7t7R**0o#DgR4')`, deals[i].ID, images[(i+j)%len(images)].URL+"?auto=format&fit=crop&w=1200&q=80", title, j); err != nil {
				return err
			}
		}
	}
	for i := 0; i < 400; i++ {
		deal := deals[i%80]
		userID := userIDs[4+i%(len(userIDs)-4)]
		var orderID, itemID, voucherID string
		if err := tx.QueryRow(ctx, `INSERT INTO orders(number,user_id,status,subtotal,total,idempotency_key,paid_at) VALUES('GR-2026-' || lpad(nextval('order_number_seq')::text, 6, '0'),$1,'paid',$2,$2,$3,now()-interval '2 days') RETURNING id`, userID, deal.Price, fmt.Sprintf("seed-order-%06d", i+1)).Scan(&orderID); err != nil {
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
		var redemptionID string
		if err := tx.QueryRow(ctx, `INSERT INTO redemptions(voucher_id,merchant_id,merchant_location_id,staff_user_id,sequence,amount,commission) VALUES($1,$2,$3,$4,1,$5,$6) RETURNING id`, voucherID, deal.MerchantID, deal.LocationID, userIDs[1], deal.Price, ledger.Commission(deal.Price, 1500)).Scan(&redemptionID); err != nil {
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
		if err := tx.QueryRow(ctx, `INSERT INTO reviews(deal_id,merchant_id,user_id,voucher_id,rating,text,locale,status) VALUES($1,$2,$3,$4,$5,$6,$7,'published') RETURNING id`, deal.ID, deal.MerchantID, userID, voucherID, 3+i%3, data.ReviewTexts[locale], locale).Scan(&reviewID); err != nil {
			return err
		}
		if i%5 == 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO review_replies(review_id,merchant_user_id,text) VALUES($1,$2,$3)`, reviewID, userIDs[1], "Fikringiz uchun rahmat! / Спасибо! / Thank you!"); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO homepage_sections(key,title,type,deal_ids,sort_order) VALUES('featured',$1,'deals',$2,1),('popular',$1,'deals',$2,2),('limited',$1,'deals',$2,3)`, description, []string{deals[0].ID, deals[1].ID, deals[2].ID}); err != nil {
		return err
	}
	return nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
	named := []string{"+998900000001", "+998900000002", "+998900000003", "+998900000004", "+998900000005", "+998900000006", "+998900000007", "+998900000008", "+998900000009"}
	if i < len(named) {
		return named[i]
	}
	return fmt.Sprintf("+99891%07d", i)
}
