package eventhub

import (
	"encoding/xml"
	"io"
	"net/http"
)

type feed struct {
	XMLName xml.Name `xml:"feed"`
	Entry   []entry  `xml:"entry"`
}

type entry struct {
	XMLName xml.Name `xml:"entry"`
	Title   string   `xml:"title"`
}

func parseXMLResponse(response *http.Response) (*feed, error) {

	body, _ := io.ReadAll(response.Body)

	var feed feed
	err := xml.Unmarshal(body, &feed)
	return &feed, err
}
