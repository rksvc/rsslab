package server

import (
	"io"
	"net/url"
	"rsslab/utils"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func sanitize(baseURL string, input string) string {
	doc, err := html.Parse(io.MultiReader(
		strings.NewReader("<body>"),
		strings.NewReader(input),
		strings.NewReader("</body>"),
	))
	if err != nil {
		return ""
	}

	var body *html.Node
	for root := range doc.ChildNodes() {
		if root.Type == html.ElementNode && root.DataAtom == atom.Html {
			for node := range root.ChildNodes() {
				if node.Type == html.ElementNode && node.DataAtom == atom.Body {
					body = node
				}
			}
		}
	}
	if body == nil {
		return ""
	}

	sanitizeChildren(body, baseURL)

	var b strings.Builder
	for node := range body.ChildNodes() {
		if err := html.Render(&b, node); err != nil {
			return ""
		}
	}
	return b.String()
}

func sanitizeChildren(parent *html.Node, baseURL string) {
	for node := parent.FirstChild; node != nil; {
		next := node.NextSibling
		switch node.Type {
		case html.ElementNode:
			sanitizeElement(parent, node, baseURL)
		case html.CommentNode, html.DoctypeNode:
			parent.RemoveChild(node)
		}
		node = next
	}
}

func sanitizeElement(parent, node *html.Node, baseURL string) {
	switch node.DataAtom {
	case atom.Noscript, atom.Script, atom.Style:
		parent.RemoveChild(node)
		return
	case atom.Tt:
		node.DataAtom = atom.Code
		node.Data = node.DataAtom.String()
	}

	// Children are sanitized first so that descendants of elements that end
	// up being removed or promoted have already been cleaned up.
	sanitizeChildren(node, baseURL)

	if !isValidTag(node.DataAtom, node.Data) {
		promoteChildren(parent, node)
		return
	}

	if !sanitizeAttrs(baseURL, node) {
		if node.DataAtom == atom.Iframe {
			// A blocked iframe should not have its inner content rendered.
			parent.RemoveChild(node)
			return
		}
		promoteChildren(parent, node)
		return
	}

	if isKnownVideoIframe(node) {
		wrapper := &html.Node{
			Type: html.ElementNode,
			Data: "div",
			Attr: []html.Attribute{{Key: "class", Val: "video-wrapper"}},
		}
		next := node.NextSibling
		parent.RemoveChild(node)
		parent.InsertBefore(wrapper, next)
		wrapper.AppendChild(node)
	}

	if node.DataAtom == atom.Iframe {
		// An iframe element never has fallback content.
		for node.FirstChild != nil {
			node.RemoveChild(node.FirstChild)
		}
	}
}

func promoteChildren(parent, node *html.Node) {
	for node.FirstChild != nil {
		child := node.FirstChild
		node.RemoveChild(child)
		parent.InsertBefore(child, node)
	}
	parent.RemoveChild(node)
}

var requiredAttrs = map[atom.Atom][]string{
	atom.A:      {"href"},
	atom.Iframe: {"src"},
	atom.Img:    {"src"},
	atom.Source: {"src", "srcset"},
}

func sanitizeAttrs(baseURL string, node *html.Node) bool {
	requiredAttrs := requiredAttrs[node.DataAtom]
	ok := requiredAttrs == nil

	var attrs []html.Attribute
	for _, attr := range node.Attr {
		key := strings.ToLower(attr.Key)
		if !isValidAttr(node.DataAtom, node.Data, key) {
			continue
		}

		val := attr.Val
		if (node.DataAtom == atom.Img || node.DataAtom == atom.Source) && key == "srcset" {
			val = sanitizeSrcsetAttr(baseURL, val)
		} else if isExternalResourceAttr(key) {
			if node.DataAtom == atom.Iframe {
				if !isSafeIframeSource(baseURL, val) {
					continue
				}
			} else if !(node.DataAtom == atom.Img && key == "src" && isValidDataAttr(val)) {
				val = utils.AbsoluteUrl(val, baseURL)
				if val == "" || !hasValidURIScheme(val) || isBlockedResource(val) {
					continue
				}
			}
		}

		attrs = append(attrs, html.Attribute{Key: key, Val: val})
		ok = ok || slices.Contains(requiredAttrs, key)
	}
	if !ok {
		return false
	}

	switch node.DataAtom {
	case atom.A:
		attrs = append(attrs,
			html.Attribute{Key: "rel", Val: "noopener noreferrer"},
			html.Attribute{Key: "target", Val: "_blank"},
			html.Attribute{Key: "referrerpolicy", Val: "no-referrer"},
		)
	case atom.Video, atom.Audio:
		attrs = append(attrs, html.Attribute{Key: "controls"})
	case atom.Iframe:
		attrs = append(attrs,
			html.Attribute{Key: "sandbox", Val: "allow-scripts allow-same-origin allow-popups"},
			html.Attribute{Key: "loading", Val: "lazy"},
		)
	case atom.Img:
		attrs = append(attrs,
			html.Attribute{Key: "loading", Val: "lazy"},
			html.Attribute{Key: "referrerpolicy", Val: "no-referrer"},
		)
	}

	node.Attr = attrs
	return true
}

// taken from: https://github.com/cure53/DOMPurify/blob/e1c19cf6/src/tags.js
var allowedTags = map[atom.Atom]struct{}{
	atom.A:          {},
	atom.Abbr:       {},
	atom.Acronym:    {},
	atom.Address:    {},
	atom.Area:       {},
	atom.Article:    {},
	atom.Aside:      {},
	atom.Audio:      {},
	atom.B:          {},
	atom.Bdi:        {},
	atom.Bdo:        {},
	atom.Big:        {},
	atom.Blink:      {},
	atom.Blockquote: {},
	atom.Body:       {},
	atom.Br:         {},
	atom.Canvas:     {},
	atom.Caption:    {},
	atom.Center:     {},
	atom.Cite:       {},
	atom.Code:       {},
	atom.Col:        {},
	atom.Colgroup:   {},
	atom.Content:    {},
	atom.Data:       {},
	atom.Datalist:   {},
	atom.Dd:         {},
	atom.Del:        {},
	atom.Details:    {},
	atom.Dfn:        {},
	atom.Dialog:     {},
	atom.Dir:        {},
	atom.Div:        {},
	atom.Dl:         {},
	atom.Dt:         {},
	atom.Em:         {},
	atom.Fieldset:   {},
	atom.Figcaption: {},
	atom.Figure:     {},
	atom.Font:       {},
	atom.Footer:     {},
	atom.Form:       {},
	atom.H1:         {},
	atom.H2:         {},
	atom.H3:         {},
	atom.H4:         {},
	atom.H5:         {},
	atom.H6:         {},
	atom.Head:       {},
	atom.Header:     {},
	atom.Hgroup:     {},
	atom.Hr:         {},
	atom.Html:       {},
	atom.I:          {},
	atom.Iframe:     {},
	atom.Img:        {},
	atom.Ins:        {},
	atom.Kbd:        {},
	atom.Label:      {},
	atom.Legend:     {},
	atom.Li:         {},
	atom.Main:       {},
	atom.Map:        {},
	atom.Mark:       {},
	atom.Marquee:    {},
	atom.Menu:       {},
	atom.Menuitem:   {},
	atom.Meter:      {},
	atom.Nav:        {},
	atom.Nobr:       {},
	atom.Ol:         {},
	atom.Optgroup:   {},
	atom.Option:     {},
	atom.Output:     {},
	atom.P:          {},
	atom.Picture:    {},
	atom.Pre:        {},
	atom.Progress:   {},
	atom.Q:          {},
	atom.Rp:         {},
	atom.Rt:         {},
	atom.Ruby:       {},
	atom.S:          {},
	atom.Samp:       {},
	atom.Section:    {},
	atom.Select:     {},
	atom.Small:      {},
	atom.Source:     {},
	atom.Spacer:     {},
	atom.Span:       {},
	atom.Strike:     {},
	atom.Strong:     {},
	atom.Sub:        {},
	atom.Summary:    {},
	atom.Sup:        {},
	atom.Table:      {},
	atom.Tbody:      {},
	atom.Td:         {},
	atom.Template:   {},
	atom.Textarea:   {},
	atom.Tfoot:      {},
	atom.Th:         {},
	atom.Thead:      {},
	atom.Time:       {},
	atom.Tr:         {},
	atom.Track:      {},
	atom.Tt:         {},
	atom.U:          {},
	atom.Ul:         {},
	atom.Var:        {},
	atom.Video:      {},
	atom.Wbr:        {},
}

var allowedSvgTags = map[string]struct{}{
	"svg":              {},
	"a":                {},
	"altglyph":         {},
	"altglyphdef":      {},
	"altglyphitem":     {},
	"animatecolor":     {},
	"animatemotion":    {},
	"animatetransform": {},
	"circle":           {},
	"clippath":         {},
	"defs":             {},
	"desc":             {},
	"ellipse":          {},
	"filter":           {},
	"font":             {},
	"g":                {},
	"glyph":            {},
	"glyphref":         {},
	"hkern":            {},
	"image":            {},
	"line":             {},
	"lineargradient":   {},
	"marker":           {},
	"mask":             {},
	"metadata":         {},
	"mpath":            {},
	"path":             {},
	"pattern":          {},
	"polygon":          {},
	"polyline":         {},
	"radialgradient":   {},
	"rect":             {},
	"stop":             {},
	"switch":           {},
	"symbol":           {},
	"text":             {},
	"textpath":         {},
	"title":            {},
	"tref":             {},
	"tspan":            {},
	"view":             {},
	"vkern":            {},
}

var allowedSvgFilters = map[string]struct{}{
	"feBlend":             {},
	"feColorMatrix":       {},
	"feComponentTransfer": {},
	"feComposite":         {},
	"feConvolveMatrix":    {},
	"feDiffuseLighting":   {},
	"feDisplacementMap":   {},
	"feDistantLight":      {},
	"feFlood":             {},
	"feFuncA":             {},
	"feFuncB":             {},
	"feFuncG":             {},
	"feFuncR":             {},
	"feGaussianBlur":      {},
	"feMerge":             {},
	"feMergeNode":         {},
	"feMorphology":        {},
	"feOffset":            {},
	"fePointLight":        {},
	"feSpecularLighting":  {},
	"feSpotLight":         {},
	"feTile":              {},
	"feTurbulence":        {},
}

func isValidTag(tagAtom atom.Atom, tagName string) bool {
	if _, ok := allowedTags[tagAtom]; ok {
		return true
	}
	if _, ok := allowedSvgTags[tagName]; ok {
		return true
	}
	_, ok := allowedSvgFilters[tagName]
	return ok
}

var allowedAttrs = map[atom.Atom]map[string]struct{}{
	atom.Img: {
		"alt":    {},
		"title":  {},
		"src":    {},
		"srcset": {},
		"sizes":  {},
	},
	atom.Audio: {
		"src": {},
	},
	atom.Video: {
		"poster": {},
		"height": {},
		"width":  {},
		"src":    {},
	},
	atom.Source: {
		"src":    {},
		"type":   {},
		"srcset": {},
		"sizes":  {},
		"media":  {},
	},
	atom.Td: {
		"rowspan": {},
		"colspan": {},
	},
	atom.Th: {
		"rowspan": {},
		"colspan": {},
	},
	atom.Q: {
		"cite": {},
	},
	atom.A: {
		"href":  {},
		"title": {},
	},
	atom.Time: {
		"datetime": {},
	},
	atom.Abbr: {
		"title": {},
	},
	atom.Acronym: {
		"title": {},
	},
	atom.Iframe: {
		"width":           {},
		"height":          {},
		"frameborder":     {},
		"src":             {},
		"allowfullscreen": {},
	},
	atom.Progress: {
		"value": {},
		"max":   {},
	},
}

var allowedSvgAttrs = map[string]struct{}{
	"accent-height":               {},
	"accumulate":                  {},
	"additive":                    {},
	"alignment-baseline":          {},
	"ascent":                      {},
	"attributename":               {},
	"attributetype":               {},
	"azimuth":                     {},
	"basefrequency":               {},
	"baseline-shift":              {},
	"begin":                       {},
	"bias":                        {},
	"by":                          {},
	"class":                       {},
	"clip":                        {},
	"clippathunits":               {},
	"clip-path":                   {},
	"clip-rule":                   {},
	"color":                       {},
	"color-interpolation":         {},
	"color-interpolation-filters": {},
	"color-profile":               {},
	"color-rendering":             {},
	"cx":                          {},
	"cy":                          {},
	"d":                           {},
	"dx":                          {},
	"dy":                          {},
	"diffuseconstant":             {},
	"direction":                   {},
	"display":                     {},
	"divisor":                     {},
	"dur":                         {},
	"edgemode":                    {},
	"elevation":                   {},
	"end":                         {},
	"fill":                        {},
	"fill-opacity":                {},
	"fill-rule":                   {},
	"filter":                      {},
	"filterunits":                 {},
	"flood-color":                 {},
	"flood-opacity":               {},
	"font-family":                 {},
	"font-size":                   {},
	"font-size-adjust":            {},
	"font-stretch":                {},
	"font-style":                  {},
	"font-variant":                {},
	"font-weight":                 {},
	"fx":                          {},
	"fy":                          {},
	"g1":                          {},
	"g2":                          {},
	"glyph-name":                  {},
	"glyphref":                    {},
	"gradientunits":               {},
	"gradienttransform":           {},
	"height":                      {},
	"href":                        {},
	"id":                          {},
	"image-rendering":             {},
	"in":                          {},
	"in2":                         {},
	"k":                           {},
	"k1":                          {},
	"k2":                          {},
	"k3":                          {},
	"k4":                          {},
	"kerning":                     {},
	"keypoints":                   {},
	"keysplines":                  {},
	"keytimes":                    {},
	"lang":                        {},
	"lengthadjust":                {},
	"letter-spacing":              {},
	"kernelmatrix":                {},
	"kernelunitlength":            {},
	"lighting-color":              {},
	"local":                       {},
	"marker-end":                  {},
	"marker-mid":                  {},
	"marker-start":                {},
	"markerheight":                {},
	"markerunits":                 {},
	"markerwidth":                 {},
	"maskcontentunits":            {},
	"maskunits":                   {},
	"max":                         {},
	"mask":                        {},
	"media":                       {},
	"method":                      {},
	"mode":                        {},
	"min":                         {},
	"name":                        {},
	"numoctaves":                  {},
	"offset":                      {},
	"operator":                    {},
	"opacity":                     {},
	"order":                       {},
	"orient":                      {},
	"orientation":                 {},
	"origin":                      {},
	"overflow":                    {},
	"paint-order":                 {},
	"path":                        {},
	"pathlength":                  {},
	"patterncontentunits":         {},
	"patterntransform":            {},
	"patternunits":                {},
	"points":                      {},
	"preservealpha":               {},
	"preserveaspectratio":         {},
	"primitiveunits":              {},
	"r":                           {},
	"rx":                          {},
	"ry":                          {},
	"radius":                      {},
	"refx":                        {},
	"refy":                        {},
	"repeatcount":                 {},
	"repeatdur":                   {},
	"restart":                     {},
	"result":                      {},
	"rotate":                      {},
	"scale":                       {},
	"seed":                        {},
	"shape-rendering":             {},
	"specularconstant":            {},
	"specularexponent":            {},
	"spreadmethod":                {},
	"startoffset":                 {},
	"stddeviation":                {},
	"stitchtiles":                 {},
	"stop-color":                  {},
	"stop-opacity":                {},
	"stroke-dasharray":            {},
	"stroke-dashoffset":           {},
	"stroke-linecap":              {},
	"stroke-linejoin":             {},
	"stroke-miterlimit":           {},
	"stroke-opacity":              {},
	"stroke":                      {},
	"stroke-width":                {},
	"surfacescale":                {},
	"systemlanguage":              {},
	"tabindex":                    {},
	"targetx":                     {},
	"targety":                     {},
	"transform":                   {},
	"text-anchor":                 {},
	"text-decoration":             {},
	"text-rendering":              {},
	"textlength":                  {},
	"type":                        {},
	"u1":                          {},
	"u2":                          {},
	"unicode":                     {},
	"values":                      {},
	"viewbox":                     {},
	"visibility":                  {},
	"version":                     {},
	"vert-adv-y":                  {},
	"vert-origin-x":               {},
	"vert-origin-y":               {},
	"width":                       {},
	"word-spacing":                {},
	"wrap":                        {},
	"writing-mode":                {},
	"xchannelselector":            {},
	"ychannelselector":            {},
	"x":                           {},
	"x1":                          {},
	"x2":                          {},
	"xmlns":                       {},
	"y":                           {},
	"y1":                          {},
	"y2":                          {},
	"z":                           {},
	"zoomandpan":                  {},
}

func isValidAttr(tagAtom atom.Atom, tagName, attrName string) bool {
	if attrs, ok := allowedAttrs[tagAtom]; ok {
		_, ok = attrs[attrName]
		return ok
	}
	if _, ok := allowedSvgTags[tagName]; ok {
		_, ok = allowedSvgAttrs[attrName]
		return ok
	}
	return false
}

func isExternalResourceAttr(attr string) bool {
	switch attr {
	case "src", "href", "poster", "cite":
		return true
	default:
		return false
	}
}

var allowedURISchemes = map[string]struct{}{
	"http":   {},
	"https":  {},
	"ftp":    {},
	"ftps":   {},
	"tel":    {},
	"mailto": {},
	"callto": {},
	"cid":    {},
	"xmpp":   {},
}

// See https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml
func hasValidURIScheme(src string) bool {
	before, _, ok := strings.Cut(src, ":")
	if !ok {
		return false
	}
	_, ok = allowedURISchemes[before]
	return ok
}

var blockedResources = []string{
	"feedsportal.com",
	"api.flattr.com",
	"stats.wordpress.com",
	"plus.google.com/share",
	"twitter.com/share",
	"feeds.feedburner.com",
}

func isBlockedResource(src string) bool {
	return slices.ContainsFunc(blockedResources, func(resource string) bool {
		return strings.Contains(src, resource)
	})
}

var safeDomains = map[string]struct{}{
	"bandcamp.com":             {},
	"cdn.embedly.com":          {},
	"invidio.us":               {},
	"player.bilibili.com":      {},
	"player.vimeo.com":         {},
	"soundcloud.com":           {},
	"vk.com":                   {},
	"w.soundcloud.com":         {},
	"www.dailymotion.com":      {},
	"www.youtube-nocookie.com": {},
	"www.youtube.com":          {},
}

func isSafeIframeSource(baseURL, src string) bool {
	domain := urlDomain(src)
	// allow iframe from same origin
	if urlDomain(baseURL) == domain {
		return true
	}
	_, ok := safeDomains[domain]
	return ok
}

// One or more strings separated by commas, indicating possible image sources for the user agent to use.
//
// Each string is composed of a URL to an image and, optionally, whitespace followed by one of:
//   - A width descriptor (a positive integer directly followed by w). The width descriptor is divided
//     by the source size given in the sizes attribute to calculate the effective pixel density.
//   - A pixel density descriptor (a positive floating point number directly followed by x).
func sanitizeSrcsetAttr(baseURL, srcset string) string {
	var sanitizedSrcs []string
	for rawSrc := range strings.SplitSeq(srcset, ", ") {
		parts := strings.SplitN(strings.TrimSpace(rawSrc), " ", 3)
		if len(parts) > 0 {
			sanitizedSrc := parts[0]
			if !strings.HasPrefix(parts[0], "data:") {
				sanitizedSrc = utils.AbsoluteUrl(sanitizedSrc, baseURL)
				if sanitizedSrc == "" {
					continue
				}
			}

			if len(parts) == 2 && isValidWidthOrDensityDescriptor(parts[1]) {
				sanitizedSrc += " " + parts[1]
			}

			sanitizedSrcs = append(sanitizedSrcs, sanitizedSrc)
		}
	}
	return strings.Join(sanitizedSrcs, ", ")
}

func isValidWidthOrDensityDescriptor(value string) bool {
	if value == "" {
		return false
	}

	lastByte := value[len(value)-1]
	if lastByte != 'w' && lastByte != 'x' {
		return false
	}

	_, err := strconv.ParseFloat(value[:len(value)-1], 32)
	return err == nil
}

var dataAttrAllowList = []string{
	"data:image/avif",
	"data:image/apng",
	"data:image/png",
	"data:image/svg",
	"data:image/svg+xml",
	"data:image/jpg",
	"data:image/jpeg",
	"data:image/gif",
	"data:image/webp",
}

func isValidDataAttr(value string) bool {
	return slices.ContainsFunc(dataAttrAllowList, func(prefix string) bool {
		return strings.HasPrefix(value, prefix)
	})
}

var videoWhitelist = map[string]struct{}{
	"player.bilibili.com":      {},
	"player.vimeo.com":         {},
	"www.dailymotion.com":      {},
	"www.youtube-nocookie.com": {},
	"www.youtube.com":          {},
}

func isKnownVideoIframe(node *html.Node) bool {
	if node.DataAtom == atom.Iframe {
		for _, attr := range node.Attr {
			if attr.Key == "src" {
				_, ok := videoWhitelist[urlDomain(attr.Val)]
				return ok
			}
		}
	}
	return false
}

func urlDomain(href string) string {
	if url, err := url.Parse(href); err == nil {
		return url.Host
	}
	return ""
}
