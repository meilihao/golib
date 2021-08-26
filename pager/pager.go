package pager

const (
	DefaultSize = 10
)

// Sizes: smaller -> bigger
type Pager struct {
	Page  int `json:"page"` // current page
	Size  int `json:"size"`
	Total int `json:"total"`

	Start     int   `json:"start"`
	End       int   `json:"end"`
	Sizes     []int `json:"sizes"`
	TotalPage int   `json:"total_page"`
}

func isValidSizes(size int, sizes []int) bool {
	for _, v := range sizes {
		if size == v {
			return true
		}
	}

	return false
}

func NewPager(page, size int, sizes ...int) *Pager {
	p := new(Pager)

	if page <= 0 {
		page = 1
	}
	p.Page = page

	n := len(sizes)
	if size <= 0 {
		if n > 0 {
			size = sizes[0]
		} else {
			size = DefaultSize
		}
	} else {
		if n > 0 && !isValidSizes(size, sizes) {
			size = sizes[0]
		}
	}

	p.Size = size
	p.Sizes = sizes

	return p
}

func (p *Pager) SetTotal(num int) {
	if num > 0 {
		p.Total = num

		d := p.Total % p.Size
		if d == 0 {
			p.TotalPage = p.Total / p.Size
		} else {
			p.TotalPage = p.Total/p.Size + 1
		}

		p.Start = p.Offset() + 1
		p.End = p.Page * p.Size

		if p.End > p.Total {
			p.End = p.Total
		}
	}
}

func (p *Pager) Offset() int {
	return (p.Page - 1) * p.Size
}

func (p *Pager) HasData() bool {
	return p != nil && p.Total > 0 && p.Page <= p.TotalPage
}
