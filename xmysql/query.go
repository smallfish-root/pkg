package xmysql

import "github.com/jinzhu/gorm"

type QueryOption struct {
	Skip  uint
	Limit uint
	Order []string
}

type QueryOptFunc func(*QueryOption)

func ApplyQueryOpts(db *gorm.DB, opts ...QueryOptFunc) *gorm.DB {
	queryOption := &QueryOption{}
	for _, opt := range opts {
		opt(queryOption)
	}

	for _, order := range queryOption.Order {
		db = db.Order(order)
	}

	if queryOption.Limit != 0 {
		db = db.Limit(queryOption.Limit)
	}

	if queryOption.Skip != 0 {
		db = db.Offset(queryOption.Skip)
	}
	return db
}

func Skip(skip uint) QueryOptFunc {
	return func(option *QueryOption) {
		option.Skip = skip
	}
}

func Limit(limit uint) QueryOptFunc {
	return func(option *QueryOption) {
		option.Limit = limit
	}
}

func Order(order ...string) QueryOptFunc {
	return func(option *QueryOption) {
		option.Order = append(option.Order, order...)
	}
}
