package utils

import (
	"io"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func Sanitize(baseURL string, input string) string {
	var b strings.Builder
	var tagStack []string
	var parentTag atom.Atom
	blacklistedTagDepth := 0

	tokenizer := html.NewTokenizer(strings.NewReader(input))
	for {
		if tokenizer.Next() == html.ErrorToken {
			if tokenizer.Err() == io.EOF {
				return b.String()
			}
			return ""
		}

		token := tokenizer.Token()
		switch token.Type {
		case html.TextToken:
			if blacklistedTagDepth > 0 {
				continue
			}
			// An iframe element never has fallback content.
			// See https://www.w3.org/TR/2010/WD-html5-20101019/the-iframe-element.html#the-iframe-element
			if parentTag == atom.Iframe {
				continue
			}
			b.WriteString(html.EscapeString(token.Data))

		case html.StartTagToken:
			parentTag = token.DataAtom
			if isValidTag(token.DataAtom, token.Data) {
				attrs, ok := sanitizeAttrs(baseURL, token)
				if ok {
					wrap := isKnownVideoIframe(token)
					if wrap {
						b.WriteString(`<div class="video-wrapper">`)
					}

					b.WriteByte('<')
					b.WriteString(token.Data)
					b.WriteString(attrs)
					b.WriteByte('>')

					if token.DataAtom == atom.Iframe {
						// autoclose iframes
						b.WriteString("</iframe>")
						if wrap {
							b.WriteString("</div>")
						}
					} else {
						tagStack = append(tagStack, token.Data)
					}
				}
			} else if isBlockedTag(token.DataAtom) {
				blacklistedTagDepth++
			}

		case html.EndTagToken:
			// iframes are autoclosed, see above
			if token.DataAtom == atom.Iframe {
				continue
			}
			if isValidTag(token.DataAtom, token.Data) && slices.Contains(tagStack, token.Data) {
				b.WriteString("</")
				b.WriteString(token.Data)
				b.WriteByte('>')
			} else if isBlockedTag(token.DataAtom) {
				blacklistedTagDepth--
			}

		case html.SelfClosingTagToken:
			if isValidTag(token.DataAtom, token.Data) {
				attrs, ok := sanitizeAttrs(baseURL, token)
				if ok {
					b.WriteByte('<')
					b.WriteString(token.Data)
					b.WriteString(attrs)
					b.WriteString("/>")
				}
			}
		}
	}
}

func sanitizeAttrs(baseURL string, token html.Token) (string, bool) {
	var requiredAttrs []string
	switch token.DataAtom {
	case atom.A:
		requiredAttrs = []string{"href"}
	case atom.Iframe:
		requiredAttrs = []string{"src"}
	case atom.Img:
		requiredAttrs = []string{"src"}
	case atom.Source:
		requiredAttrs = []string{"src", "srcset"}
	}
	ok := requiredAttrs == nil

	var b strings.Builder
	for _, attr := range token.Attr {
		if !isValidAttr(token.DataAtom, token.Data, attr.Key) {
			continue
		}

		val := attr.Val
		if (token.DataAtom == atom.Img || token.DataAtom == atom.Source) && attr.Key == "srcset" {
			val = sanitizeSrcsetAttr(baseURL, val)
		} else if isExternalResourceAttr(attr.Key) {
			if token.DataAtom == atom.Iframe {
				if !isSafeIframeSource(baseURL, attr.Val) {
					continue
				}
			} else if !(token.DataAtom == atom.Img && attr.Key == "src" && isValidDataAttr(attr.Val)) {
				val = AbsoluteUrl(val, baseURL)
				if val == "" || !hasValidURIScheme(val) || isBlockedResource(val) {
					continue
				}
			}
		}

		b.WriteByte(' ')
		b.WriteString(attr.Key)
		b.WriteByte('=')
		b.WriteByte('"')
		b.WriteString(html.EscapeString(val))
		b.WriteByte('"')
		ok = ok || slices.Contains(requiredAttrs, attr.Key)
	}
	if !ok {
		return "", false
	}

	switch token.DataAtom {
	case atom.A:
		b.WriteString(` rel="noopener noreferrer" target="_blank" referrerpolicy="no-referrer"`)
	case atom.Video, atom.Audio:
		b.WriteString(` controls`)
	case atom.Iframe:
		b.WriteString(` sandbox="allow-scripts allow-same-origin allow-popups" loading="lazy"`)
	case atom.Img:
		b.WriteString(` loading="lazy" referrerpolicy="no-referrer"`)
	}

	return b.String(), true
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
	atom.Button:     {},
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
	atom.Input:      {},
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
	i := strings.IndexByte(src, ':')
	if i == -1 {
		return false
	}
	_, ok := allowedURISchemes[src[:i]]
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
	for _, resource := range blockedResources {
		if strings.Contains(src, resource) {
			return true
		}
	}
	return false
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
	domain := UrlDomain(src)
	// allow iframe from same origin
	if UrlDomain(baseURL) == domain {
		return true
	}
	_, ok := safeDomains[domain]
	return ok
}

func isBlockedTag(tagAtom atom.Atom) bool {
	switch tagAtom {
	case atom.Noscript, atom.Script, atom.Style:
		return true
	default:
		return false
	}
}

// One or more strings separated by commas, indicating possible image sources for the user agent to use.
//
// Each string is composed of a URL to an image and, optionally, whitespace followed by one of:
//   - A width descriptor (a positive integer directly followed by w). The width descriptor is divided
//     by the source size given in the sizes attribute to calculate the effective pixel density.
//   - A pixel density descriptor (a positive floating point number directly followed by x).
func sanitizeSrcsetAttr(baseURL, srcset string) string {
	var sanitizedSrcs []string
	for _, rawSrc := range strings.Split(srcset, ", ") {
		parts := strings.SplitN(strings.TrimSpace(rawSrc), " ", 3)
		if len(parts) > 0 {
			sanitizedSrc := parts[0]
			if !strings.HasPrefix(parts[0], "data:") {
				sanitizedSrc = AbsoluteUrl(sanitizedSrc, baseURL)
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

func isValidDataAttr(attr string) bool {
	for _, prefix := range dataAttrAllowList {
		if strings.HasPrefix(attr, prefix) {
			return true
		}
	}
	return false
}

var videoWhitelist = map[string]struct{}{
	"player.bilibili.com":      {},
	"player.vimeo.com":         {},
	"www.dailymotion.com":      {},
	"www.youtube-nocookie.com": {},
	"www.youtube.com":          {},
}

func isKnownVideoIframe(token html.Token) bool {
	if token.DataAtom == atom.Iframe {
		for _, attr := range token.Attr {
			if attr.Key == "src" {
				_, ok := videoWhitelist[UrlDomain(attr.Val)]
				return ok
			}
		}
	}
	return false
}
