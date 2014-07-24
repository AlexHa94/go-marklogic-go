package go_marklogic_go

import (
	//"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
)

// Response represents a response from the search API
type Response struct {
	Total      int64     `xml:"total,attr"`
	Start      int64     `xml:"start,attr"`
	PageLength int64     `xml:"page-length,attr"`
	Results    []*Result `xml:"http://marklogic.com/appservices/search result"`
	Facets     []*Facet  `xml:"http://marklogic.com/appservices/search facet"`
}

// Result is an individual document fragment found by the search
type Result struct {
	Uri        string     `xml:"uri,attr"`
	Href       string     `xml:"href,attr"`
	MimeType   string     `xml:"mimetype,attr"`
	Format     string     `xml:"format,attr"`
	Path       string     `xml:"path,attr"`
	Index      int64      `xml:"index,attr"`
	Score      int64      `xml:"score,attr"`
	Confidence float64    `xml:"confidence,attr"`
	Fitness    float64    `xml:"fitness,attr"`
	Snippets   []*Snippet `xml:"http://marklogic.com/appservices/search snippet"`
}

//Snippet
type Snippet struct {
	Location string   `xml:"uri,attr"`
	Matches  []*Match `xml:"http://marklogic.com/appservices/search match"`
}

// Matches in a snippet
type Match struct {
	Path string
	Text []*Text
}

// Text in a match. HightlightedText tells whether the text matched the query
// or not.
type Text struct {
	Text            string
	HighlightedText bool
}

// Facet represents a facet and contains a slice of FacetValue
type Facet struct {
	Name        string        `xml:"name,attr"`
	Type        string        `xml:"type,attr"`
	FacetValues []*FacetValue `xml:"http://marklogic.com/appservices/search facet-value"`
}

//
type FacetValue struct {
	Name  string `xml:"name,attr"`
	Label string `xml:",chardata"`
	Count int64  `xml:"count,attr"`
}

func (c *Client) Search(text string) (*Response, error) {
	req, _ := http.NewRequest("GET", c.Base+"/search?q="+text, nil)
	ApplyAuth(c, req)
	resp, _ := c.HttpClient.Do(req)
	defer resp.Body.Close()
	return ReadResults(resp.Body)
}

func (c *Client) StructuredSearch(query *Query) (*Response, error) {
	buf := query.Encode()
	req, _ := http.NewRequest("POST", c.Base+"/search", buf)
	ApplyAuth(c, req)
	resp, _ := c.HttpClient.Do(req)
	defer resp.Body.Close()
	return ReadResults(resp.Body)
}

func ReadResults(reader io.Reader) (*Response, error) {
	results := &Response{}
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(results); err != nil {
		return nil, err
	}
	return results, nil
}

func (m *Match) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for i := range start.Attr {
		attr := start.Attr[i]
		if attr.Name.Local == "path" {
			m.Path = attr.Value
			break
		}
	}
	for {
		if token, err := d.Token(); (err == nil) && (token != nil) {
			switch t := token.(type) {
			case xml.StartElement:
				var content string
				e := xml.StartElement(t)
				d.DecodeElement(&content, &e)
				text := &Text{
					Text:            content,
					HighlightedText: e.Name.Space == "http://marklogic.com/appservices/search" && e.Name.Local == "highlight",
				}
				m.Text = append(m.Text, text)
			case xml.EndElement:
				e := xml.EndElement(t)
				if e.Name.Space == "http://marklogic.com/appservices/search" && e.Name.Local == "match" {
					return nil
				}
			case xml.CharData:
				b := xml.CharData(t)
				text := &Text{
					Text:            string([]byte(b)),
					HighlightedText: false,
				}
				m.Text = append(m.Text, text)
			}
		} else {
			return err
		}
	}
}
