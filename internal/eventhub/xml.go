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
	Content content  `xml:"content"`
}

type content struct {
	XMLName             xml.Name            `xml:"content"`
	EventHubDescription eventhubDescription `xml:"EventHubDescription"`
}

type eventhubDescription struct {
	XMLName                xml.Name `xml:"EventHubDescription"`
	MessageRetentionInDays int      `xml:"MessageRetentionInDays"`
	PartitionCount         int      `xml:"PartitionCount"`
	PartitionIDs           []string `xml:"PartitionIds>string"`
}

func parseXMLResponse(response *http.Response) (*feed, error) {

	body, _ := io.ReadAll(response.Body)

	var feed feed
	err := xml.Unmarshal(body, &feed)
	return &feed, err
}
