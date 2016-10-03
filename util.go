package snowboard

import (
	"reflect"
	"strconv"

	"github.com/subosito/snowboard/blueprint"
)

func digTitle(el *Element) string {
	if hasClass("api", el) {
		return el.Path("meta.title").String()
	}

	return ""
}

func digDescription(el *Element) string {
	children, err := el.Path("content").Children()
	if err != nil {
		return ""
	}

	for _, child := range children {
		if child.Path("element").String() == "copy" {
			return child.Path("content").String()
		}
	}

	return ""
}

func digMetadata(el *Element) []blueprint.Metadata {
	mds := []blueprint.Metadata{}

	children, err := el.Path("attributes.meta").Children()
	if err != nil {
		return mds
	}

	for _, v := range children {
		md := blueprint.Metadata{
			Key:   v.Path("content.key.content").String(),
			Value: v.Path("content.value.content").String(),
		}

		mds = append(mds, md)
	}

	return mds
}

func digResourceGroups(el *Element) (gs []blueprint.ResourceGroup) {
	children := filterContentByClass("resourceGroup", el)

	for _, child := range children {
		g := &blueprint.ResourceGroup{
			Title:       child.Path("meta.title").String(),
			Description: digDescription(child),
			Resources:   digResources(child),
		}

		gs = append(gs, *g)
	}

	return
}

func digResources(el *Element) []blueprint.Resource {
	children := filterContentByElement("resource", el)

	cr := make(chan blueprint.Resource)
	oc := make([]string, len(children))
	rs := make([]blueprint.Resource, len(children))

	for i, child := range children {
		oc[i] = child.Path("meta.title").String()

		go func(c *Element) {
			cr <- blueprint.Resource{
				Title:          c.Path("meta.title").String(),
				Description:    digDescription(c),
				Transitions:    digTransitions(c),
				Href:           extractHrefs(c),
				DataStructures: digDataStructures(c),
			}
		}(child)
	}

	for i := 0; i < len(children); i++ {
		r := <-cr

		for n := range oc {
			if oc[n] == r.Title {
				rs[n] = r
			}
		}
	}

	return rs
}

func digDataStructures(el *Element) (ds []blueprint.DataStructure) {
	children := filterContentByElement("dataStructure", el)

	if len(children) != 0 {
		return extractDataStructures(children)
	}

	contents := filterContentByClass("dataStructures", el)

	for _, c := range contents {
		children = filterContentByElement("dataStructure", c)
		cs := extractDataStructures(children)
		ds = append(ds, cs...)
	}

	return
}

func extractDataStructures(children []*Element) (ds []blueprint.DataStructure) {
	for _, child := range children {
		cx, err := child.Path("content").Children()
		if err != nil {
			continue
		}

		for _, c := range cx {
			d := blueprint.DataStructure{
				Name: c.Path("element").String(),
				ID:   c.Path("meta.id").String(),
			}

			cz, err := c.Path("content").Children()
			if err == nil {
				for _, z := range cz {
					if z.Path("content").Value().IsValid() {
						s := blueprint.Structure{
							Required:    isContains("attributes.typeAttributes", "required", z),
							Description: z.Path("meta.description").String(),
							Key:         z.Path("content.key.content").String(),
							Value:       z.Path("content.value.content").String(),
							Kind:        z.Path("content.value.element").String(),
						}

						d.Structures = append(d.Structures, s)
					} else {
						d.Items = append(d.Items, z.Path("element").String())
					}
				}
			}

			ds = append(ds, d)
		}
	}

	return
}

func digTransitions(el *Element) (ts []blueprint.Transition) {
	children := filterContentByElement("transition", el)

	for _, child := range children {
		t := &blueprint.Transition{
			Title:        child.Path("meta.title").String(),
			Description:  digDescription(child),
			Transactions: digTransactions(child),
			Href:         extractHrefs(child),
		}

		c := child.Path("attributes.data")
		if c.Value().IsValid() {
			if c.Path("element").String() == "dataStructure" {
				t.DataStructures = extractDataStructures([]*Element{c})
			}
		}

		ts = append(ts, *t)
	}

	return
}

func digTransactions(el *Element) (xs []blueprint.Transaction) {
	children := filterContentByElement("httpTransaction", el)

	for _, child := range children {
		cx, err := child.Path("content").Children()
		if err != nil {
			continue
		}

		xs = append(xs, extractTransaction(cx))
	}

	return
}

func extractTransaction(children []*Element) (x blueprint.Transaction) {
	for _, child := range children {
		if child.Path("element").String() == "httpRequest" {
			x.Request = extractRequest(child)
		}

		if child.Path("element").String() == "httpResponse" {
			x.Response = extractResponse(child)
		}
	}

	return
}

func extractRequest(child *Element) (r blueprint.Request) {
	r = blueprint.Request{
		Title:   child.Path("meta.title").String(),
		Method:  child.Path("attributes.method").String(),
		Headers: extractHeaders(child.Path("attributes.headers")),
	}

	cx, err := child.Path("content").Children()
	if err != nil {
		return
	}

	for _, c := range cx {
		if hasClass("messageBody", c) {
			r.Body = extractAsset(c)
		}

		if hasClass("messageBodySchema", c) {
			r.Schema = extractAsset(c)
		}
	}

	return
}

func extractResponse(child *Element) (r blueprint.Response) {
	r = blueprint.Response{
		StatusCode:     extractInt("attributes.statusCode", child),
		Headers:        extractHeaders(child.Path("attributes.headers")),
		DataStructures: digDataStructures(child),
	}

	cx, err := child.Path("content").Children()
	if err != nil {
		return
	}

	for _, c := range cx {
		if hasClass("messageBody", c) {
			r.Body = extractAsset(c)
		}

		if hasClass("messageBodySchema", c) {
			r.Schema = extractAsset(c)
		}
	}

	return
}

func extractHeaders(child *Element) (hs []blueprint.Header) {
	if child.Path("element").String() == "httpHeaders" {
		contents, err := child.Path("content").Children()
		if err != nil {
			return
		}

		for _, content := range contents {
			h := blueprint.Header{
				Key:   content.Path("content.key.content").String(),
				Value: content.Path("content.value.content").String(),
			}

			hs = append(hs, h)
		}

		return
	}

	return
}

func extractHrefs(child *Element) (h blueprint.Href) {
	href := child.Path("attributes.href")

	if href.Value().IsValid() {
		h.Path = href.String()
	}

	contents, err := child.Path("attributes.hrefVariables.content").Children()
	if err != nil {
		return
	}

	for _, content := range contents {
		v := &blueprint.Parameter{
			Required:    isContains("attributes.typeAttributes", "required", content),
			Key:         content.Path("content.key.content").String(),
			Value:       content.Path("content.value.content").String(),
			Kind:        content.Path("content.value.element").String(),
			Description: content.Path("meta.description").String(),
		}

		h.Parameters = append(h.Parameters, *v)
	}

	return
}

func extractAnnotation(child *Element) (a blueprint.Annotation) {
	if child.Path("element").String() == "annotation" {
		return blueprint.Annotation{
			Description: child.Path("content").String(),
			Classes:     extractSliceString("meta.classes", child),
			Code:        extractInt("attributes.code", child),
			SourceMaps:  digSourceMaps(child.Path("attributes.sourceMap")),
		}
	}

	return
}

func digSourceMaps(el *Element) (ms []blueprint.SourceMap) {
	children, err := el.Children()
	if err != nil {
		return

	}

	for _, child := range children {
		cx := child.Path("content").Value()

		if cx.IsValid() && cx.Kind() == reflect.Slice {
			for i := 0; i < cx.Len(); i++ {
				ns := [2]int{}

				for j, n := range cx.Index(i).Interface().([]interface{}) {
					ns[j] = int(n.(float64))
				}

				m := blueprint.SourceMap{Row: ns[0], Col: ns[1]}
				ms = append(ms, m)
			}

		}
	}

	return
}

func extractInt(key string, child *Element) int {
	var err error

	s := child.Path(key).String()
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}

	return n
}

func extractAsset(child *Element) (a blueprint.Asset) {
	if child.Path("element").String() == "asset" {
		return blueprint.Asset{
			ContentType: child.Path("attributes.contentType").String(),
			Body:        child.Path("content").String(),
		}
	}

	return
}

func extractSliceString(key string, child *Element) []string {
	x := []string{}
	v := child.Path(key).Value()

	if !v.IsValid() {
		return x
	}

	for i := 0; i < v.Len(); i++ {
		x = append(x, v.Index(i).Interface().(string))
	}

	return x
}

func hasClass(s string, child *Element) bool {
	return isContains("meta.classes", s, child)
}

func isContains(key, s string, child *Element) bool {
	v := child.Path(key).Value()

	if !v.IsValid() {
		return false
	}

	for i := 0; i < v.Len(); i++ {
		if s == v.Index(i).Interface().(string) {
			return true
		}
	}

	return false
}

func filterContentByElement(s string, el *Element) (xs []*Element) {
	children, err := el.Path("content").Children()
	if err != nil {
		return
	}

	for _, child := range children {
		if child.Path("element").String() == s {
			xs = append(xs, child)
		}
	}

	return
}

func filterContentByClass(s string, el *Element) (xs []*Element) {
	children, err := el.Path("content").Children()
	if err != nil {
		return
	}

	for _, child := range children {
		if hasClass(s, child) {
			xs = append(xs, child)
		}
	}

	return
}
