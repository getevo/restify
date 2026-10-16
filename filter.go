package restify

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/generic"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

// filterConditions is a map that defines the filter conditions used in the filterMapper function.
// The keys represent the condition name, and the values represent the corresponding condition symbol or keyword.
var filterConditions = map[string]string{
	"eq":       "=",
	"neq":      "!=",
	"gt":       ">",
	"lt":       "<",
	"gte":      ">=",
	"lte":      "<=",
	"in":       "IN",
	"between":  "BETWEEN",
	"contains": "LIKE",
	"isnull":   "IS NULL",
	"notnull":  "IS NOT NULL",
}

// ContainOperator represents the string value "contains"
// which is used as an operator for containment operations.
// Examples of containment operations could be checking if a string contains
// a specific substring or if an array contains a specific element.
// This constant is used to indicate the containment operator in code logic.
// NotNullOperator represents the string value "notnull"
// which is used as an operator for checking if a value is not null.
// This constant is used to indicate the not null operator in code logic.
// IsNullOperator represents the string value "isnull"
// which is used as an operator for checking if a value is null.
// This constant is used to indicate the is null operator in code logic.
// InOperator represents the string
const (
	ContainOperator        = "contains"
	NotNullOperator        = "notnull"
	IsNullOperator         = "isnull"
	InOperator             = "in"
	NotInOperator          = "notin"
	BetweenOperator        = "between"
	FulltextSearchOperator = "search"
)

var aggregateFieldRegex = regexp.MustCompile(`(?i)^[a-z0-9_*\-]+\.(count|sum|min|max|avg|first|last)$`)

var groupByRegex = regexp.MustCompile(`(?mi)^[a-z0-9_\-.,]+$`)

// result will be [{"column":"column1","condition":"condition1","value":"value1"},{"column":"column2","condition":"condition2","value":"value2"},{"column":"column3","condition":"condition
func filterRegEx(str string) []map[string]string {
	var re = regexp.MustCompile(`(?m)((?P<column>[a-zA-Z_\-0-9]+)\[(?P<condition>[a-zA-Z]+)\](\=((?P<value>[a-zA-Z_\-0-9\s\%\:\,.\*]+))){0,1})\&*`)
	var keys = re.SubexpNames()
	var result []map[string]string
	for _, match := range re.FindAllStringSubmatch(str, -1) {
		item := map[string]string{}
		for i, name := range keys {
			if i != 0 && name != "" {
				item[name] = match[i]
			}
		}
		result = append(result, item)
	}
	return result
}

// filterMapper applies filters to the given query based on the provided filter string.
// It parses the filter
func filterMapper(filters string, context *Context, query *gorm.DB) (*gorm.DB, *Error) {
	var table = context.Schema.Table
	fRegEx := filterRegEx(filters)
	for _, filter := range fRegEx {
		var obj = context.CreateIndirectObject().Interface()
		var ref = reflect.ValueOf(obj)
		fnd := false
		var fieldName = ""
		filter["value"], _ = url.QueryUnescape(filter["value"])
		for _, field := range context.Schema.Fields {
			if field.DBName == filter["column"] {
				fieldName = field.Name
				fnd = true
				break
			}
		}
		if !fnd {
			return nil, &ErrorColumnNotExist
		}
		v := ref.FieldByName(fieldName)

		if obj, ok := v.Interface().(interface {
			RestFilter(context *Context, query *gorm.DB, filter map[string]string)
		}); ok {
			obj.RestFilter(context, query, filter)
			return query, nil
		}

		if filter["condition"] == NotNullOperator || filter["condition"] == IsNullOperator {
			if filter["column"] == "deleted_at" {
				query = query.Unscoped()
			}
			query = query.Where(fmt.Sprintf("`%s`.`%s` %s", table, filter["column"], filterConditions[filter["condition"]]))
		} else {
			if filter["condition"] == ContainOperator {
				query = query.Where(fmt.Sprintf("`%s`.`%s` %s ?", table, filter["column"], "LIKE"), fmt.Sprintf("%%%s%%", filter["value"]))
			} else if filter["condition"] == NotInOperator {
				valSlice := strings.Split(filter["value"], ",")
				query = query.Where(fmt.Sprintf("`%s`.`%s` NOT IN (?)", table, filter["column"]), valSlice)
			} else if filter["condition"] == InOperator {
				valSlice := strings.Split(filter["value"], ",")
				query = query.Where(fmt.Sprintf("`%s`.`%s` IN (?)", table, filter["column"]), valSlice)
			} else if filter["condition"] == FulltextSearchOperator {
				query = query.Where(fmt.Sprintf("MATCH (`%s`.`%s`) AGAINST (? IN NATURAL LANGUAGE MODE)", table, filter["column"]), filter["value"])
			} else if filter["condition"] == BetweenOperator {
				valSlice := strings.Split(filter["value"], ",")
				if len(valSlice) != 2 {
					var err = NewError(fmt.Sprintf("invalid filter value for between operator, expected 2 values got %d", len(valSlice)), 400)
					return query, &err
				}
				t1, err := generic.Parse(valSlice[0]).Time()
				if err != nil {
					var err = NewError(fmt.Sprintf("invalid filter value for between operator, expected date got %s", valSlice[0]), 400)
					return query, &err
				}
				t2, err := generic.Parse(valSlice[1]).Time()
				if err != nil {
					var err = NewError(fmt.Sprintf("invalid filter value for between operator, expected date got %s", valSlice[1]), 400)
					return query, &err
				}
				query = query.Where(fmt.Sprintf("`%s`.`%s` BETWEEN ? AND ?", table, filter["column"]), t1.Format("2006-01-02 15:04:05"), t2.Format("2006-01-02 15:04:05"))

			} else {
				if v, ok := filterConditions[filter["condition"]]; ok {
					query = query.Where(fmt.Sprintf("`%s`.`%s` %s ?", table, filter["column"], v), filter["value"])
				} else {
					var err = NewError(fmt.Sprintf("invalid filter condition %s", filter["condition"]), 500)
					return query, &err
				}

			}
		}
	}

	for _, condition := range context.Conditions {
		query = query.Where(fmt.Sprintf("`%s`.`%s` %s ?", table, condition.Field, condition.Op), condition.Value)
	}
	//query = query.Debug()
	return query, nil
}

// ApplyFilters applies filters to the query based on the request parameters in the context. It modifies the
func (context *Context) ApplyFilters(query *gorm.DB) (*gorm.DB, *Error) {
	var table = context.Schema.Table
	if context.CustomFilter != nil {
		query = context.CustomFilter(context, query)
	}
	var association = context.Request.Query("associations").String()
	if association != "" {
		if association == "1" || association == "true" || association == "*" {
			query = query.Preload(clause.Associations)
		} else if association == "deep" {
			var preload = getAssociations("", context.Schema)
			for _, item := range preload {
				query = query.Preload(item)
			}
		} else {
			var ls = strings.Split(association, ",")
			for _, item := range ls {
				query = query.Preload(item)
			}
		}

	}

	var order = context.Request.Query("order").String()
	if order != "" {
		clause, httpErr := parseOrderBy(order, context.Schema)
		if httpErr != nil {
			return nil, httpErr
		}
		if clause != "" {
			query = query.Order(clause)
		}
	}

	var groupBy = context.Request.Query("group_by").String()
	if groupBy != "" {
		if groupByRegex.MatchString(groupBy) {
			query = query.Group(groupBy)
		}
	}

	var fields = context.Request.Query("fields").String()
	if len(fields) > 0 {
		var columns []string
		for _, field := range strings.Split(fields, ",") {
			field = strings.TrimSpace(field)
			if field == "" || aggregateFieldRegex.MatchString(field) {
				// "col.count" etc: the Aggregate handler checks it and sets its own Select
				continue
			}
			if _, ok := context.Schema.FieldsByDBName[field]; !ok {
				return nil, &ErrorInvalidColumn
			}
			columns = append(columns, fmt.Sprintf("`%s`.`%s`", table, field))
		}
		if len(columns) > 0 {
			query = query.Select(columns)
		}
	}

	var join = context.Request.Query("join").String()
	if len(join) > 0 {
		if relations := relationsMapper(join); relations != "" {
			query = query.Preload(relations)
		}
	}
	var httpErr *Error
	query, httpErr = filterMapper(context.Request.QueryString(), context, query)

	var offset = context.Request.Query("offset").Int()
	if offset > 0 {
		query = query.Offset(offset)
	}

	var limit = context.Request.Query("limit").Int()
	if limit > 0 {
		query = query.Limit(limit)
	}
	return query, httpErr
}

// parseOrderBy turns "col.asc,col2.desc" into an ORDER BY clause. Every column must be a
// real column of the model; anything else is rejected, never passed through as raw SQL.
func parseOrderBy(input string, sch *schema.Schema) (string, *Error) {
	var clauses []string
	for _, cl := range strings.Split(input, ",") {
		cl = strings.TrimSpace(cl)
		if cl == "" {
			continue
		}
		column, direction := cl, "ASC"
		if i := strings.LastIndexAny(cl, ". "); i >= 0 {
			if d := strings.ToUpper(strings.TrimSpace(cl[i+1:])); d == "ASC" || d == "DESC" {
				column, direction = strings.TrimSpace(cl[:i]), d
			}
		}
		if _, ok := sch.FieldsByDBName[column]; !ok {
			return "", &ErrorInvalidColumn
		}
		clauses = append(clauses, fmt.Sprintf("`%s`.`%s` %s", sch.Table, column, direction))
	}
	return strings.Join(clauses, ","), nil
}
