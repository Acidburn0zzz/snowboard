package snowboard

import (
	"net/http"

	"github.com/subosito/snowboard/blueprint"
)

func digString(key string, el *Element) string {
	return el.Path(key).Value().String()
}

func digTitle(el *Element) string {
	return digString("meta.title", el)
}

func digDescription(el *Element) string {
	return el.Path("content").Index(0).Path("content").Value().String()
}

func digMetadata(el *Element) []blueprint.Metadata {
	mds := []blueprint.Metadata{}

	children, err := el.Path("attributes.meta").Children()
	if err != nil {
		return mds
	}

	for _, v := range children {
		md := blueprint.Metadata{
			Name:  digString("content.key.content", v),
			Value: digString("content.value.content", v),
		}

		mds = append(mds, md)
	}

	return mds
}

func digResourceGroups(el *Element) (gs []blueprint.ResourceGroup) {
	children, err := el.Path("content").Children()
	if err != nil {
		return
	}

	for _, child := range children {
		if digString("element", child) == "category" {
			g := &blueprint.ResourceGroup{
				Title:     digString("meta.title", child),
				Resources: digResources(child),
			}

			gs = append(gs, *g)
		}
	}

	return
}

func digResources(el *Element) (rs []blueprint.Resource) {
	children, err := el.Path("content").Children()
	if err != nil {
		return
	}

	for _, child := range children {
		if digString("element", child) == "resource" {
			r := &blueprint.Resource{
				Title:       digString("meta.title", child),
				Transitions: digTransitions(child),
				Href:        extractHrefs(child),
			}

			rs = append(rs, *r)
		}
	}

	return
}

func digTransitions(el *Element) (ts []blueprint.Transition) {
	children, err := el.Path("content").Children()
	if err != nil {
		return
	}

	for _, child := range children {
		if digString("element", child) == "transition" {
			t := &blueprint.Transition{
				Title:        digString("meta.title", child),
				Transactions: digTransactions(child),
			}

			ts = append(ts, *t)
		}
	}

	return
}

func digTransactions(el *Element) (xs []blueprint.Transaction) {
	children, err := el.Path("content").Children()
	if err != nil {
		return
	}

	for _, child := range children {
		if digString("element", child) == "httpTransaction" {
			x := &blueprint.Transaction{
				Request:  extractRequest(child),
				Response: extractResponse(child),
			}

			xs = append(xs, *x)
		}
	}

	return
}

func extractRequest(child *Element) (r blueprint.Request) {
	if digString("element", child) == "httpRequest" {
		return blueprint.Request{
			Method: digString("attributes.method", child),
		}
	}

	return
}

func extractResponse(child *Element) (r blueprint.Response) {
	if digString("element", child) == "httpResponse" {
		return blueprint.Response{
			StatusCode: int(child.Path("attributes.statusCode").Value().Int()),
			Headers:    extractHeaders(child),
		}
	}

	return
}

func extractHeaders(child *Element) (h http.Header) {
	if digString("element", child) == "httpHeaders" {
		contents, err := child.Path("content").Children()
		if err != nil {
			return
		}

		for _, content := range contents {
			key := digString("content.key.content", content)
			val := digString("content.value.content", content)

			h.Set(key, val)
		}

		return
	}

	return
}

func extractHrefs(child *Element) (h blueprint.Href) {
	if child.Path("href").Value().IsValid() {
		h.Path = digString("href", child)
	}

	contents, err := child.Path("attributes.hrefVariables.content").Children()
	if err != nil {
		return
	}

	for _, content := range contents {
		v := &blueprint.HVariable{
			Name:        digString("content.key.content", content),
			Value:       digString("content.value.content", content),
			Description: digString("meta.description", content),
		}

		h.Variables = append(h.Variables, *v)
	}

	return
}
