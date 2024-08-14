package restify

import (
	"github.com/getevo/evo/v2"
	"gorm.io/gorm"
)

// Pagination represents a utility type for handling pagination in Go.
//
// Fields:
// - Records: Total number of rows.
// - CurrentPage: Current page loaded.
// - Pages: Total number of pages.
// - Limit: Number of rows per page.
// - First: First page.
// - Last: Last page.
// - PageRange: Range of visible pages.
//
// Methods:
// - SetCurrentPage: Sets the current page based on the provided value. If the value is 0, the current page is set to 1.
// - SetLimit: Sets the limit of rows per page. If the value is 0, the limit is set to the minimum limit of 10. If the limit is less than the minimum limit, it is set to the minimum
type Pagination struct {
	Records    int         `json:"records,omitempty"`    // Total rows
	Pages      int         `json:"pages,omitempty"`      // total number of pages
	Limit      int         `json:"limit,omitempty"`      // number of rows per page
	First      int         `json:"first,omitempty"`      // First Page
	Last       int         `json:"last,omitempty"`       // Last Page
	PageRange  []int       `json:"page_range,omitempty"` // Range of visible pages
	Data       interface{} `json:"data,omitempty"`
	Total      int64       `json:"total"`
	Offset     int         `json:"offset"`
	TotalPages int         `json:"total_pages"`
	Page       int         `json:"current_page"`
	Size       int         `json:"size"`
	Success    bool        `json:"success"`
	Error      string      `json:"error"`
	Type       string      `json:"type"`
}

// SetCurrentPage sets the value of CurrentPage in the Pagination struct.
// If the input page is not equal to zero, p.CurrentPage will be set to the input page.
// Otherwise, p.CurrentPage will be set to 1.
func (p *Pagination) SetCurrentPage(page int) {
	if page > 0 {
		p.Page = page
	} else {
		p.Page = 1
	}
}

// SetLimit sets the limit per page for the pagination struct. The limit must be between 10 and 100 (inclusive). If the limit is 0, it will be set to 10. If the limit is less than 10
func (p *Pagination) SetLimit(limit int) {
	maxLimit := 100
	minLimit := 10

	if limit != 0 {
		p.Limit = limit
	} else {
		p.Limit = minLimit
	}

	if p.Limit < minLimit {
		p.Limit = minLimit
	} else if p.Limit > maxLimit {
		p.Limit = maxLimit
	}

}

// SetPages sets the total number of pages in the pagination struct based on the number of records and the limit per page.
// If the number of records is 0, it sets the number of pages to 1.
// If there is no remainder when dividing the number of records by the limit, it sets the number of pages to the integer division.
// Otherwise, it sets the number of pages to the integer division plus 1.
// If the number of pages is 0, it sets it to 1.
// After setting the number of pages, it calls the SetLast and SetPageRange methods to update the last page indicator and the range of visible pages respectively.
func (p *Pagination) SetPages() {

	if p.Records == 0 {
		p.Pages = 1
		return
	}

	res := p.Records % p.Limit
	if res == 0 {
		p.Pages = p.Records / p.Limit
	} else {
		p.Pages = (p.Records / p.Limit) + 1

	}

	if p.Pages == 0 {
		p.Pages = 1
	}

	p.SetLast()
	p.SetPageRange()

}

// SetLast sets the value of the Last page in the pagination struct.
// It calculates the value by adding the current offset to the limit.
// If the calculated value is greater than the total number of records,
// it sets the Last page to the total number of records.
func (p *Pagination) SetLast() {
	p.Last = p.GetOffset() + p.Limit
	if p.Last > p.Records {
		p.Last = p.Records
	}
}

// SetPageRange sets the value of PageRange in the Pagination struct.
// It determines the range of visible pages based on the current page and the total number of pages.
func (p *Pagination) SetPageRange() {
	to := p.Page + 5
	if to > p.Pages {
		to = p.Pages + 1
	}
	for i := p.Page - 2; i < to; i++ {
		if i > 0 {
			p.PageRange = append(p.PageRange, i)
		}
	}
}

// GetOffset calculates the offset for paginating the data based on the current page and limit
func (p *Pagination) GetOffset() int {
	return (p.GetPage() - 1) * p.Limit
}

// GetPage returns the current page of the pagination struct
func (p *Pagination) GetPage() int {
	if p.Page < 1 {
		p.Page = 1
	}

	return p.Page
}

func (p *Pagination) Create(model *gorm.DB, i interface{}, request *evo.Request) error {
	var total int64
	if err := model.Count(&total).Error; err != nil {
		return err
	}
	p.Records = int(total)
	var limit = request.Query("limit").Int()
	var page = request.Query("page").Int()
	if limit < 10 {
		limit = 10
	}
	if page < 1 {
		page = 1
	}
	p.SetCurrentPage(page)
	p.SetLimit(limit)
	p.SetPages()

	model = model.Limit(limit)
	model = model.Offset(p.GetOffset())
	if err := model.Find(i).Error; err != nil {
		return err
	}
	p.Data = i
	return nil
}
