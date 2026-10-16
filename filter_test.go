package restify

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

type orderTestModel struct {
	OrderID   int `gorm:"primaryKey"`
	CreatedAt string
	Status    string
}

func (orderTestModel) TableName() string { return "order" }

func TestParseOrderBy(t *testing.T) {
	sch, err := schema.Parse(&orderTestModel{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	ok := map[string]string{
		"created_at.desc":               "`order`.`created_at` DESC",
		"order_id.desc,created_at.desc": "`order`.`order_id` DESC,`order`.`created_at` DESC",
		"created_at":                    "`order`.`created_at` ASC",
		"created_at asc":                "`order`.`created_at` ASC",
		" status.ASC , ":                "`order`.`status` ASC",
	}
	for in, want := range ok {
		got, e := parseOrderBy(in, sch)
		if e != nil || got != want {
			t.Errorf("parseOrderBy(%q) = %q, %v; want %q", in, got, e, want)
		}
	}
	for _, in := range []string{
		"(SELECT 1)",
		"created_at.desc, (select 1)",
		"created_at`;--.asc",
		"product.Name.asc",
		"unknown.desc",
		"1",
	} {
		if got, e := parseOrderBy(in, sch); e == nil {
			t.Errorf("parseOrderBy(%q) = %q; want error", in, got)
		}
	}
}

func TestAggregateFieldRegex(t *testing.T) {
	for _, in := range []string{"voucher_id.count", "*.count", "amount.SUM"} {
		if !aggregateFieldRegex.MatchString(in) {
			t.Errorf("%q should match", in)
		}
	}
	for _, in := range []string{"(select 1).count", "x.count,y", "id", "a.count)"} {
		if aggregateFieldRegex.MatchString(in) {
			t.Errorf("%q should not match", in)
		}
	}
}
