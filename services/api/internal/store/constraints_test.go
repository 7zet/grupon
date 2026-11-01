package store_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDatabaseConstraints(t *testing.T) {
	ctx := context.Background()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		container, err := postgres.Run(ctx, "postgres:18.6-alpine",
			postgres.WithDatabase("app"), postgres.WithUsername("app"), postgres.WithPassword("app"),
			testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)),
		)
		if err != nil {
			t.Skipf("Docker is unavailable: %v", err)
		}
		t.Cleanup(func() { _ = container.Terminate(ctx) })
		url, err = container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatal(err)
		}
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	_, file, _, _ := runtime.Caller(0)
	migrations := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
	if err := goose.UpContext(ctx, db, migrations); err != nil {
		t.Fatal(err)
	}

	fixture := `
INSERT INTO users(id,email,password_hash,name) VALUES
 ('019a0000-0000-7000-8000-000000000001','one@test.local','hash','One'),
 ('019a0000-0000-7000-8000-000000000002','two@test.local','hash','Two');
INSERT INTO merchants(id,slug,legal_name,display_name,phone) VALUES ('019a0000-0000-7000-8000-000000000010','merchant','Merchant','Merchant','+998900000000');
INSERT INTO cities(id,slug,name,lat,lng) VALUES ('019a0000-0000-7000-8000-000000000020','city','{"uz":"Shahar","ru":"Город","en":"City"}',0,0);
INSERT INTO categories(id,slug,name,icon) VALUES ('019a0000-0000-7000-8000-000000000030','category','{"uz":"Tur","ru":"Тип","en":"Type"}','tag');
INSERT INTO subcategories(id,category_id,slug,name) VALUES ('019a0000-0000-7000-8000-000000000031','019a0000-0000-7000-8000-000000000030','subcategory','{"uz":"Tur","ru":"Тип","en":"Type"}');
INSERT INTO merchant_locations(id,merchant_id,city_id,address,lat,lng,opening_hours,phone) VALUES ('019a0000-0000-7000-8000-000000000040','019a0000-0000-7000-8000-000000000010','019a0000-0000-7000-8000-000000000020','Address',0,0,'{}','+998900000000');
INSERT INTO deals(id,slug,merchant_id,category_id,subcategory_id,title,short_description,description,highlights,terms,redemption_instructions,starts_at,ends_at,voucher_valid_days)
VALUES ('019a0000-0000-7000-8000-000000000050','deal','019a0000-0000-7000-8000-000000000010','019a0000-0000-7000-8000-000000000030','019a0000-0000-7000-8000-000000000031','{"uz":"Taklif","ru":"Предложение","en":"Deal"}','{"uz":"Qisqa","ru":"Кратко","en":"Short"}','{"uz":"Batafsil","ru":"Подробно","en":"Detail"}','[{"uz":"1"},{"uz":"2"},{"uz":"3"}]','{"uz":"Shart","ru":"Условие","en":"Term"}','{"uz":"Korsat","ru":"Покажите","en":"Show"}',now(),now()+interval '1 day',30);
INSERT INTO deal_options(id,deal_id,title,original_price,deal_price,stock_total) VALUES ('019a0000-0000-7000-8000-000000000060','019a0000-0000-7000-8000-000000000050','{"uz":"Tanlov","ru":"Опция","en":"Option"}',200,100,10);
INSERT INTO orders(id,number,user_id,subtotal,total,idempotency_key) VALUES
 ('019a0000-0000-7000-8000-000000000070','GR-1','019a0000-0000-7000-8000-000000000001',100,100,'key-1'),
 ('019a0000-0000-7000-8000-000000000071','GR-2','019a0000-0000-7000-8000-000000000002',100,100,'key-2');
INSERT INTO order_items(id,order_id,deal_id,deal_option_id,title_snapshot,unit_price_snapshot,original_price_snapshot,quantity) VALUES
 ('019a0000-0000-7000-8000-000000000080','019a0000-0000-7000-8000-000000000070','019a0000-0000-7000-8000-000000000050','019a0000-0000-7000-8000-000000000060','{"uz":"Tanlov","ru":"Опция","en":"Option"}',100,200,1),
 ('019a0000-0000-7000-8000-000000000081','019a0000-0000-7000-8000-000000000071','019a0000-0000-7000-8000-000000000050','019a0000-0000-7000-8000-000000000060','{"uz":"Tanlov","ru":"Опция","en":"Option"}',100,200,1);
INSERT INTO vouchers(id,order_item_id,user_id,deal_id,deal_option_id,merchant_id,code,qr_token,valid_from,valid_until) VALUES
 ('019a0000-0000-7000-8000-000000000090','019a0000-0000-7000-8000-000000000080','019a0000-0000-7000-8000-000000000001','019a0000-0000-7000-8000-000000000050','019a0000-0000-7000-8000-000000000060','019a0000-0000-7000-8000-000000000010','AAAA-0001','token-1',now(),now()+interval '1 day'),
 ('019a0000-0000-7000-8000-000000000091','019a0000-0000-7000-8000-000000000081','019a0000-0000-7000-8000-000000000002','019a0000-0000-7000-8000-000000000050','019a0000-0000-7000-8000-000000000060','019a0000-0000-7000-8000-000000000010','AAAA-0002','token-2',now(),now()+interval '1 day');`
	if _, err := db.ExecContext(ctx, fixture); err != nil {
		t.Fatal(err)
	}

	t.Run("stock cannot exceed total", func(t *testing.T) {
		if _, err := db.ExecContext(ctx, `UPDATE deal_options SET stock_reserved=11 WHERE id='019a0000-0000-7000-8000-000000000060'`); err == nil {
			t.Fatal("expected stock constraint violation")
		}
	})
	t.Run("one review per voucher", func(t *testing.T) {
		insert := `INSERT INTO reviews(deal_id,merchant_id,user_id,voucher_id,rating,text,locale) VALUES('019a0000-0000-7000-8000-000000000050','019a0000-0000-7000-8000-000000000010','019a0000-0000-7000-8000-000000000001','019a0000-0000-7000-8000-000000000090',5,'This review has enough characters.','en')`
		if _, err := db.ExecContext(ctx, insert); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, insert); err == nil {
			t.Fatal("expected unique voucher review violation")
		}
	})
	t.Run("voucher codes are unique", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `UPDATE vouchers SET code='AAAA-0001' WHERE id='019a0000-0000-7000-8000-000000000091'`)
		if err == nil {
			t.Fatal("expected unique voucher code violation")
		}
	})
}
